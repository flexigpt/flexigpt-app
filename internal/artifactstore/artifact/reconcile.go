package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type bindingIdentity struct {
	CollectionID artifactstore.CollectionID
	Binding      artifactstore.SourceBinding
}

// occurrenceIdentity intentionally excludes ExpectedKind. A source can stop
// producing one kind and start producing another at the same physical
// location. Existing Artifacts must become incompatible rather than silently
// becoming missing while a second automatically adopted Artifact is created.
type occurrenceIdentity struct {
	SourceID           artifactstore.SourceID
	Locator            artifactstore.Locator
	SubresourceLocator artifactstore.SubresourceLocator
}

func occurrenceIdentityForBinding(
	binding artifactstore.SourceBinding,
) occurrenceIdentity {
	return occurrenceIdentity{
		SourceID:           binding.SourceID,
		Locator:            binding.Locator,
		SubresourceLocator: binding.SubresourceLocator,
	}
}

func occurrenceIdentityForKey(key catalog.OccurrenceKey) occurrenceIdentity {
	return occurrenceIdentity{
		SourceID:           key.SourceID,
		Locator:            key.Locator,
		SubresourceLocator: key.SubresourceLocator,
	}
}

type Reconciler struct {
	ids   artifactstore.IDGenerator
	clock artifactstore.Clock
}

func NewReconciler(
	ids artifactstore.IDGenerator,
	clock artifactstore.Clock,
) (*Reconciler, error) {
	if ids == nil || clock == nil {
		return nil, fmt.Errorf(
			"%w: artifact reconciler dependencies are incomplete",
			artifactstore.ErrInvalid,
		)
	}
	return &Reconciler{ids: ids, clock: clock}, nil
}

// DeriveSourceState derives only the source-owned fields of an existing
// Artifact from its current Collection occurrence.
//
// It is shared by reconciliation and SQLite publication validation so a
// Publisher cannot apply a source-derived state transition that the
// Reconciler itself would never produce.
func DeriveSourceState(
	current Artifact,
	occurrence *catalog.Occurrence,
) (
	*artifactstore.Digest,
	State,
	[]artifactstore.Diagnostic,
	error,
) {
	if occurrence == nil || occurrence.State == catalog.OccurrenceMissing {
		return nil, StateMissing, []artifactstore.Diagnostic{{
			Severity: artifactstore.DiagnosticWarning,
			Code:     "artifact.source-missing",
			Message:  "the artifact source binding is missing",
		}}, nil
	}

	switch occurrence.State {
	case catalog.OccurrenceInvalid:
		return nil,
			StateInvalid,
			artifactstore.CloneDiagnostics(occurrence.Diagnostics),
			nil

	case catalog.OccurrenceValid:
		if occurrence.DefinitionDigest == nil {
			return nil, "", nil, fmt.Errorf(
				"%w: valid source occurrence has no definition digest",
				artifactstore.ErrInvalid,
			)
		}

		resolved := cloneDigest(occurrence.DefinitionDigest)
		if occurrence.Kind == current.Kind {
			return resolved,
				StateAvailable,
				artifactstore.CloneDiagnostics(occurrence.Diagnostics),
				nil
		}

		diagnostics := artifactstore.AppendDiagnostics(
			occurrence.Diagnostics,
			artifactstore.Diagnostic{
				Severity: artifactstore.DiagnosticError,
				Code:     "artifact.kind-incompatible",
				Message:  "the source occurrence changed artifact kind",
				Location: &artifactstore.DiagnosticLocation{
					Locator:            current.Binding.Locator,
					SubresourceLocator: current.Binding.SubresourceLocator,
				},
			},
		)
		return resolved, StateIncompatible, diagnostics, nil

	default:
		return nil, "", nil, fmt.Errorf(
			"%w: unsupported occurrence state %q",
			artifactstore.ErrInvalid,
			occurrence.State,
		)
	}
}

