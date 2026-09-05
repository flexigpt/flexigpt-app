package artifact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Service struct {
	repository  Repository
	collections collection.Reader
	catalogs    catalog.Reader
	clock       clockutil.Clock
	policy      protection.RootPolicy
}

func NewService(
	repository Repository,
	collections collection.Reader,
	catalogs catalog.Reader,
	timeClock clockutil.Clock,
	policy protection.RootPolicy,
) (*Service, error) {
	if repository == nil ||
		collections == nil ||
		catalogs == nil ||
		timeClock == nil {
		return nil, fmt.Errorf(
			"%w: artifact service dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		repository:  repository,
		collections: collections,
		catalogs:    catalogs,
		clock:       timeClock,
		policy:      policy,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	ref ArtifactRef,
) (Artifact, error) {
	if err := ref.Validate(); err != nil {
		return Artifact{}, err
	}
	return s.repository.Get(ctx, ref)
}

func (s *Service) ListByCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]Artifact, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.collections.Get(ctx, ref); err != nil {
		return nil, err
	}
	return s.repository.ListByCollection(ctx, ref)
}

func (s *Service) ListSuppressions(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]Suppression, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.collections.Get(ctx, ref); err != nil {
		return nil, err
	}
	values, err := s.repository.ListSuppressions(ctx, ref)
	if err != nil {
		return nil, err
	}
	return append([]Suppression(nil), values...), nil
}

type AdoptRequest struct {
	ArtifactID              basespec.ArtifactID
	Collection              collection.CollectionRef
	Occurrence              catalog.OccurrenceKey
	ExpectedCatalogRevision uint64
	Name                    string
	Enabled                 bool
	Data                    json.RawMessage
}

func (s *Service) Adopt(
	ctx context.Context,
	request AdoptRequest,
) (Artifact, error) {
	if err := protection.RequireMutableRoot(
		ctx,
		s.policy,
		request.Collection.RootID,
	); err != nil {
		return Artifact{}, err
	}
	if err := basespec.ValidateArtifactID(request.ArtifactID); err != nil {
		return Artifact{}, err
	}
	if err := request.Collection.Validate(); err != nil {
		return Artifact{}, err
	}
	if err := request.Occurrence.Validate(); err != nil {
		return Artifact{}, err
	}
	if request.Occurrence.CollectionID != request.Collection.CollectionID {
		return Artifact{}, fmt.Errorf(
			"%w: occurrence belongs to another collection",
			basespec.ErrInvalid,
		)
	}
	collectionValue, err := s.activeCollection(ctx, request.Collection)
	if err != nil {
		return Artifact{}, err
	}
	if _, err := s.collections.GetAttachment(
		ctx,
		request.Collection,
		request.Occurrence.SourceID,
	); err != nil {
		return Artifact{}, err
	}
	snapshot, err := catalog.ReadCurrent(ctx, s.catalogs, request.Collection)
	if err != nil {
		return Artifact{}, err
	}
	if request.ExpectedCatalogRevision == 0 ||
		snapshot.Revision != request.ExpectedCatalogRevision {
		return Artifact{}, basespec.ErrConflict
	}

	var occurrence *catalog.Occurrence
	for index := range snapshot.Occurrences {
		if snapshot.Occurrences[index].Key == request.Occurrence {
			value := snapshot.Occurrences[index]
			occurrence = &value
			break
		}
	}

	if occurrence == nil ||
		occurrence.State != catalog.OccurrenceValid ||
		occurrence.DefinitionDigest == nil {
		return Artifact{}, fmt.Errorf(
			"%w: occurrence is not currently adoptable",
			basespec.ErrReferenceUnresolved,
		)
	}

	data, err := canonicalArtifactData(request.Data)
	if err != nil {
		return Artifact{}, err
	}

	name := request.Name
	if name == "" {
		name = string(occurrence.LogicalName)
	}
	resolved := *occurrence.DefinitionDigest
	now := clockutil.NowUTC(s.clock)
	value := Artifact{
		ID:           request.ArtifactID,
		RootID:       request.Collection.RootID,
		CollectionID: request.Collection.CollectionID,
		Binding: SourceBinding{
			SourceID:           occurrence.Key.SourceID,
			Locator:            occurrence.Key.Locator,
			SubresourceLocator: occurrence.Key.SubresourceLocator,
			ExpectedKind:       occurrence.Kind,
		},
		Kind:               occurrence.Kind,
		Name:               name,
		Enabled:            request.Enabled,
		Adoption:           AdoptionObserved,
		ResolvedDefinition: &resolved,
		Data:               data,
		State:              StateAvailable,
		Diagnostics:        providerapi.CloneDiagnostics(occurrence.Diagnostics),
		Revision:           1,
		CreatedAt:          now,
		ModifiedAt:         now,
	}
	if err := value.Validate(); err != nil {
		return Artifact{}, err
	}
	if err := s.repository.CreateAdopted(
		ctx,
		value,
		collectionValue.Revision,
		request.ExpectedCatalogRevision,
	); err != nil {
		return s.resolveCreateConflict(ctx, value, err)
	}
	return value.Clone(), nil
}

