package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sort"

	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

func (s *Service) Catalog(
	ctx context.Context,
	rootID artifactstoreSpec.RootID,
) (Catalog, error) {
	workspace, err := s.GetWorkspace(ctx, rootID)
	if err != nil {
		return Catalog{}, err
	}
	generation, err := s.store.GetRootCatalogGeneration(ctx, rootID)
	if err != nil {
		return Catalog{}, err
	}
	occurrences, err := s.store.ListCatalogResourcesForRoot(ctx, rootID)
	if err != nil {
		return Catalog{}, err
	}
	records, err := s.store.ListRecords(ctx, rootID)
	if err != nil {
		return Catalog{}, err
	}
	collections, err := s.store.ListCollections(ctx, rootID, false)
	if err != nil {
		return Catalog{}, err
	}

	confirmed, err := s.store.GetRootCatalogGeneration(ctx, rootID)
	if err != nil {
		return Catalog{}, err
	}
	if confirmed.Generation != generation.Generation ||
		confirmed.RootRevision != generation.RootRevision ||
		!maps.Equal(confirmed.SourceVersions, generation.SourceVersions) {
		return Catalog{}, fmt.Errorf(
			"%w: root catalog changed while Workspace catalog was loading",
			artifactstoreSpec.ErrConflict,
		)
	}

	collectionsByID := make(map[artifactstoreSpec.CollectionID]artifactstoreSpec.ArtifactCollection)
	for _, collection := range collections {
		collectionsByID[collection.CollectionID] = collection
	}
	occurrencesByKey := make(map[string]artifactstoreSpec.CatalogResource, len(occurrences))
	for _, occurrence := range occurrences {
		occurrencesByKey[workspaceOccurrenceKey(
			occurrence.SourceID,
			occurrence.Locator,
			occurrence.SubresourceLocator,
			occurrence.Kind,
		)] = occurrence
	}
	recorded := make(map[string]struct{}, len(records))
	resources := make([]CatalogResource, 0, len(records))
	for _, record := range records {
		if record.LastResolvedDefinitionDigest == nil {
			continue
		}
		definition, err := s.store.GetDefinitionByDigest(
			ctx,
			*record.LastResolvedDefinitionDigest,
		)
		if err != nil {
			return Catalog{}, err
		}
		key := workspaceOccurrenceKey(
			record.SourceID,
			record.Locator,
			record.SubresourceLocator,
			record.Kind,
		)
		var occurrence *artifactstoreSpec.CatalogResource
		if value, exists := occurrencesByKey[key]; exists {
			copyValue := value
			occurrence = &copyValue
			recorded[key] = struct{}{}
		}
		var collection *artifactstoreSpec.ArtifactCollection
		if record.CollectionID != nil {
			if value, exists := collectionsByID[*record.CollectionID]; exists {
				copyValue := value
				collection = &copyValue
			}
		}
		resources = append(resources, CatalogResource{
			Record:     record,
			Definition: definition,
			Collection: collection,
			Occurrence: occurrence,
		})
	}
	sort.Slice(resources, func(left, right int) bool {
		if resources[left].Record.Kind != resources[right].Record.Kind {
			return resources[left].Record.Kind < resources[right].Record.Kind
		}
		if resources[left].Record.Name != resources[right].Record.Name {
			return resources[left].Record.Name < resources[right].Record.Name
		}
		return resources[left].Record.RecordID < resources[right].Record.RecordID
	})

	unrecorded := make([]artifactstoreSpec.CatalogResource, 0)
	for _, occurrence := range occurrences {
		key := workspaceOccurrenceKey(
			occurrence.SourceID,
			occurrence.Locator,
			occurrence.SubresourceLocator,
			occurrence.Kind,
		)
		if _, exists := recorded[key]; !exists {
			unrecorded = append(unrecorded, occurrence)
		}
	}
	return Catalog{
		Workspace:  workspace,
		Generation: generation,
		Resources:  resources,
		Unrecorded: unrecorded,
	}, nil
}

func (s *Service) Project(
	ctx context.Context,
	recordID artifactstoreSpec.RecordID,
) (Projection, error) {
	record, err := s.store.GetRecord(ctx, recordID)
	if err != nil {
		return Projection{}, err
	}
	workspace, err := s.GetWorkspace(ctx, record.RootID)
	if err != nil {
		return Projection{}, err
	}
	if record.LastResolvedDefinitionDigest == nil {
		return Projection{}, fmt.Errorf(
			"%w: record %q has no resolved definition",
			artifactstoreSpec.ErrConflict,
			recordID,
		)
	}
	definition, err := s.store.GetDefinitionByDigest(
		ctx,
		*record.LastResolvedDefinitionDigest,
	)
	if err != nil {
		return Projection{}, err
	}
	projector, exists := s.projectors[record.Kind]
	if !exists {
		return Projection{
			Kind:     record.Kind,
			RecordID: recordID,
			Value:    append(json.RawMessage(nil), definition.DefinitionJSON...),
		}, nil
	}
	value, diagnostics := projector.Project(ctx, ProjectionInput{
		Workspace:  workspace,
		Record:     record,
		Definition: definition,
	})
	if err := validate.ValidateDiagnostics(diagnostics); err != nil {
		return Projection{}, fmt.Errorf(
			"%w: projector %q returned invalid diagnostics: %w",
			ErrProjectionUnavailable,
			record.Kind,
			err,
		)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == artifactstoreSpec.DiagnosticSeverityError {
			return Projection{}, fmt.Errorf(
				"%w: projector %q reported an error",
				ErrProjectionUnavailable,
				record.Kind,
			)
		}
	}
	return Projection{
		Kind:        record.Kind,
		RecordID:    recordID,
		Value:       value,
		Diagnostics: diagnostics,
	}, nil
}