func (r *Reconciler) Reconcile(
	ctx context.Context,
	collectionValue collection.Collection,
	occurrences []catalog.Occurrence,
	existing []Artifact,
	suppressions []Suppression,
	definitions definition.Reader,
	policy Policy,
) (Reconciliation, error) {
	if definitions == nil || policy == nil {
		return Reconciliation{}, fmt.Errorf(
			"%w: artifact reconciliation dependencies are incomplete",
			artifactstore.ErrInvalid,
		)
	}
	if err := collectionValue.Validate(); err != nil {
		return Reconciliation{}, err
	}

	orderedOccurrenceBindings := make(
		[]bindingIdentity,
		0,
		len(occurrences),
	)
	occurrencesByBinding := make(
		map[bindingIdentity]catalog.Occurrence,
		len(occurrences),
	)
	occurrencesBySource := make(
		map[occurrenceIdentity]catalog.Occurrence,
		len(occurrences),
	)
	for index, occurrence := range occurrences {
		if err := occurrence.Validate(); err != nil {
			return Reconciliation{}, fmt.Errorf(
				"occurrence %d: %w",
				index,
				err,
			)
		}

		if occurrence.RootID != collectionValue.RootID ||
			occurrence.CollectionID != collectionValue.ID {
			return Reconciliation{}, fmt.Errorf(
				"%w: occurrence belongs to another collection",
				artifactstore.ErrInvalid,
			)
		}

		sourceIdentity := occurrenceIdentityForKey(occurrence.Key)
		if _, duplicate := occurrencesBySource[sourceIdentity]; duplicate {
			return Reconciliation{}, fmt.Errorf(
				"%w: duplicate source occurrence",
				artifactstore.ErrInvalid,
			)
		}
		occurrencesBySource[sourceIdentity] = occurrence

		if occurrence.Kind == "" {
			continue
		}

		identity := bindingIdentity{
			CollectionID: collectionValue.ID,
			Binding: artifactstore.SourceBinding{
				SourceID:           occurrence.Key.SourceID,
				Locator:            occurrence.Key.Locator,
				SubresourceLocator: occurrence.Key.SubresourceLocator,
				ExpectedKind:       occurrence.Kind,
			},
		}
		if _, duplicate := occurrencesByBinding[identity]; duplicate {
			return Reconciliation{}, fmt.Errorf(
				"%w: duplicate typed occurrence",
				artifactstore.ErrInvalid,
			)
		}

		occurrencesByBinding[identity] = occurrence
		orderedOccurrenceBindings = append(orderedOccurrenceBindings, identity)
	}

	existingByBinding := make(
		map[bindingIdentity]Artifact,
		len(existing),
	)
	existingSourceBindings := make(
		map[occurrenceIdentity]struct{},
		len(existing),
	)
	for index, value := range existing {
		if err := value.Validate(); err != nil {
			return Reconciliation{}, fmt.Errorf(
				"existing artifact %d: %w",
				index,
				err,
			)
		}

		if value.RootID != collectionValue.RootID ||
			value.CollectionID != collectionValue.ID {
			return Reconciliation{}, fmt.Errorf(
				"%w: artifact belongs to another collection",
				artifactstore.ErrInvalid,
			)
		}

		identity := bindingIdentity{
			CollectionID: value.CollectionID,
			Binding:      value.Binding,
		}
		if _, duplicate := existingByBinding[identity]; duplicate {
			return Reconciliation{}, fmt.Errorf(
				"%w: duplicate artifact source binding",
				artifactstore.ErrInvalid,
			)
		}

		existingByBinding[identity] = value
		existingSourceBindings[occurrenceIdentityForBinding(value.Binding)] = struct{}{}
	}

	suppressed := make(map[bindingIdentity]struct{}, len(suppressions))
	for index, value := range suppressions {
		if err := value.Validate(); err != nil {
			return Reconciliation{}, fmt.Errorf(
				"suppression %d: %w",
				index,
				err,
			)
		}

		if value.RootID != collectionValue.RootID ||
			value.CollectionID != collectionValue.ID {
			return Reconciliation{}, fmt.Errorf(
				"%w: suppression belongs to another collection",
				artifactstore.ErrInvalid,
			)
		}

		suppressed[bindingIdentity{
			CollectionID: value.CollectionID,
			Binding:      value.Binding,
		}] = struct{}{}
	}

	result := Reconciliation{}
	now := r.clock.Now().UTC()

	orderedExisting := append([]Artifact(nil), existing...)
	sort.Slice(orderedExisting, func(left, right int) bool {
		return orderedExisting[left].ID < orderedExisting[right].ID
	})

	for _, current := range orderedExisting {
		next := current
		observed, found := occurrencesBySource[occurrenceIdentityForBinding(current.Binding)]
		var occurrence *catalog.Occurrence
		if found {
			occurrence = &observed
		}

		resolved, state, diagnostics, err := DeriveSourceState(
			current,
			occurrence,
		)
		if err != nil {
			return Reconciliation{}, err
		}
		next.ResolvedDefinition = resolved
		next.State = state
		next.Diagnostics = diagnostics

		if equivalentSourceState(current, next) {
			continue
		}
		next.Revision++
		next.ModifiedAt = now
		if !next.ModifiedAt.After(current.ModifiedAt) {
			next.ModifiedAt = current.ModifiedAt.Add(1)
		}

		if err := next.Validate(); err != nil {
			return Reconciliation{}, err
		}
		result.Updates = append(result.Updates, SourceStateUpdate{
			ArtifactID:         next.ID,
			RootID:             next.RootID,
			CollectionID:       next.CollectionID,
			ResolvedDefinition: cloneDigest(next.ResolvedDefinition),
			State:              next.State,
			Diagnostics:        artifactstore.CloneDiagnostics(next.Diagnostics),
			Revision:           next.Revision,
			ModifiedAt:         next.ModifiedAt,
			ExpectedRevision:   current.Revision,
		})
	}

	sort.Slice(orderedOccurrenceBindings, func(left, right int) bool {
		leftValue := orderedOccurrenceBindings[left]
		rightValue := orderedOccurrenceBindings[right]
		if leftValue.Binding.SourceID != rightValue.Binding.SourceID {
			return leftValue.Binding.SourceID < rightValue.Binding.SourceID
		}
		if leftValue.Binding.Locator != rightValue.Binding.Locator {
			return leftValue.Binding.Locator < rightValue.Binding.Locator
		}
		if leftValue.Binding.SubresourceLocator != rightValue.Binding.SubresourceLocator {
			return leftValue.Binding.SubresourceLocator <
				rightValue.Binding.SubresourceLocator
		}
		return leftValue.Binding.ExpectedKind < rightValue.Binding.ExpectedKind
	})

	for _, identity := range orderedOccurrenceBindings {
		occurrence := occurrencesByBinding[identity]
		if occurrence.State != catalog.OccurrenceValid ||
			occurrence.DefinitionDigest == nil {
			continue
		}
		if _, exists := existingByBinding[identity]; exists {
			continue
		}

		if _, exists := existingSourceBindings[occurrenceIdentityForBinding(identity.Binding)]; exists {
			continue
		}

		if _, blocked := suppressed[identity]; blocked {
			continue
		}

		value, err := definition.ReadCanonical(
			ctx,
			definitions,
			collectionValue.RootID,
			*occurrence.DefinitionDigest,
		)
		if err != nil {
			return Reconciliation{}, err
		}

		draft, create, diagnostics := policy.Derive(
			ctx,
			collectionValue,
			occurrence,
			value,
		)
		if err := artifactstore.ValidateDiagnostics(diagnostics); err != nil {
			return Reconciliation{}, err
		}

		result.Diagnostics = artifactstore.AppendDiagnostics(
			result.Diagnostics,
			diagnostics...,
		)
		if !create || artifactstore.ContainsErrorDiagnostic(diagnostics) {
			continue
		}

		data, err := jsoncanon.CanonicalizeObject(
			draft.Data,
			artifactstore.MaxLocalDataBytes,
		)
		if err != nil {
			return Reconciliation{}, err
		}

		id, err := r.ids.NewID(ctx)
		if err != nil {
			return Reconciliation{}, err
		}

		resolved := *occurrence.DefinitionDigest
		created := Artifact{
			ID:                 artifactstore.ArtifactID(id),
			RootID:             collectionValue.RootID,
			CollectionID:       collectionValue.ID,
			Binding:            identity.Binding,
			Kind:               occurrence.Kind,
			Name:               draft.Name,
			Enabled:            draft.Enabled,
			Adoption:           AdoptionObserved,
			ResolvedDefinition: &resolved,
			Data:               json.RawMessage(data),
			State:              StateAvailable,
			Diagnostics: artifactstore.AppendDiagnostics(
				occurrence.Diagnostics,
				diagnostics...,
			),
			Revision:   1,
			CreatedAt:  now,
			ModifiedAt: now,
		}
		if err := created.Validate(); err != nil {
			return Reconciliation{}, err
		}

		result.Creates = append(result.Creates, created)
		existingByBinding[identity] = created
	}
	return result, nil
}

func equivalentSourceState(left, right Artifact) bool {
	return left.State == right.State &&
		digestPointersEqual(left.ResolvedDefinition, right.ResolvedDefinition) &&
		artifactstore.EqualDiagnostics(left.Diagnostics, right.Diagnostics)
}

func cloneDigest(value *artifactstore.Digest) *artifactstore.Digest {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func digestPointersEqual(left, right *artifactstore.Digest) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