type PinRequest struct {
	ArtifactID                 basespec.ArtifactID
	Collection                 collection.CollectionRef
	ExpectedCollectionRevision uint64
	Binding                    SourceBinding
	Name                       string
	Enabled                    bool
	Data                       json.RawMessage
}

func (s *Service) Pin(
	ctx context.Context,
	request PinRequest,
) (Artifact, error) {
	if err := protection.RequireMutableRoot(
		ctx,
		s.policy,
		request.Collection.RootID,
	); err != nil {
		return Artifact{}, err
	}
	if err := basespec.ValidateArtifactID(request.ArtifactID); err != nil {
		return Artifact{}, err
	}
	collectionValue, err := s.activeCollection(ctx, request.Collection)
	if err != nil {
		return Artifact{}, err
	}

	if request.ExpectedCollectionRevision == 0 {
		return Artifact{}, fmt.Errorf(
			"%w: expected collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if collectionValue.Revision != request.ExpectedCollectionRevision {
		return Artifact{}, basespec.ErrConflict
	}

	if err := s.requireAttachedBinding(ctx, request.Collection, request.Binding); err != nil {
		return Artifact{}, err
	}
	if err := basespec.ValidateRequiredText(
		"pinned artifact name",
		request.Name,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return Artifact{}, err
	}

	data, err := canonicalArtifactData(request.Data)
	if err != nil {
		return Artifact{}, err
	}

	var (
		resolvedDefinition *cryptoutil.Digest
		state              = StateMissing
		diagnostics        = []providerapi.Diagnostic{{
			Severity: providerapi.DiagnosticInfo,
			Code:     "artifact.pinned.awaiting-catalog",
			Message:  "the pinned source binding has no current catalog observation",
		}}
		expectedCatalogRevision uint64
	)

	snapshot, snapshotErr := catalog.ReadCurrent(ctx, s.catalogs, request.Collection)
	switch {
	case snapshotErr == nil:
		expectedCatalogRevision = snapshot.Revision
		var observed *catalog.Occurrence
		for index := range snapshot.Occurrences {
			value := snapshot.Occurrences[index]
			if value.Key.SourceID != request.Binding.SourceID ||
				value.Key.Locator != request.Binding.Locator ||
				value.Key.SubresourceLocator != request.Binding.SubresourceLocator {
				continue
			}
			observed = &value
			break
		}

		provisional := Artifact{
			Binding: request.Binding,
			Kind:    request.Binding.ExpectedKind,
		}
		resolvedDefinition, state, diagnostics, err = DeriveSourceState(
			provisional,
			observed,
		)
		if err != nil {
			return Artifact{}, err
		}

	case errors.Is(snapshotErr, basespec.ErrCatalogUnavailable):
	case errors.Is(snapshotErr, basespec.ErrCatalogStale):
		diagnostics = providerapi.AppendDiagnostics(
			diagnostics,
			providerapi.Diagnostic{
				Severity: providerapi.DiagnosticWarning,
				Code:     "artifact.pinned.catalog-stale",
				Message:  "the pinned source binding will be reconciled after the collection catalog is refreshed",
			},
		)
	default:
		return Artifact{}, snapshotErr
	}

	now := clockutil.NowUTC(s.clock)
	value := Artifact{
		ID:                 request.ArtifactID,
		RootID:             request.Collection.RootID,
		CollectionID:       request.Collection.CollectionID,
		Binding:            request.Binding,
		Kind:               request.Binding.ExpectedKind,
		Name:               request.Name,
		Enabled:            request.Enabled,
		Adoption:           AdoptionPinned,
		Data:               data,
		ResolvedDefinition: resolvedDefinition,
		State:              state,
		Diagnostics:        diagnostics,
		Revision:           1,
		CreatedAt:          now,
		ModifiedAt:         now,
	}
	if err := value.Validate(); err != nil {
		return Artifact{}, err
	}
	if err := s.repository.CreatePinned(
		ctx,
		value,
		request.ExpectedCollectionRevision,
		expectedCatalogRevision,
	); err != nil {
		return s.resolveCreateConflict(ctx, value, err)
	}
	return value.Clone(), nil
}

func (s *Service) SetEnabled(
	ctx context.Context,
	ref ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (Artifact, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return Artifact{}, err
	}
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return Artifact{}, err
	}
	if err := s.ensureArtifactCollection(ctx, current); err != nil {
		return Artifact{}, err
	}
	if expectedRevision == 0 || current.Revision != expectedRevision {
		return Artifact{}, basespec.ErrConflict
	}
	if current.Enabled == enabled {
		return current, nil
	}
	next := current
	next.Enabled = enabled
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return Artifact{}, err
	}
	if err := s.repository.Update(ctx, next, expectedRevision); err != nil {
		return Artifact{}, err
	}
	return next.Clone(), nil
}

