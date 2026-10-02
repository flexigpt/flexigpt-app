package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/idprovider"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	ingestimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/impl"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type typedBinding struct {
	Binding artifact.SourceBinding
	Kind    artifact.ArtifactKind
}

type Synchronizer struct {
	clock clockutil.Clock
	ids   idprovider.Provider
}

func NewSynchronizer(
	timeClock clockutil.Clock,
	ids idprovider.Provider,
) (*Synchronizer, error) {
	if timeClock == nil || ids == nil {
		return nil, fmt.Errorf(
			"%w: Artifact synchronizer dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	return &Synchronizer{
		clock: timeClock,
		ids:   ids,
	}, nil
}

// Synchronize converts generic discovery observations into deterministic
// source-backed Artifact creation and source-state updates.
//
// Valid named Definitions always become Artifacts. No provider adoption,
// pinning, or suppression policy is involved.
func (s *Synchronizer) Synchronize(
	ctx context.Context,
	rootID root.RootID,
	sourceValue source.Source,
	observations []ingestimpl.Observation,
	seenLocators []spec.Locator,
	existing []artifact.Artifact,
) (Synchronization, error) {
	if s == nil || s.clock == nil || s.ids == nil {
		return Synchronization{}, spec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return Synchronization{}, err
	}
	if err := sourceValue.Validate(); err != nil {
		return Synchronization{}, err
	}
	if sourceValue.RootID != rootID {
		return Synchronization{}, fmt.Errorf(
			"%w: Source belongs to another Root",
			spec.ErrInvalid,
		)
	}
	discoverySpec := sourceValue.Discovery.Effective()
	if err := discoverySpec.Validate(); err != nil {
		return Synchronization{}, fmt.Errorf(
			"source discovery: %w",
			err,
		)
	}

	validByTypedBinding := make(
		map[typedBinding]ingestimpl.Observation,
		len(observations),
	)
	validByBinding := make(
		map[artifact.SourceBinding][]ingestimpl.Observation,
	)
	invalidByBinding := make(
		map[artifact.SourceBinding]ingestimpl.Observation,
	)

	for index, observation := range observations {
		if err := observation.Validate(); err != nil {
			return Synchronization{}, fmt.Errorf(
				"source observation %d: %w",
				index,
				err,
			)
		}
		if observation.RootID != rootID ||
			observation.Binding.SourceID != sourceValue.ID {
			return Synchronization{}, fmt.Errorf(
				"%w: Source observation belongs to another Root or Source",
				spec.ErrInvalid,
			)
		}

		switch observation.State {
		case ingestimpl.ObservationValid:
			key := typedBinding{
				Binding: observation.Binding,
				Kind:    observation.Kind,
			}
			if _, duplicate := validByTypedBinding[key]; duplicate {
				return Synchronization{}, fmt.Errorf(
					"%w: duplicate valid typed Source observation",
					spec.ErrInvalid,
				)
			}
			validByTypedBinding[key] = observation.Clone()
			validByBinding[observation.Binding] = append(
				validByBinding[observation.Binding],
				observation.Clone(),
			)

		case ingestimpl.ObservationInvalid:
			if _, duplicate := invalidByBinding[observation.Binding]; duplicate {
				return Synchronization{}, fmt.Errorf(
					"%w: duplicate invalid Source observation",
					spec.ErrInvalid,
				)
			}
			invalidByBinding[observation.Binding] = observation.Clone()
		}
	}

	for binding := range validByBinding {
		sort.Slice(validByBinding[binding], func(left, right int) bool {
			leftValue := validByBinding[binding][left]
			rightValue := validByBinding[binding][right]
			if leftValue.Kind != rightValue.Kind {
				return leftValue.Kind < rightValue.Kind
			}
			return leftValue.LogicalName < rightValue.LogicalName
		})
	}

	seen := make(map[spec.Locator]struct{}, len(seenLocators))
	for _, locator := range seenLocators {
		if err := locator.Validate(false); err != nil {
			return Synchronization{}, err
		}
		seen[locator] = struct{}{}
	}

	existingByTypedBinding := make(
		map[typedBinding]artifact.Artifact,
		len(existing),
	)
	seenIDs := make(map[artifact.ArtifactID]struct{}, len(existing))
	orderedExisting := append([]artifact.Artifact(nil), existing...)
	sort.Slice(orderedExisting, func(left, right int) bool {
		return orderedExisting[left].ID < orderedExisting[right].ID
	})

	for index, current := range orderedExisting {
		if err := current.Validate(); err != nil {
			return Synchronization{}, fmt.Errorf(
				"existing Artifact %d: %w",
				index,
				err,
			)
		}
		if current.RootID != rootID ||
			current.Binding.SourceID != sourceValue.ID {
			return Synchronization{}, fmt.Errorf(
				"%w: existing Artifact belongs to another Root or Source",
				spec.ErrInvalid,
			)
		}
		if _, duplicate := seenIDs[current.ID]; duplicate {
			return Synchronization{}, fmt.Errorf(
				"%w: duplicate existing Artifact ID %q",
				spec.ErrInvalid,
				current.ID,
			)
		}
		seenIDs[current.ID] = struct{}{}

		key := typedBinding{
			Binding: current.Binding,
			Kind:    current.Kind,
		}
		if _, duplicate := existingByTypedBinding[key]; duplicate {
			return Synchronization{}, fmt.Errorf(
				"%w: duplicate source-origin Artifact",
				spec.ErrInvalid,
			)
		}
		existingByTypedBinding[key] = current.Clone()
	}

	result := Synchronization{}
	now := clockutil.NowUTC(s.clock)
	for _, current := range orderedExisting {
		next, changed, err := deriveCurrentArtifact(
			current,
			validByTypedBinding,
			validByBinding,
			invalidByBinding,
			seen,
			discoverySpec,
		)
		if err != nil {
			return Synchronization{}, err
		}
		if !changed {
			continue
		}
		if current.Revision == ^uint64(0) {
			return Synchronization{}, fmt.Errorf(
				"%w: Artifact revision is exhausted",
				spec.ErrInvalid,
			)
		}

		next.Revision++
		next.ModifiedAt = clockutil.Advance(
			now,
			current.ModifiedAt,
		)
		if err := next.Validate(); err != nil {
			return Synchronization{}, err
		}
		result.Updates = append(result.Updates, SourceStateUpdate{
			ArtifactID:          next.ID,
			RootID:              next.RootID,
			Binding:             next.Binding,
			LogicalName:         next.LogicalName,
			LogicalVersion:      next.LogicalVersion,
			ResolvedDefinition:  cryptoutil.CloneDigest(next.ResolvedDefinition),
			SourceContentDigest: cryptoutil.CloneDigest(next.SourceContentDigest),
			State:               next.State,
			Diagnostics:         diagnostic.Clone(next.Diagnostics),
			Revision:            next.Revision,
			ModifiedAt:          next.ModifiedAt,
			ExpectedRevision:    current.Revision,
		})
	}

	orderedObservations := make(
		[]ingestimpl.Observation,
		0,
		len(validByTypedBinding),
	)
	for _, observation := range validByTypedBinding {
		orderedObservations = append(
			orderedObservations,
			observation.Clone(),
		)
	}
	sort.Slice(orderedObservations, func(left, right int) bool {
		leftValue := orderedObservations[left]
		rightValue := orderedObservations[right]
		if leftValue.Binding.Locator != rightValue.Binding.Locator {
			return leftValue.Binding.Locator <
				rightValue.Binding.Locator
		}
		if leftValue.Binding.SubresourceLocator !=
			rightValue.Binding.SubresourceLocator {
			return leftValue.Binding.SubresourceLocator <
				rightValue.Binding.SubresourceLocator
		}
		return leftValue.Kind < rightValue.Kind
	})

	for _, observation := range orderedObservations {
		key := typedBinding{
			Binding: observation.Binding,
			Kind:    observation.Kind,
		}
		if _, found := existingByTypedBinding[key]; found {
			continue
		}
		if observation.Definition == nil ||
			observation.SourceContentDigest == nil {
			return Synchronization{}, fmt.Errorf(
				"%w: valid Source observation is incomplete",
				spec.ErrInvalid,
			)
		}

		id, err := s.ids.NewArtifactID(ctx)
		if err != nil {
			return Synchronization{}, err
		}
		if err := id.Validate(); err != nil {
			return Synchronization{}, err
		}
		if _, duplicate := seenIDs[id]; duplicate {
			return Synchronization{}, fmt.Errorf(
				"%w: Artifact ID provider reused %q",
				spec.ErrConflict,
				id,
			)
		}
		seenIDs[id] = struct{}{}

		displayName := observation.Definition.DisplayName
		if displayName == "" {
			displayName = string(observation.LogicalName)
		}
		resolved := observation.Definition.Digest
		created := artifact.Artifact{
			ID:      id,
			RootID:  rootID,
			Binding: observation.Binding,

			Kind:           observation.Kind,
			LogicalName:    observation.LogicalName,
			LogicalVersion: observation.LogicalVersion,

			ResolvedDefinition: &resolved,
			SourceContentDigest: cryptoutil.CloneDigest(
				observation.SourceContentDigest,
			),
			State:       artifact.StateAvailable,
			Diagnostics: diagnostic.Clone(observation.Diagnostics),

			DisplayName: displayName,
			Enabled:     true,
			Data:        json.RawMessage(jsonutil.EmptyObject),

			Revision:   1,
			CreatedAt:  now,
			ModifiedAt: now,
		}
		if err := created.Validate(); err != nil {
			return Synchronization{}, err
		}
		result.Creates = append(result.Creates, created)
		existingByTypedBinding[key] = created.Clone()
	}

	return result.Clone(), nil
}

func deriveCurrentArtifact(
	current artifact.Artifact,
	validByTypedBinding map[typedBinding]ingestimpl.Observation,
	validByBinding map[artifact.SourceBinding][]ingestimpl.Observation,
	invalidByBinding map[artifact.SourceBinding]ingestimpl.Observation,
	seenLocators map[spec.Locator]struct{},
	discoverySpec source.DiscoverySpec,
) (artifact.Artifact, bool, error) {
	next := current.Clone()
	key := typedBinding{
		Binding: current.Binding,
		Kind:    current.Kind,
	}

	if observation, found := validByTypedBinding[key]; found {
		if observation.Definition == nil ||
			observation.SourceContentDigest == nil {
			return artifact.Artifact{}, false, fmt.Errorf(
				"%w: valid Source observation is incomplete",
				spec.ErrInvalid,
			)
		}
		resolved := observation.Definition.Digest
		next.LogicalName = observation.LogicalName
		next.LogicalVersion = observation.LogicalVersion
		next.ResolvedDefinition = &resolved
		next.SourceContentDigest = cryptoutil.CloneDigest(
			observation.SourceContentDigest,
		)
		next.State = artifact.StateAvailable
		next.Diagnostics = diagnostic.Clone(observation.Diagnostics)
		return next, !equivalentSourceState(current, next), nil
	}

	if observation, found := invalidForArtifact(
		current,
		invalidByBinding,
	); found {
		next.ResolvedDefinition = nil
		next.SourceContentDigest = cryptoutil.CloneDigest(
			observation.SourceContentDigest,
		)
		next.State = artifact.StateInvalid
		next.Diagnostics = diagnostic.Clone(observation.Diagnostics)
		return next, !equivalentSourceState(current, next), nil
	}

	if alternatives := validByBinding[current.Binding]; len(alternatives) != 0 {
		alternative := alternatives[0]
		if alternative.Definition == nil ||
			alternative.SourceContentDigest == nil {
			return artifact.Artifact{}, false, fmt.Errorf(
				"%w: incompatible Source observation is incomplete",
				spec.ErrInvalid,
			)
		}
		resolved := alternative.Definition.Digest
		next.ResolvedDefinition = &resolved
		next.SourceContentDigest = cryptoutil.CloneDigest(
			alternative.SourceContentDigest,
		)
		next.State = artifact.StateIncompatible
		next.Diagnostics = diagnostic.Append(
			alternative.Diagnostics,
			diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Code:     "artifact.kind-incompatible",
				Message:  "the source declaration now emits another artifact kind",
				Location: &diagnostic.Location{
					Locator: current.Binding.Locator,
					SubresourceLocator: current.Binding.
						SubresourceLocator,
				},
			},
		)
		return next, !equivalentSourceState(current, next), nil
	}

	_, sourceEntryObserved := seenLocators[current.Binding.Locator]
	inScope, err := discoverySpec.InScope(
		current.Binding.Locator,
	)
	if err != nil {
		return artifact.Artifact{}, false, err
	}
	if sourceEntryObserved ||
		inScope ||
		discoverySpec.Authoritative {
		next.ResolvedDefinition = nil
		next.SourceContentDigest = nil
		next.State = artifact.StateMissing
		next.Diagnostics = []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityWarning,
			Code:     "artifact.source-missing",
			Message:  "the source declaration is missing from the current refresh",
			Location: &diagnostic.Location{
				Locator: current.Binding.Locator,
				SubresourceLocator: current.Binding.
					SubresourceLocator,
			},
		}}
		return next, !equivalentSourceState(current, next), nil
	}

	return current, false, nil
}

func invalidForArtifact(
	current artifact.Artifact,
	invalidByBinding map[artifact.SourceBinding]ingestimpl.Observation,
) (ingestimpl.Observation, bool) {
	if value, exact := invalidByBinding[current.Binding]; exact {
		return value, true
	}

	// A candidate-level failure is represented by an empty subresource. It
	// invalidates every prior subresource emitted from that physical entry.
	candidateBinding := current.Binding
	candidateBinding.SubresourceLocator = ""
	value, found := invalidByBinding[candidateBinding]
	return value, found
}

func equivalentSourceState(
	left artifact.Artifact,
	right artifact.Artifact,
) bool {
	return left.LogicalName == right.LogicalName &&
		left.LogicalVersion == right.LogicalVersion &&
		left.State == right.State &&
		cryptoutil.IsDigestEqual(
			left.ResolvedDefinition,
			right.ResolvedDefinition,
		) &&
		cryptoutil.IsDigestEqual(
			left.SourceContentDigest,
			right.SourceContentDigest,
		) &&
		diagnostic.Equal(left.Diagnostics, right.Diagnostics)
}
