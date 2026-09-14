package refreshimpl

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	artifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/discovery"
	rootimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/root"
	sourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/source"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Service struct {
	sources      sourceimpl.Runtime
	artifacts    ArtifactReader
	states       RefreshStateReader
	discovery    *discovery.Engine
	synchronizer *artifactimpl.Synchronizer
	publisher    Publisher
	clock        clockutil.Clock
	policy       root.RootPolicy
}

func NewService(
	sources sourceimpl.Runtime,
	artifacts ArtifactReader,
	states RefreshStateReader,
	discoveryEngine *discovery.Engine,
	synchronizer *artifactimpl.Synchronizer,
	publisher Publisher,
	timeClock clockutil.Clock,
	policy root.RootPolicy,
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
			basespec.ErrInvalid,
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
	rootID root.RootID,
) (refresh.RefreshRootResult, error) {
	if s == nil {
		return refresh.RefreshRootResult{}, basespec.ErrClosed
	}
	if ctx == nil {
		return refresh.RefreshRootResult{}, fmt.Errorf(
			"%w: Root refresh context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return refresh.RefreshRootResult{}, err
	}
	if err := rootID.Validate(); err != nil {
		return refresh.RefreshRootResult{}, err
	}
	if err := rootimpl.RequireMutableRoot(
		ctx,
		s.policy,
		rootID,
	); err != nil {
		return refresh.RefreshRootResult{}, err
	}

	values, err := s.sources.List(ctx, rootID)
	if err != nil {
		return refresh.RefreshRootResult{}, err
	}
	sort.Slice(values, func(left, right int) bool {
		return values[left].ID < values[right].ID
	})

	result := refresh.RefreshRootResult{
		RootID:  rootID,
		Sources: make([]refresh.RefreshSourceResult, 0),
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
			return refresh.RefreshRootResult{}, err
		}
		result.Sources = append(result.Sources, refreshed)
	}
	if err := result.Validate(); err != nil {
		return refresh.RefreshRootResult{}, err
	}
	return result.Clone(), nil
}

func (s *Service) RefreshSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) (refresh.RefreshSourceResult, error) {
	if s == nil {
		return refresh.RefreshSourceResult{}, basespec.ErrClosed
	}
	if ctx == nil {
		return refresh.RefreshSourceResult{}, fmt.Errorf(
			"%w: Source refresh context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return refresh.RefreshSourceResult{}, err
	}
	if err := rootID.Validate(); err != nil {
		return refresh.RefreshSourceResult{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return refresh.RefreshSourceResult{}, err
	}
	if err := rootimpl.RequireMutableRoot(
		ctx,
		s.policy,
		rootID,
	); err != nil {
		return refresh.RefreshSourceResult{}, err
	}

	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return refresh.RefreshSourceResult{}, err
	}
	if !value.Enabled {
		return refresh.RefreshSourceResult{}, fmt.Errorf(
			"%w: Source %q is disabled",
			basespec.ErrConflict,
			sourceID,
		)
	}
	if value.Discovery.Empty() {
		return refresh.RefreshSourceResult{}, fmt.Errorf(
			"%w: Source %q has no declaration discovery configuration",
			basespec.ErrRefreshRequired,
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
	case errors.Is(stateErr, basespec.ErrRefreshStateNotFound):
	default:
		return refresh.RefreshSourceResult{}, stateErr
	}

	snapshot, err := s.sources.Open(ctx, value)
	if err != nil {
		return refresh.RefreshSourceResult{}, err
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
		return refresh.RefreshSourceResult{}, err
	}
	existing, err := s.artifacts.ListBySource(
		ctx,
		rootID,
		sourceID,
	)
	if err != nil {
		return refresh.RefreshSourceResult{}, err
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
		return refresh.RefreshSourceResult{}, err
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return refresh.RefreshSourceResult{}, err
	}
	if err := snapshot.Close(); err != nil {
		return refresh.RefreshSourceResult{}, err
	}
	snapshotOpen = false

	discoveryFingerprint, err := value.Discovery.Fingerprint()
	if err != nil {
		return refresh.RefreshSourceResult{}, err
	}
	decoderFingerprint, err := s.discovery.DecoderFingerprint()
	if err != nil {
		return refresh.RefreshSourceResult{}, err
	}
	definitions, err := definitionsFromObservations(
		discovered.Observations,
	)
	if err != nil {
		return refresh.RefreshSourceResult{}, err
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
		return refresh.RefreshSourceResult{}, err
	}
	if err := published.Validate(); err != nil {
		return refresh.RefreshSourceResult{}, fmt.Errorf(
			"%w: refresh publisher returned invalid state: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	result := refresh.RefreshSourceResult{
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
		case artifact.StateMissing:
			result.MissingArtifacts = append(
				result.MissingArtifacts,
				value.ArtifactID,
			)
		case artifact.StateInvalid:
			result.InvalidArtifacts = append(
				result.InvalidArtifacts,
				value.ArtifactID,
			)
		case artifact.StateIncompatible:
			result.IncompatibleArtifacts = append(
				result.IncompatibleArtifacts,
				value.ArtifactID,
			)
		default:
		}
	}
	if err := result.Validate(); err != nil {
		return refresh.RefreshSourceResult{}, err
	}
	return result.Clone(), nil
}

func (s *Service) InspectSource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) (source.RefreshInspection, error) {
	if s == nil {
		return source.RefreshInspection{}, basespec.ErrClosed
	}
	if ctx == nil {
		return source.RefreshInspection{}, fmt.Errorf(
			"%w: Source refresh inspection context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return source.RefreshInspection{}, err
	}
	if err := rootID.Validate(); err != nil {
		return source.RefreshInspection{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return source.RefreshInspection{}, err
	}

	state, err := s.states.GetRefreshState(
		ctx,
		rootID,
		sourceID,
	)
	if err != nil {
		return source.RefreshInspection{}, err
	}
	value, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return source.RefreshInspection{}, err
	}
	discoveryFingerprint, err := value.Discovery.Fingerprint()
	if err != nil {
		return source.RefreshInspection{}, err
	}
	decoderFingerprint, err := s.discovery.DecoderFingerprint()
	if err != nil {
		return source.RefreshInspection{}, err
	}

	snapshot, err := s.sources.Open(ctx, value)
	if err != nil {
		return source.RefreshInspection{}, err
	}
	generation := snapshot.Generation()
	confirmErr := snapshot.Confirm(ctx)
	closeErr := snapshot.Close()
	if err := errors.Join(confirmErr, closeErr); err != nil {
		return source.RefreshInspection{}, err
	}

	result := source.RefreshInspection{
		State:                   state,
		SourceRevisionChanged:   state.SourceRevision != value.Revision,
		DiscoveryChanged:        state.DiscoveryFingerprint != discoveryFingerprint,
		DecoderChanged:          state.DecoderFingerprint != decoderFingerprint,
		SourceGenerationChanged: state.SourceGeneration != generation,
	}
	if err := result.Validate(); err != nil {
		return source.RefreshInspection{}, err
	}
	return result.Clone(), nil
}

func definitionsFromObservations(
	observations []discovery.Observation,
) ([]definition.Definition, error) {
	seen := make(map[cryptoutil.Digest]struct{})
	output := make([]definition.Definition, 0)
	for _, observation := range observations {
		if observation.State != discovery.ObservationValid {
			continue
		}
		if observation.Definition == nil {
			return nil, fmt.Errorf(
				"%w: valid Source observation has no Definition",
				basespec.ErrInvalid,
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