func (s *Service) ResolveReference(
	ctx context.Context,
	rootID artifactstoreSpec.RootID,
	reference Reference,
) (CatalogResource, error) {
	if (reference.RecordID == nil) == (reference.Selector == nil) {
		return CatalogResource{}, fmt.Errorf(
			"%w: exactly one of recordID or selector is required",
			ErrReferenceUnresolved,
		)
	}
	catalog, err := s.Catalog(ctx, rootID)
	if err != nil {
		return CatalogResource{}, err
	}
	if reference.RecordID != nil {
		for _, resource := range catalog.Resources {
			if resource.Record.RecordID == *reference.RecordID {
				return resource, nil
			}
		}
		return CatalogResource{}, fmt.Errorf(
			"%w: record %q does not belong to workspace %q",
			ErrReferenceUnresolved,
			*reference.RecordID,
			rootID,
		)
	}

	candidates, err := s.store.FindCandidates(ctx, rootID, *reference.Selector)
	if err != nil {
		return CatalogResource{}, err
	}
	if len(candidates) != 1 {
		return CatalogResource{}, fmt.Errorf(
			"%w: selector resolved to %d catalog candidates",
			ErrReferenceUnresolved,
			len(candidates),
		)
	}
	candidate := candidates[0].Resource
	for _, resource := range catalog.Resources {
		if resource.Record.SourceID == candidate.SourceID &&
			resource.Record.Locator == candidate.Locator &&
			resource.Record.SubresourceLocator == candidate.SubresourceLocator &&
			resource.Record.Kind == candidate.Kind {
			return resource, nil
		}
	}
	return CatalogResource{}, fmt.Errorf(
		"%w: matching catalog occurrence has not been synchronized",
		ErrReferenceUnresolved,
	)
}

func (s *Service) ComposeLoadPlan(
	ctx context.Context,
	rootID artifactstoreSpec.RootID,
	recordIDs []artifactstoreSpec.RecordID,
) (LoadPlan, error) {
	catalog, err := s.Catalog(ctx, rootID)
	if err != nil {
		return LoadPlan{}, err
	}
	requested := make(map[artifactstoreSpec.RecordID]struct{}, len(recordIDs))
	for _, recordID := range recordIDs {
		if recordID == "" {
			return LoadPlan{}, fmt.Errorf("%w: load-plan record ID is empty", ErrInvalidWorkspace)
		}
		if _, duplicate := requested[recordID]; duplicate {
			return LoadPlan{}, fmt.Errorf(
				"%w: duplicate load-plan record %q",
				ErrInvalidWorkspace,
				recordID,
			)
		}
		requested[recordID] = struct{}{}
	}
	items := make([]LoadPlanItem, 0, len(requested))
	for _, resource := range catalog.Resources {
		if _, selected := requested[resource.Record.RecordID]; !selected {
			continue
		}
		if !resource.Record.Enabled ||
			resource.Record.State != artifactstoreSpec.RecordStateAvailable {
			return LoadPlan{}, fmt.Errorf(
				"%w: record %q is not enabled and available",
				artifactstoreSpec.ErrConflict,
				resource.Record.RecordID,
			)
		}
		projection, err := s.Project(ctx, resource.Record.RecordID)
		if err != nil {
			return LoadPlan{}, err
		}
		graph, err := s.store.BuildDependencyGraph(ctx, resource.Record.RecordID)
		if err != nil {
			return LoadPlan{}, err
		}
		items = append(items, LoadPlanItem{
			Resource:   resource,
			Projection: projection,
			Dependency: graph,
		})
	}
	if len(items) != len(requested) {
		return LoadPlan{}, fmt.Errorf(
			"%w: one or more requested records were not found",
			ErrReferenceUnresolved,
		)
	}
	sort.Slice(items, func(left, right int) bool {
		return items[left].Resource.Record.RecordID <
			items[right].Resource.Record.RecordID
	})
	diagnostics := make([]artifactstoreSpec.Diagnostic, 0)
	for _, item := range items {
		diagnostics = append(diagnostics, item.Projection.Diagnostics...)
		diagnostics = append(diagnostics, item.Dependency.Diagnostics...)
	}
	return LoadPlan{
		RootID:      rootID,
		Generation:  catalog.Generation,
		Items:       items,
		Diagnostics: diagnostics,
	}, nil
}

func workspaceOccurrenceKey(
	sourceID artifactstoreSpec.SourceID,
	locator artifactstoreSpec.SourceLocator,
	subresource artifactstoreSpec.SubresourceLocator,
	kind artifactstoreSpec.ArtifactKind,
) string {
	return string(sourceID) + "\x00" +
		string(locator) + "\x00" +
		string(subresource) + "\x00" +
		string(kind)
}