func (s *Service) SetName(
	ctx context.Context,
	ref ArtifactRef,
	expectedRevision uint64,
	name string,
) (Artifact, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return Artifact{}, err
	}
	if err := basespec.ValidateRequiredText(
		"artifact name",
		name,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return Artifact{}, err
	}
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return Artifact{}, err
	}
	if err := s.ensureArtifactCollection(ctx, current); err != nil {
		return Artifact{}, err
	}
	if expectedRevision == 0 || current.Revision != expectedRevision {
		return Artifact{}, basespec.ErrConflict
	}
	if current.Name == name {
		return current.Clone(), nil
	}
	next := current
	next.Name = name
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return Artifact{}, err
	}
	if err := s.repository.Update(ctx, next, expectedRevision); err != nil {
		return Artifact{}, err
	}
	return next.Clone(), nil
}

func (s *Service) UpdateData(
	ctx context.Context,
	ref ArtifactRef,
	expectedRevision uint64,
	data json.RawMessage,
) (Artifact, error) {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return Artifact{}, err
	}
	canonical, err := canonicalArtifactData(data)
	if err != nil {
		return Artifact{}, err
	}
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return Artifact{}, err
	}
	if err := s.ensureArtifactCollection(ctx, current); err != nil {
		return Artifact{}, err
	}
	if expectedRevision == 0 || current.Revision != expectedRevision {
		return Artifact{}, basespec.ErrConflict
	}
	if jsonutil.Equal(current.Data, canonical) {
		return current, nil
	}
	next := current
	next.Data = canonical
	next.Revision++
	next.ModifiedAt = clockutil.Next(s.clock, current.ModifiedAt)
	if err := next.Validate(); err != nil {
		return Artifact{}, err
	}
	if err := s.repository.Update(ctx, next, expectedRevision); err != nil {
		return Artifact{}, err
	}
	return next.Clone(), nil
}

func (s *Service) Unadopt(
	ctx context.Context,
	ref ArtifactRef,
	expectedRevision uint64,
	suppress bool,
) error {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return err
	}
	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return err
	}
	if err := s.ensureArtifactCollection(ctx, current); err != nil {
		return err
	}
	if expectedRevision == 0 || current.Revision != expectedRevision {
		return basespec.ErrConflict
	}
	if current.Adoption != AdoptionObserved {
		return fmt.Errorf(
			"%w: only observed artifacts can be unadopted",
			basespec.ErrConflict,
		)
	}

	var suppression *Suppression
	if suppress {
		ref := collection.CollectionRef{
			RootID:       current.RootID,
			CollectionID: current.CollectionID,
		}
		if err := s.requireAttachedBinding(ctx, ref, current.Binding); err != nil {
			return err
		}
		now := clockutil.NowUTC(s.clock)
		value := Suppression{
			RootID:       current.RootID,
			CollectionID: current.CollectionID,
			Binding:      current.Binding,
			Revision:     1,
			CreatedAt:    now,
			ModifiedAt:   now,
		}
		if err := value.Validate(); err != nil {
			return err
		}
		suppression = &value
	}

	return s.repository.Unadopt(ctx, ref, expectedRevision, suppression)
}

type SuppressRequest struct {
	Collection                 collection.CollectionRef
	ExpectedCollectionRevision uint64
	Binding                    SourceBinding
}

func (s *Service) Suppress(
	ctx context.Context,
	request SuppressRequest,
) (Suppression, error) {
	if err := protection.RequireMutableRoot(
		ctx,
		s.policy,
		request.Collection.RootID,
	); err != nil {
		return Suppression{}, err
	}
	if err := request.Collection.Validate(); err != nil {
		return Suppression{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return Suppression{}, fmt.Errorf(
			"%w: expected collection revision is required",
			basespec.ErrInvalid,
		)
	}

	current, err := s.activeCollection(ctx, request.Collection)
	if err != nil {
		return Suppression{}, err
	}
	if current.Revision != request.ExpectedCollectionRevision {
		return Suppression{}, basespec.ErrConflict
	}

	if err := s.requireAttachedBinding(ctx, request.Collection, request.Binding); err != nil {
		return Suppression{}, err
	}

	now := clockutil.NowUTC(s.clock)
	value := Suppression{
		RootID:       request.Collection.RootID,
		CollectionID: request.Collection.CollectionID,
		Binding:      request.Binding,
		Revision:     1,
		CreatedAt:    now,
		ModifiedAt:   now,
	}

	if err := value.Validate(); err != nil {
		return Suppression{}, err
	}
	if err := s.repository.Suppress(
		ctx,
		value,
		request.ExpectedCollectionRevision,
	); err != nil {
		return Suppression{}, err
	}
	return value, nil
}

func (s *Service) Unsuppress(
	ctx context.Context,
	ref collection.CollectionRef,
	binding SourceBinding,
	expectedRevision uint64,
) error {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if err := binding.Validate(); err != nil {
		return err
	}

	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected suppression revision is required",
			basespec.ErrInvalid,
		)
	}
	if _, err := s.collections.Get(ctx, ref); err != nil {
		return err
	}

	return s.repository.Unsuppress(
		ctx,
		ref,
		binding,
		expectedRevision,
	)
}

