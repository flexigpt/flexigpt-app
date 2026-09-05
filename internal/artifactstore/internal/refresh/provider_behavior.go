package refreshimpl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	artifactimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/artifactid"
	catalogimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type providerRefreshInput struct {
	collection collection.Collection
	behavior   providerapi.CollectionBehavior

	providerCollection  providerapi.Collection
	providerAttachments []providerapi.Attachment
	providerSources     []providerapi.Source

	attachmentsBySource map[basespec.SourceID]providerapi.Attachment
}

// RefreshCollection is the Store-owned collection refresh entrypoint.
//
// The caller supplies only Collection identity. Artifact Store resolves the
// CollectionKind behavior, asks it for a provider plan and adoption decisions,
// then executes existing discovery, reconciliation, and publication logic.
func (s *Service) RefreshCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) (refresh.Result, error) {
	if s == nil || s.providers == nil || s.artifactIDs == nil {
		return refresh.Result{}, basespec.ErrClosed
	}
	if ctx == nil {
		return refresh.Result{}, fmt.Errorf(
			"%w: collection refresh context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return refresh.Result{}, err
	}
	if err := ref.Validate(); err != nil {
		return refresh.Result{}, err
	}
	if err := protection.RequireMutableRoot(
		ctx,
		s.policy,
		ref.RootID,
	); err != nil {
		return refresh.Result{}, err
	}

	input, err := s.loadProviderRefreshInput(ctx, ref)
	if err != nil {
		return refresh.Result{}, err
	}
	plan, err := s.buildProviderPlan(ctx, input)
	if err != nil {
		return refresh.Result{}, err
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
	return catalog.CloneSnapshot(inspection.Catalog), nil
}

// InspectCollectionCatalog reads the latest catalog and reports every Store-
// owned freshness dimension without exposing plan construction or source
// snapshot access to callers.
func (s *Service) InspectCollectionCatalog(
	ctx context.Context,
	ref collection.CollectionRef,
) (refresh.CatalogInspection, error) {
	if s == nil ||
		s.providers == nil ||
		s.catalogs == nil ||
		s.discovery == nil {
		return refresh.CatalogInspection{}, basespec.ErrClosed
	}
	if ctx == nil {
		return refresh.CatalogInspection{}, fmt.Errorf(
			"%w: collection catalog inspection context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return refresh.CatalogInspection{}, err
	}
	if err := ref.Validate(); err != nil {
		return refresh.CatalogInspection{}, err
	}

	input, err := s.loadProviderRefreshInput(ctx, ref)
	if err != nil {
		return refresh.CatalogInspection{}, err
	}
	plan, err := s.buildProviderPlan(ctx, input)
	if err != nil {
		return refresh.CatalogInspection{}, err
	}
	snapshot, catalogErr := catalogimpl.ReadCurrent(ctx, s.catalogs, ref)
	if catalogErr != nil &&
		!errors.Is(catalogErr, basespec.ErrCatalogStale) {
		return refresh.CatalogInspection{}, catalogErr
	}

	planFingerprint, err := plan.Fingerprint()
	if err != nil {
		return refresh.CatalogInspection{}, err
	}
	decoderFingerprint, err := s.discovery.DecoderFingerprint()
	if err != nil {
		return refresh.CatalogInspection{}, err
	}

	return refresh.CatalogInspection{
		Catalog:         catalog.CloneSnapshot(snapshot),
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
		map[basespec.SourceID]providerapi.Attachment,
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
) (discovery.Plan, error) {
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
		return discovery.Plan{}, fmt.Errorf(
			"%w: collection behavior %q has no planning role",
			basespec.ErrInvalid,
			input.behavior.CollectionKind(),
		)
	}
	if err != nil {
		return discovery.Plan{}, err
	}

	revision := input.behavior.Revision()
	if err := basespec.ValidateRequiredText(
		"collection behavior revision",
		revision,
		basespec.MaxVersionBytes,
	); err != nil {
		return discovery.Plan{}, err
	}

	if plan.Revision != "" && plan.Revision != revision {
		return discovery.Plan{}, fmt.Errorf(
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
		return discovery.Plan{}, err
	}

	return discoveryPlanFromProvider(plan)
}

func discoveryPlanFromProvider(
	input providerapi.Plan,
) (discovery.Plan, error) {
	input = input.Normalized()
	if err := input.Validate(); err != nil {
		return discovery.Plan{}, err
	}

	output := discovery.Plan{
		Revision: input.Revision,
		Sources:  make([]discovery.SourcePlan, len(input.Sources)),
	}

	for index, sourcePlan := range input.Sources {
		value := discovery.SourcePlan{
			SourceID: sourcePlan.SourceID,
			ExplicitLocators: append(
				[]basespec.Locator(nil),
				sourcePlan.ExplicitLocators...,
			),
			DirectoryRoots: make(
				[]discovery.DirectoryRoot,
				len(sourcePlan.DirectoryRoots),
			),
			DecoderHints: make(
				[]discovery.DecoderHint,
				len(sourcePlan.DecoderHints),
			),
			ExpectedContentDigests: maps.Clone(
				sourcePlan.ExpectedContentDigests,
			),
			ExpectedGeneration: sourcePlan.ExpectedGeneration,
			AllowedDecoderIDs: append(
				[]basespec.DecoderID(nil),
				sourcePlan.AllowedDecoderIDs...,
			),
			Authoritative:     sourcePlan.Authoritative,
			MaxCandidateBytes: sourcePlan.MaxCandidateBytes,
			MaxTotalBytes:     sourcePlan.MaxTotalBytes,
			MaxCandidates:     sourcePlan.MaxCandidates,
			MaxEntries:        sourcePlan.MaxEntries,
			MaxDepth:          sourcePlan.MaxDepth,
		}

		for rootIndex, root := range sourcePlan.DirectoryRoots {
			value.DirectoryRoots[rootIndex] = discovery.DirectoryRoot{
				Root:      root.Root,
				Recursive: root.Recursive,
				IncludePatterns: append(
					[]string(nil),
					root.IncludePatterns...,
				),
			}
		}
		for hintIndex, hint := range sourcePlan.DecoderHints {
			value.DecoderHints[hintIndex] = discovery.DecoderHint{
				Locator:   hint.Locator,
				Recursive: hint.Recursive,
				DecoderIDs: append(
					[]basespec.DecoderID(nil),
					hint.DecoderIDs...,
				),
			}
		}

		output.Sources[index] = value.Normalized()
	}

	if err := output.Validate(); err != nil {
		return discovery.Plan{}, err
	}
	return output, nil
}

type providerAdoptionPolicy struct {
	behavior    providerapi.CollectionBehavior
	attachments map[basespec.SourceID]providerapi.Attachment
	ids         artifactid.Provider
}

func (p providerAdoptionPolicy) Derive(
	ctx context.Context,
	collectionValue collection.Collection,
	occurrence catalog.Occurrence,
	definitionValue providerapi.Definition,
) (
	artifactimpl.Draft,
	bool,
	[]providerapi.Diagnostic,
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
		providerapi.ContainsErrorDiagnostic(decision.Diagnostics) {
		return artifactimpl.Draft{},
			false,
			providerapi.CloneDiagnostics(decision.Diagnostics),
			nil
	}

	id, err := p.ids.NewArtifactID(ctx)
	if err != nil {
		return artifactimpl.Draft{}, false, nil, err
	}
	if err := basespec.ValidateArtifactID(id); err != nil {
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
		providerapi.CloneDiagnostics(decision.Diagnostics),
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
