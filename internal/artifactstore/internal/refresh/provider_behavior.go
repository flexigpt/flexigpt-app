package refreshimpl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	artifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/artifactid"
	catalogimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/catalog"
	rootimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

type providerRefreshInput struct {
	collection collection.Collection
	behavior   providerapi.CollectionBehavior

	providerCollection  providerapi.Collection
	providerAttachments []providerapi.Attachment
	providerSources     []providerapi.Source

	attachmentsBySource map[source.SourceID]providerapi.Attachment
}

// RefreshCollection is the Store-owned collection refresh entrypoint.
//
// The caller supplies only Collection identity. Artifact Store resolves the
// CollectionKind behavior, asks it for a provider plan and adoption decisions,
// then executes existing discovery, reconciliation, and publication logic.
func (s *Service) RefreshCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) (catalog.RefreshCollectionResult, error) {
	if s == nil || s.providers == nil || s.artifactIDs == nil {
		return catalog.RefreshCollectionResult{}, basespec.ErrClosed
	}
	if ctx == nil {
		return catalog.RefreshCollectionResult{}, fmt.Errorf(
			"%w: collection refresh context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return catalog.RefreshCollectionResult{}, err
	}
	if err := ref.Validate(); err != nil {
		return catalog.RefreshCollectionResult{}, err
	}
	if err := rootimpl.RequireMutableRoot(
		ctx,
		s.policy,
		ref.RootID,
	); err != nil {
		return catalog.RefreshCollectionResult{}, err
	}

	input, err := s.loadProviderRefreshInput(ctx, ref)
	if err != nil {
		return catalog.RefreshCollectionResult{}, err
	}
	plan, err := s.buildProviderPlan(ctx, input)
	if err != nil {
		return catalog.RefreshCollectionResult{}, err
	}

	return s.refresh(
		ctx,
		ref,
		plan,
		providerAdoptionPolicy{
			behavior:    input.behavior,
			attachments: input.attachmentsBySource,
			ids:         s.artifactIDs,
		},
	)
}

// CurrentCatalog returns a catalog only after Artifact Store has confirmed
// that Collection metadata, provider planning behavior, and decoder
// capabilities still match the publication.
func (s *Service) CurrentCatalog(
	ctx context.Context,
	ref collection.CollectionRef,
) (catalog.Snapshot, error) {
	inspection, err := s.InspectCollectionCatalog(ctx, ref)
	if err != nil {
		return catalog.Snapshot{}, err
	}
	if !inspection.IsCurrent() {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: collection %q catalog is stale",
			basespec.ErrCatalogStale,
			ref.CollectionID,
		)
	}
	return inspection.Catalog.Clone(), nil
}

