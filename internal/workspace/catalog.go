package workspace

import (
	"context"
	"fmt"
	"maps"
	"sort"

	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

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
	return s.projectLoaded(ctx, ProjectionInput{
		Workspace:  workspace,
		Record:     record,
		Definition: definition,
	})
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
	selected, diagnostics := selectWorkspaceCandidate(catalog.Workspace, candidates)
	if selected == nil {
		message := fmt.Sprintf("selector resolved to %d catalog candidates", len(candidates))
		if len(diagnostics) != 0 {
			message = diagnostics[0].Message
		}
		return CatalogResource{}, fmt.Errorf(
			"%w: %s",
			ErrReferenceUnresolved,
			message,
		)
	}
	candidate := selected.Resource
	for _, resource := range catalog.Resources {
		if resource.Record.SourceID == candidate.SourceID &&
			resource.Record.Locator == candidate.Locator &&
			resource.Record.SubresourceLocator == candidate.SubresourceLocator &&
			resource.Record.Kind == candidate.Kind {
			if !resource.CatalogCurrent {
				continue
			}
			if err := s.ensureCatalogGeneration(ctx, catalog.Generation); err != nil {
				return CatalogResource{}, err
			}
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
		if resource.Record.TrackingMode == artifactstoreSpec.TrackingModeFollowSource &&
			!resource.CatalogCurrent {
			return LoadPlan{}, fmt.Errorf(
				"%w: follow-source record %q is not synchronized to the current catalog occurrence",
				artifactstoreSpec.ErrConflict,
				resource.Record.RecordID,
			)
		}
		if resource.Collection != nil && !resource.Collection.Enabled {
			return LoadPlan{}, fmt.Errorf(
				"%w: record %q collection is disabled",
				artifactstoreSpec.ErrConflict,
				resource.Record.RecordID,
			)
		}
		projection, err := s.projectLoaded(ctx, ProjectionInput{
			Workspace:  catalog.Workspace,
			Record:     resource.Record,
			Definition: resource.Definition,
		})
		if err != nil {
			return LoadPlan{}, err
		}
		graph, err := s.buildWorkspaceDependencyGraph(ctx, catalog, resource)
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
	if err := s.ensureCatalogGeneration(ctx, catalog.Generation); err != nil {
		return LoadPlan{}, err
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
	if !sameCatalogGeneration(confirmed, generation) {
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
		occurrencesByKey[workspaceSourceOccurrenceKey(
			occurrence.SourceID,
			occurrence.Locator,
			occurrence.SubresourceLocator,
		)] = occurrence
	}
	recorded := make(map[string]struct{}, len(records))
	resources := make([]CatalogResource, 0, len(records))
	unresolved := make([]artifactstoreSpec.ArtifactRecord, 0)
	for _, record := range records {
		key := workspaceOccurrenceKey(
			record.SourceID,
			record.Locator,
			record.SubresourceLocator,
			record.Kind,
		)
		recorded[key] = struct{}{}

		if record.LastResolvedDefinitionDigest == nil {
			unresolved = append(unresolved, record)
			continue
		}
		definition, err := s.store.GetDefinitionByDigest(
			ctx,
			*record.LastResolvedDefinitionDigest,
		)
		if err != nil {
			return Catalog{}, err
		}
		var occurrence *artifactstoreSpec.CatalogResource
		sourceKey := workspaceSourceOccurrenceKey(
			record.SourceID,
			record.Locator,
			record.SubresourceLocator,
		)
		if value, exists := occurrencesByKey[sourceKey]; exists {
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
		catalogCurrent := occurrence != nil &&
			occurrence.State == artifactstoreSpec.CatalogStateValid &&
			occurrence.CurrentDefinitionDigest != nil &&
			record.LastResolvedDefinitionDigest != nil &&
			*occurrence.CurrentDefinitionDigest == *record.LastResolvedDefinitionDigest
		resources = append(resources, CatalogResource{
			Record:         record,
			Definition:     definition,
			Collection:     collection,
			Occurrence:     occurrence,
			CatalogCurrent: catalogCurrent,
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
	sort.Slice(unresolved, func(left, right int) bool {
		return unresolved[left].RecordID < unresolved[right].RecordID
	})
	return Catalog{
		Workspace:         workspace,
		Generation:        generation,
		Resources:         resources,
		Unrecorded:        unrecorded,
		UnresolvedRecords: unresolved,
	}, nil
}

func (s *Service) projectLoaded(
	ctx context.Context,
	input ProjectionInput,
) (Projection, error) {
	record := input.Record
	definition := input.Definition
	if definition.Kind != record.Kind {
		return Projection{}, fmt.Errorf(
			"%w: definition kind %q does not match record kind %q",
			ErrProjectionUnavailable,
			definition.Kind,
			record.Kind,
		)
	}
	descriptor, known := s.descriptors[record.Kind]
	if !known || definition.SchemaID != descriptor.DefinitionSchemaID {
		return Projection{}, fmt.Errorf(
			"%w: definition schema %q is not registered for kind %q",
			ErrProjectionUnavailable,
			definition.SchemaID,
			record.Kind,
		)
	}
	projector, exists := s.projectors[record.Kind]
	if !exists {
		return Projection{}, fmt.Errorf(
			"%w: no projector is registered for kind %q",
			ErrProjectionUnavailable,
			record.Kind,
		)
	}
	value, diagnostics := projector.Project(ctx, input)
	if err := validate.ValidateDiagnostics(diagnostics); err != nil {
		return Projection{}, fmt.Errorf(
			"%w: projector %q returned invalid diagnostics: %w",
			ErrProjectionUnavailable,
			record.Kind,
			err,
		)
	}
	projection := Projection{
		Kind:        record.Kind,
		RecordID:    record.RecordID,
		Value:       value,
		Diagnostics: append([]artifactstoreSpec.Diagnostic(nil), diagnostics...),
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == artifactstoreSpec.DiagnosticSeverityError {
			return projection, &ProjectionDiagnosticError{
				Kind:        record.Kind,
				Diagnostics: append([]artifactstoreSpec.Diagnostic(nil), diagnostics...),
			}
		}
	}
	if value == nil {
		return projection, fmt.Errorf(
			"%w: projector for kind %q returned a nil value",
			ErrProjectionUnavailable,
			record.Kind,
		)
	}
	return projection, nil
}

func (s *Service) ensureCatalogGeneration(
	ctx context.Context,
	expected artifactstoreSpec.RootCatalogGeneration,
) error {
	current, err := s.store.GetRootCatalogGeneration(ctx, expected.RootID)
	if err != nil {
		return err
	}
	if !sameCatalogGeneration(current, expected) {
		return fmt.Errorf(
			"%w: root catalog changed while Workspace data was loading",
			artifactstoreSpec.ErrConflict,
		)
	}
	return nil
}

func sameCatalogGeneration(
	left artifactstoreSpec.RootCatalogGeneration,
	right artifactstoreSpec.RootCatalogGeneration,
) bool {
	return left.RootID == right.RootID &&
		left.Generation == right.Generation &&
		left.RootRevision == right.RootRevision &&
		left.ScanPlanDigest == right.ScanPlanDigest &&
		left.CatalogDigest == right.CatalogDigest &&
		maps.Equal(left.SourceVersions, right.SourceVersions)
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

func workspaceSourceOccurrenceKey(
	sourceID artifactstoreSpec.SourceID,
	locator artifactstoreSpec.SourceLocator,
	subresource artifactstoreSpec.SubresourceLocator,
) string {
	return string(sourceID) + "\x00" +
		string(locator) + "\x00" +
		string(subresource)
}