// Purge destructively removes one local Artifact Record.
//
// Unlike Unadopt, Purge never creates a suppression. It does not remove
// externally owned Source content or immutable definitions.
func (s *Service) Purge(
	ctx context.Context,
	ref ArtifactRef,
	expectedRevision uint64,
) error {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	return s.repository.Purge(ctx, ref, expectedRevision)
}

// PurgeAndSuppress destructively removes one local Artifact Record and
// records its binding as suppressed in the same repository transaction.
//
// It is for source-owned Artifacts whose source role would otherwise
// automatically adopt the same occurrence again after the next refresh.
func (s *Service) PurgeAndSuppress(
	ctx context.Context,
	ref ArtifactRef,
	expectedRevision uint64,
) error {
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected artifact revision is required",
			basespec.ErrInvalid,
		)
	}

	current, err := s.repository.Get(ctx, ref)
	if err != nil {
		return err
	}
	if err := s.ensureArtifactCollection(ctx, current); err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return basespec.ErrConflict
	}

	collectionRef := collection.CollectionRef{
		RootID:       current.RootID,
		CollectionID: current.CollectionID,
	}
	if err := s.requireAttachedBinding(ctx, collectionRef, current.Binding); err != nil {
		return err
	}

	now := clockutil.NowUTC(s.clock)
	suppression := Suppression{
		RootID:       current.RootID,
		CollectionID: current.CollectionID,
		Binding:      current.Binding,
		Revision:     1,
		CreatedAt:    now,
		ModifiedAt:   now,
	}
	if err := suppression.Validate(); err != nil {
		return err
	}
	return s.repository.PurgeAndSuppress(
		ctx,
		ref,
		expectedRevision,
		suppression,
	)
}

func (s *Service) activeCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) (collection.Collection, error) {
	if err := ref.Validate(); err != nil {
		return collection.Collection{}, err
	}
	value, err := s.collections.Get(ctx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	if !value.Enabled {
		return collection.Collection{}, fmt.Errorf(
			"%w: collection %q is disabled",
			basespec.ErrConflict,
			ref.CollectionID,
		)
	}
	return value, nil
}

func (s *Service) requireAttachedBinding(
	ctx context.Context,
	ref collection.CollectionRef,
	binding SourceBinding,
) error {
	if err := binding.Validate(); err != nil {
		return err
	}
	_, err := s.collections.GetAttachment(ctx, ref, binding.SourceID)
	return err
}

func (s *Service) ensureArtifactCollection(
	ctx context.Context,
	value Artifact,
) error {
	ref := collection.CollectionRef{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
	}
	collectionValue, err := s.collections.Get(ctx, ref)
	if err != nil {
		return err
	}
	if collectionValue.RootID != value.RootID ||
		collectionValue.ID != value.CollectionID {
		return fmt.Errorf(
			"%w: artifact belongs to an unavailable collection",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func canonicalArtifactData(raw json.RawMessage) (json.RawMessage, error) {
	value, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(value), nil
}

func (s *Service) resolveCreateConflict(
	ctx context.Context,
	requested Artifact,
	createErr error,
) (Artifact, error) {
	if !errors.Is(createErr, basespec.ErrConflict) {
		return Artifact{}, createErr
	}
	existing, err := s.repository.Get(ctx, requested.Ref())
	if err != nil {
		return Artifact{}, createErr
	}
	if existing.RootID != requested.RootID ||
		existing.CollectionID != requested.CollectionID ||
		existing.Binding != requested.Binding ||
		existing.Kind != requested.Kind ||
		existing.Adoption != requested.Adoption ||
		existing.Name != requested.Name ||
		existing.Enabled != requested.Enabled ||
		!jsonutil.Equal(existing.Data, requested.Data) {
		return Artifact{}, fmt.Errorf(
			"%w: artifact %q creation intent differs",
			basespec.ErrConflict,
			requested.ID,
		)
	}

	return existing.Clone(), nil
}
