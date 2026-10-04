package refresh

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	refreshModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Service struct {
	sources      source.Runtime
	artifacts    ArtifactReader
	states       StateReader
	discovery    *ingest.Engine
	synchronizer *artifact.Synchronizer
	publisher    Repository
	clock        clockutil.Clock
	policy       root.Policy
}

func NewService(
	sources source.Runtime,
	artifacts ArtifactReader,
	states StateReader,
	discoveryEngine *ingest.Engine,
	synchronizer *artifact.Synchronizer,
	publisher Repository,
	timeClock clockutil.Clock,
	policy root.Policy,
) (*Service, error) {
	if sources == nil ||
		artifacts == nil ||
		states == nil ||
		discoveryEngine == nil ||
		synchronizer == nil ||
		publisher == nil ||
		timeClock == nil {
		return nil, fmt.Errorf(
			"%w: Source refresh service dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	return &Service{
		sources:      sources,
		artifacts:    artifacts,
		states:       states,
		discovery:    discoveryEngine,
		synchronizer: synchronizer,
		publisher:    publisher,
		clock:        timeClock,
		policy:       policy,
	}, nil
}

func (s *Service) RefreshRoot(
	ctx context.Context,
	rootID rootModel.RootID,
) (refreshModel.RefreshRootResult, error) {
	if s == nil {
		return refreshModel.RefreshRootResult{}, spec.ErrClosed
	}
	if ctx == nil {
		return refreshModel.RefreshRootResult{}, fmt.Errorf(
			"%w: Root refresh context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return refreshModel.RefreshRootResult{}, err
	}
	if err := rootID.Validate(); err != nil {
		return refreshModel.RefreshRootResult{}, err
	}
	if err := root.RequireMutableRoot(
		ctx,
		s.policy,
		rootID,
	); err != nil {
		return refreshModel.RefreshRootResult{}, err
	}

	values, err := s.sources.List(ctx, rootID)
	if err != nil {
		return refreshModel.RefreshRootResult{}, err
	}
	sort.Slice(values, func(left, right int) bool {
		return values[left].ID < values[right].ID
	})

	result := refreshModel.RefreshRootResult{
		RootID:  rootID,
		Sources: make([]refreshModel.RefreshSourceResult, 0),
	}
	for _, value := range values {
		if !value.Enabled || value.Discovery.Empty() {
			continue
		}
		refreshed, err := s.RefreshSource(
			ctx,
			rootID,
			value.ID,
		)
		if err != nil {
			return refreshModel.RefreshRootResult{}, err
		}
		result.Sources = append(result.Sources, refreshed)
		result.Diagnostics = diagnostic.Append(
			result.Diagnostics,
			refreshed.Diagnostics...,
		)
	}
	if err := result.Validate(); err != nil {
		return refreshModel.RefreshRootResult{}, err
	}
	return result.Clone(), nil
}

func (s *Service) RefreshSource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (refreshModel.RefreshSourceResult, error) {
	if s == nil {
		return refreshModel.RefreshSourceResult{}, spec.ErrClosed
	}
	if ctx == nil {
		return refreshModel.RefreshSourceResult{}, fmt.Errorf(
			"%w: Source refresh context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	if err := rootID.Validate(); err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	if err := root.RequireMutableRoot(
		ctx,
		s.policy,
		rootID,
	); err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}

	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	if !value.Enabled {
		return refreshModel.RefreshSourceResult{}, fmt.Errorf(
			"%w: Source %q is disabled",
			spec.ErrConflict,
			sourceID,
		)
	}
	if value.Discovery.Empty() {
		return refreshModel.RefreshSourceResult{}, fmt.Errorf(
			"%w: Source %q has no declaration discovery configuration",
			spec.ErrRefreshRequired,
			sourceID,
		)
	}

	var expectedRefreshRevision uint64
	previous, stateErr := s.states.GetRefreshState(
		ctx,
		rootID,
		sourceID,
	)
	switch {
	case stateErr == nil:
		expectedRefreshRevision = previous.Revision
	case errors.Is(stateErr, spec.ErrRefreshStateNotFound):
	default:
		return refreshModel.RefreshSourceResult{}, stateErr
	}

	snapshot, err := s.sources.Open(ctx, value)
	if err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	snapshotOpen := true
	defer func() {
		if snapshotOpen {
			_ = snapshot.Close()
		}
	}()
	sourceGeneration := snapshot.Generation()

	discovered, err := s.discovery.Discover(
		ctx,
		value,
		snapshot,
	)
	if err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	existing, err := s.artifacts.ListBySource(
		ctx,
		rootID,
		sourceID,
	)
	if err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	synchronization, err := s.synchronizer.Synchronize(
		ctx,
		rootID,
		value,
		discovered.Observations,
		discovered.SeenLocators,
		existing,
	)
	if err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	if err := snapshot.Close(); err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	snapshotOpen = false

	discoveryFingerprint, err := value.Discovery.Fingerprint()
	if err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	decoderFingerprint, err := s.discovery.DecoderFingerprint()
	if err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	definitions, err := definitionsFromObservations(
		discovered.Observations,
	)
	if err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}

	publication := Publication{
		RootID:                  rootID,
		SourceID:                sourceID,
		ExpectedSourceRevision:  value.Revision,
		ExpectedRefreshRevision: expectedRefreshRevision,
		SourceGeneration:        sourceGeneration,
		DiscoveryFingerprint:    discoveryFingerprint,
		DecoderFingerprint:      decoderFingerprint,
		Definitions:             definitions,
		ArtifactCreates:         synchronization.Creates,
		ArtifactUpdates:         synchronization.Updates,
		Diagnostics: diagnostic.Append(
			discovered.Diagnostics,
			synchronization.Diagnostics...,
		),
		RefreshedAt: clockutil.NowUTC(s.clock),
	}
	published, err := s.publisher.Publish(ctx, publication)
	if err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	if err := published.Validate(); err != nil {
		return refreshModel.RefreshSourceResult{}, fmt.Errorf(
			"%w: refresh publisher returned invalid state: %w",
			spec.ErrInvalid,
			err,
		)
	}

	result := refreshModel.RefreshSourceResult{
		State:       published,
		Diagnostics: append([]diagnostic.Diagnostic(nil), publication.Diagnostics...),
		Candidates:  discovered.Candidates,
	}
	for _, value := range synchronization.Creates {
		result.CreatedArtifacts = append(
			result.CreatedArtifacts,
			value.ID,
		)
	}
	for _, value := range synchronization.Updates {
		result.UpdatedArtifacts = append(
			result.UpdatedArtifacts,
			value.ArtifactID,
		)
		switch value.State {
		case artifactModel.StateMissing:
			result.MissingArtifacts = append(
				result.MissingArtifacts,
				value.ArtifactID,
			)
		case artifactModel.StateInvalid:
			result.InvalidArtifacts = append(
				result.InvalidArtifacts,
				value.ArtifactID,
			)
		case artifactModel.StateIncompatible:
			result.IncompatibleArtifacts = append(
				result.IncompatibleArtifacts,
				value.ArtifactID,
			)
		default:
		}
	}
	if err := result.Validate(); err != nil {
		return refreshModel.RefreshSourceResult{}, err
	}
	return result.Clone(), nil
}

func (s *Service) InspectSource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (refreshModel.Inspection, error) {
	if s == nil {
		return refreshModel.Inspection{}, spec.ErrClosed
	}
	if ctx == nil {
		return refreshModel.Inspection{}, fmt.Errorf(
			"%w: Source refresh inspection context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return refreshModel.Inspection{}, err
	}
	if err := rootID.Validate(); err != nil {
		return refreshModel.Inspection{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return refreshModel.Inspection{}, err
	}

	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return refreshModel.Inspection{}, err
	}
	result, err := s.InspectSourceMetadata(ctx, value)
	if err != nil {
		return refreshModel.Inspection{}, err
	}

	// A metadata difference already makes the Source stale. Do not open and
	// fingerprint a potentially large filesystem tree immediately before the
	// caller refreshes it.
	if result.SourceRevisionChanged ||
		result.DiscoveryChanged ||
		result.DecoderChanged {
		return result.Clone(), nil
	}

	return s.inspectSourceGeneration(ctx, value, result)
}

func (s *Service) InspectSourceMetadata(
	ctx context.Context,
	value sourceModel.Source,
) (refreshModel.Inspection, error) {
	if s == nil {
		return refreshModel.Inspection{}, spec.ErrClosed
	}
	if ctx == nil {
		return refreshModel.Inspection{}, spec.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return refreshModel.Inspection{}, err
	}
	state, err := s.states.GetRefreshState(ctx, value.RootID, value.ID)
	if err != nil {
		return refreshModel.Inspection{}, err
	}
	discoveryFingerprint, err := value.Discovery.Fingerprint()
	if err != nil {
		return refreshModel.Inspection{}, err
	}
	decoderFingerprint, err := s.discovery.DecoderFingerprint()
	if err != nil {
		return refreshModel.Inspection{}, err
	}

	return refreshModel.Inspection{
		State:                 state,
		SourceRevisionChanged: state.SourceRevision != value.Revision,
		DiscoveryChanged:      state.DiscoveryFingerprint != discoveryFingerprint,
		DecoderChanged:        state.DecoderFingerprint != decoderFingerprint,
	}, nil
}

func (s *Service) inspectSourceGeneration(
	ctx context.Context,
	value sourceModel.Source,
	result refreshModel.Inspection,
) (refreshModel.Inspection, error) {
	snapshot, err := s.sources.Open(ctx, value)
	if err != nil {
		return refreshModel.Inspection{}, err
	}
	generation := snapshot.Generation()
	confirmErr := snapshot.Confirm(ctx)
	closeErr := snapshot.Close()
	if err := errors.Join(confirmErr, closeErr); err != nil {
		return refreshModel.Inspection{}, err
	}

	result.SourceGenerationChanged = result.State.SourceGeneration != generation
	return result.Clone(), nil
}

func definitionsFromObservations(
	observations []ingest.Observation,
) ([]definitionModel.Definition, error) {
	seen := make(map[cryptoutil.Digest]struct{})
	output := make([]definitionModel.Definition, 0)
	for _, observation := range observations {
		if observation.State != ingest.ObservationValid {
			continue
		}
		if observation.Definition == nil {
			return nil, fmt.Errorf(
				"%w: valid Source observation has no Definition",
				spec.ErrInvalid,
			)
		}
		if _, duplicate := seen[observation.Definition.Digest]; duplicate {
			continue
		}
		seen[observation.Definition.Digest] = struct{}{}
		output = append(
			output,
			observation.Definition.Clone(),
		)
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].Digest < output[right].Digest
	})
	return output, nil
}