// InspectCollectionCatalog reads the latest catalog and reports every Store-
// owned freshness dimension without exposing plan construction or source
// snapshot access to callers.
func (s *Service) InspectCollectionCatalog(
	ctx context.Context,
	ref collection.CollectionRef,
) (catalog.CatalogInspection, error) {
	if s == nil ||
		s.providers == nil ||
		s.catalogs == nil ||
		s.discovery == nil {
		return catalog.CatalogInspection{}, basespec.ErrClosed
	}
	if ctx == nil {
		return catalog.CatalogInspection{}, fmt.Errorf(
			"%w: collection catalog inspection context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return catalog.CatalogInspection{}, err
	}
	if err := ref.Validate(); err != nil {
		return catalog.CatalogInspection{}, err
	}

	input, err := s.loadProviderRefreshInput(ctx, ref)
	if err != nil {
		return catalog.CatalogInspection{}, err
	}
	plan, err := s.buildProviderPlan(ctx, input)
	if err != nil {
		return catalog.CatalogInspection{}, err
	}
	snapshot, catalogErr := catalogimpl.ReadCurrent(ctx, s.catalogs, ref)
	if catalogErr != nil &&
		!errors.Is(catalogErr, basespec.ErrCatalogStale) {
		return catalog.CatalogInspection{}, catalogErr
	}

	planFingerprint, err := plan.FingerprintNormalized()
	if err != nil {
		return catalog.CatalogInspection{}, err
	}
	decoderFingerprint, err := s.discovery.DecoderFingerprint()
	if err != nil {
		return catalog.CatalogInspection{}, err
	}

	return catalog.CatalogInspection{
		Catalog:         snapshot.Clone(),
		MetadataChanged: errors.Is(catalogErr, basespec.ErrCatalogStale),
		PlanChanged:     snapshot.PlanFingerprint != planFingerprint,
		DecoderChanged:  snapshot.DecoderFingerprint != decoderFingerprint,
	}, nil
}

func (s *Service) loadProviderRefreshInput(
	ctx context.Context,
	ref collection.CollectionRef,
) (providerRefreshInput, error) {
	collectionValue, err := s.collections.Get(ctx, ref)
	if err != nil {
		return providerRefreshInput{}, err
	}
	if err := collectionValue.Validate(); err != nil {
		return providerRefreshInput{}, fmt.Errorf(
			"%w: collection reader returned an invalid collection: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if collectionValue.Ref() != ref {
		return providerRefreshInput{}, fmt.Errorf(
			"%w: collection reader returned another collection",
			basespec.ErrInvalid,
		)
	}

	behavior, found := s.providers.CollectionBehavior(
		collectionValue.Kind,
	)
	if !found {
		return providerRefreshInput{}, fmt.Errorf(
			"%w: collection kind %q has no registered Artifact Store behavior",
			basespec.ErrUnsupported,
			collectionValue.Kind,
		)
	}
	if err := providerapi.ValidateCollectionBehavior(behavior); err != nil {
		return providerRefreshInput{}, fmt.Errorf(
			"%w: collection behavior for kind %q is invalid: %w",
			basespec.ErrInvalid,
			collectionValue.Kind,
			err,
		)
	}
	if behavior.CollectionKind() != collectionValue.Kind {
		return providerRefreshInput{}, fmt.Errorf(
			"%w: collection behavior kind %q does not match collection kind %q",
			basespec.ErrInvalid,
			behavior.CollectionKind(),
			collectionValue.Kind,
		)
	}

	attachments, err := s.collections.ListAttachments(ctx, ref)
	if err != nil {
		return providerRefreshInput{}, err
	}

	providerAttachments := make(
		[]providerapi.Attachment,
		0,
		len(attachments),
	)
	providerSources := make(
		[]providerapi.Source,
		0,
		len(attachments),
	)
	attachmentsBySource := make(
		map[source.SourceID]providerapi.Attachment,
		len(attachments),
	)

	for index, attachment := range attachments {
		if err := attachment.Validate(); err != nil {
			return providerRefreshInput{}, fmt.Errorf(
				"%w: collection reader returned invalid attachment %d: %w",
				basespec.ErrInvalid,
				index,
				err,
			)
		}
		if attachment.RootID != ref.RootID ||
			attachment.CollectionID != ref.CollectionID {
			return providerRefreshInput{}, fmt.Errorf(
				"%w: attachment %d belongs to another collection",
				basespec.ErrInvalid,
				index,
			)
		}
		if _, duplicate := attachmentsBySource[attachment.SourceID]; duplicate {
			return providerRefreshInput{}, fmt.Errorf(
				"%w: collection reader returned duplicate attachment source %q",
				basespec.ErrInvalid,
				attachment.SourceID,
			)
		}

		sourceValue, err := s.sources.Get(
			ctx,
			ref.RootID,
			attachment.SourceID,
		)
		if err != nil {
			return providerRefreshInput{}, err
		}
		if err := sourceValue.Validate(); err != nil {
			return providerRefreshInput{}, fmt.Errorf(
				"%w: source runtime returned invalid source %q: %w",
				basespec.ErrInvalid,
				attachment.SourceID,
				err,
			)
		}
		if sourceValue.RootID != ref.RootID ||
			sourceValue.ID != attachment.SourceID {
			return providerRefreshInput{}, fmt.Errorf(
				"%w: source runtime returned another source for attachment %q",
				basespec.ErrInvalid,
				attachment.SourceID,
			)
		}

		providerAttachment := providerAttachmentFromStore(attachment)
		providerAttachments = append(
			providerAttachments,
			providerAttachment,
		)
		providerSources = append(
			providerSources,
			providerSourceFromStore(sourceValue),
		)
		attachmentsBySource[attachment.SourceID] = providerAttachment.Clone()
	}

	sort.Slice(providerAttachments, func(left, right int) bool {
		return providerAttachments[left].SourceID <
			providerAttachments[right].SourceID
	})
	sort.Slice(providerSources, func(left, right int) bool {
		return providerSources[left].ID < providerSources[right].ID
	})

	return providerRefreshInput{
		collection:          collectionValue,
		behavior:            behavior,
		providerCollection:  providerCollectionFromStore(collectionValue),
		providerAttachments: providerAttachments,
		providerSources:     providerSources,
		attachmentsBySource: attachmentsBySource,
	}, nil
}

func (s *Service) buildProviderPlan(
	ctx context.Context,
	input providerRefreshInput,
) (providerapi.Plan, error) {
	var (
		plan providerapi.Plan
		err  error
	)
	switch behavior := input.behavior.(type) {
	case providerapi.DocumentPlanningBehavior:
		plan, err = behavior.BuildDiscoveryPlanWithDocuments(
			ctx,
			input.providerCollection.Clone(),
			cloneProviderAttachments(input.providerAttachments),
			cloneProviderSources(input.providerSources),
			newProviderPlanningDocumentReader(
				input,
				s.sources,
				s.documents,
			),
		)

	case providerapi.CollectionPlanner:
		plan, err = behavior.BuildDiscoveryPlan(
			ctx,
			input.providerCollection.Clone(),
			cloneProviderAttachments(input.providerAttachments),
			cloneProviderSources(input.providerSources),
		)

	default:
		return providerapi.Plan{}, fmt.Errorf(
			"%w: collection behavior %q has no planning role",
			basespec.ErrInvalid,
			input.behavior.CollectionKind(),
		)
	}
	if err != nil {
		return providerapi.Plan{}, err
	}

	revision := input.behavior.Revision()
	if err := basespec.ValidateRequiredText(
		"collection behavior revision",
		revision,
		basespec.MaxVersionBytes,
	); err != nil {
		return providerapi.Plan{}, err
	}

	if plan.Revision != "" && plan.Revision != revision {
		return providerapi.Plan{}, fmt.Errorf(
			"%w: collection behavior %q returned plan revision %q, expected %q",
			basespec.ErrInvalid,
			input.behavior.CollectionKind(),
			plan.Revision,
			revision,
		)
	}
	plan.Revision = revision
	plan = plan.Normalized()

	if err := plan.Validate(); err != nil {
		return providerapi.Plan{}, err
	}

	return plan, nil
}

type providerAdoptionPolicy struct {
	behavior    providerapi.CollectionBehavior
	attachments map[source.SourceID]providerapi.Attachment
	ids         artifactid.Provider
}

func (p providerAdoptionPolicy) Derive(
	ctx context.Context,
	collectionValue collection.Collection,
	occurrence catalog.Occurrence,
	definitionValue definition.Definition,
) (
	artifactimpl.Draft,
	bool,
	[]diagnostic.Diagnostic,
	error,
) {
	if p.behavior == nil || p.ids == nil {
		return artifactimpl.Draft{},
			false,
			nil,
			fmt.Errorf(
				"%w: provider adoption dependencies are incomplete",
				basespec.ErrInvalid,
			)
	}

	attachment, found := p.attachments[occurrence.Key.SourceID]
	if !found {
		return artifactimpl.Draft{},
			false,
			nil,
			fmt.Errorf(
				"%w: occurrence source %q has no collection attachment",
				basespec.ErrInvalid,
				occurrence.Key.SourceID,
			)
	}

	decision, err := p.behavior.DecideAutomaticAdoption(
		ctx,
		providerapi.AdoptionInput{
			Collection: providerCollectionFromStore(collectionValue),
			Attachment: attachment.Clone(),
			Occurrence: providerapi.Occurrence{
				RootID:             occurrence.RootID,
				CollectionID:       occurrence.CollectionID,
				SourceID:           occurrence.Key.SourceID,
				Locator:            occurrence.Key.Locator,
				SubresourceLocator: occurrence.Key.SubresourceLocator,
				Kind:               occurrence.Kind,
			},
			Definition: definitionValue.Clone(),
		},
	)
	if err != nil {
		return artifactimpl.Draft{}, false, nil, err
	}

	decision = decision.Clone()
	if err := decision.Validate(); err != nil {
		return artifactimpl.Draft{}, false, nil, err
	}
	if !decision.Adopt ||
		diagnostic.ContainsError(decision.Diagnostics) {
		return artifactimpl.Draft{},
			false,
			diagnostic.Clone(decision.Diagnostics),
			nil
	}

	id, err := p.ids.NewArtifactID(ctx)
	if err != nil {
		return artifactimpl.Draft{}, false, nil, err
	}
	if err := id.Validate(); err != nil {
		return artifactimpl.Draft{}, false, nil, err
	}

	return artifactimpl.Draft{
			ID:      id,
			Name:    decision.Name,
			Enabled: decision.Enabled,
			Data: append(
				json.RawMessage(nil),
				decision.Data...,
			),
		},
		true,
		diagnostic.Clone(decision.Diagnostics),
		nil
}

func providerCollectionFromStore(
	value collection.Collection,
) providerapi.Collection {
	return providerapi.Collection{
		ID:          value.ID,
		RootID:      value.RootID,
		Kind:        value.Kind,
		DisplayName: value.DisplayName,
		Description: value.Description,
		Enabled:     value.Enabled,
		Revision:    value.Revision,
		Data: append(
			json.RawMessage(nil),
			value.Data...,
		),
	}
}

func providerAttachmentFromStore(
	value collection.Attachment,
) providerapi.Attachment {
	return providerapi.Attachment{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
		SourceID:     value.SourceID,
		Role:         value.Role,
		Enabled:      value.Enabled,
		Revision:     value.Revision,
		Data: append(
			json.RawMessage(nil),
			value.Data...,
		),
	}
}

func providerSourceFromStore(
	value source.Source,
) providerapi.Source {
	return providerapi.Source{
		ID:          value.ID,
		RootID:      value.RootID,
		StorageKey:  value.StorageKey,
		Kind:        value.Kind,
		DisplayName: value.DisplayName,
		Enabled:     value.Enabled,
		Revision:    value.Revision,
	}
}

func cloneProviderAttachments(
	input []providerapi.Attachment,
) []providerapi.Attachment {
	output := make([]providerapi.Attachment, len(input))
	for index, value := range input {
		output[index] = value.Clone()
	}
	return output
}

func cloneProviderSources(
	input []providerapi.Source,
) []providerapi.Source {
	return append([]providerapi.Source(nil), input...)
}
