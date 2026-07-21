package engine

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type occurrenceKindKey struct {
	Occurrence catalog.OccurrenceKey
	Kind       artifactstore.ArtifactKind
}

func occurrenceKindIdentity(
	key catalog.OccurrenceKey,
	kind artifactstore.ArtifactKind,
) occurrenceKindKey {
	return occurrenceKindKey{Occurrence: key, Kind: kind}
}

type QueryService struct {
	workspaces  *Service
	catalogs    catalogSnapshotReader
	sources     sourceLookup
	records     recordLookup
	definitions definitionLookup
}

func NewQueryService(
	workspaces *Service,
	catalogs catalogSnapshotReader,
	sources sourceLookup,
	records recordLookup,
	definitions definitionLookup,
) (*QueryService, error) {
	if workspaces == nil ||
		catalogs == nil ||
		sources == nil ||
		records == nil ||
		definitions == nil {
		return nil, fmt.Errorf(
			"%w: Workspace query dependencies are incomplete",
			ErrInvalidWorkspace,
		)
	}
	return &QueryService{
		workspaces:  workspaces,
		catalogs:    catalogs,
		sources:     sources,
		records:     records,
		definitions: definitions,
	}, nil
}

func (q *QueryService) GetWorkspace(
	ctx context.Context,
	rootID artifactstore.RootID,
) (Workspace, error) {
	return q.workspaces.Get(ctx, rootID)
}

func (q *QueryService) Resolve(
	ctx context.Context,
	rootID artifactstore.RootID,
	reference Reference,
) (Resource, error) {
	if (reference.RecordID == nil) == (reference.Selector == nil) {
		return Resource{}, fmt.Errorf(
			"%w: exactly one record ID or selector is required",
			ErrReferenceUnresolved,
		)
	}
	view, err := q.Catalog(ctx, rootID)
	if err != nil {
		return Resource{}, err
	}

	if reference.RecordID != nil {
		for _, resourceValue := range view.Resources {
			if resourceValue.Record.ID == *reference.RecordID {
				return resourceValue, nil
			}
		}
		return Resource{}, fmt.Errorf(
			"%w: record %q does not belong to Workspace %q",
			ErrReferenceUnresolved,
			*reference.RecordID,
			rootID,
		)
	}

	selector := *reference.Selector
	if err := selector.Validate(); err != nil {
		return Resource{}, err
	}
	priorities := make(map[artifactstore.SourceID]int)
	for _, attachment := range view.Workspace.Attachments {
		if attachment.Enabled {
			priorities[attachment.SourceID] = attachment.Priority
		}
	}

	var selected *Resource
	selectedPriority := 0
	prioritySet := false
	tied := false

	for index := range view.Resources {
		resourceValue := &view.Resources[index]
		if !resourceValue.Record.Enabled ||
			resourceValue.Record.State != record.StateAvailable ||
			!matchesSelector(resourceValue.Definition, selector) {
			continue
		}
		priority, attached := priorities[resourceValue.Source.ID]
		if !attached {
			continue
		}
		if !prioritySet || priority > selectedPriority {
			copyValue := *resourceValue
			selected = &copyValue
			selectedPriority = priority
			prioritySet = true
			tied = false
		} else if priority == selectedPriority {
			tied = true
		}
	}
	if selected == nil {
		return Resource{}, ErrReferenceUnresolved
	}
	if tied {
		return Resource{}, ErrReferenceAmbiguous
	}
	return *selected, nil
}

func (q *QueryService) ComposeLoadPlan(
	ctx context.Context,
	rootID artifactstore.RootID,
	recordIDs []artifactstore.RecordID,
) (LoadPlan, error) {
	view, err := q.Catalog(ctx, rootID)
	if err != nil {
		return LoadPlan{}, err
	}
	requested := make(map[artifactstore.RecordID]struct{}, len(recordIDs))
	for _, recordID := range recordIDs {
		if err := artifactstore.ValidateRecordID(recordID); err != nil {
			return LoadPlan{}, err
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

	plan := LoadPlan{
		RootID:          rootID,
		CatalogRevision: view.Catalog.Revision,
	}
	for _, resourceValue := range view.Resources {
		if _, selected := requested[resourceValue.Record.ID]; !selected {
			continue
		}
		if !resourceValue.Record.Enabled ||
			resourceValue.Record.State != record.StateAvailable {
			return LoadPlan{}, fmt.Errorf(
				"%w: record %q is not enabled and available",
				artifactstore.ErrConflict,
				resourceValue.Record.ID,
			)
		}
		if resourceValue.Record.Mode == record.ModeLinked &&
			!resourceValue.CatalogCurrent {
			return LoadPlan{}, fmt.Errorf(
				"%w: linked record %q is not current",
				artifactstore.ErrCatalogStale,
				resourceValue.Record.ID,
			)
		}
		plan.Items = append(plan.Items, LoadPlanItem{
			Record:     resourceValue.Record,
			Definition: resourceValue.Definition,
			Source:     resourceValue.Source,
		})
		plan.Diagnostics = artifactstore.AppendDiagnostics(
			plan.Diagnostics,
			resourceValue.Record.Diagnostics...,
		)
	}
	if len(plan.Items) != len(requested) {
		return LoadPlan{}, ErrReferenceUnresolved
	}
	sort.Slice(plan.Items, func(left, right int) bool {
		return plan.Items[left].Record.ID < plan.Items[right].Record.ID
	})
	return plan, nil
}

func (q *QueryService) Catalog(
	ctx context.Context,
	rootID artifactstore.RootID,
) (CatalogView, error) {
	workspaceValue, err := q.workspaces.Get(ctx, rootID)
	if err != nil {
		return CatalogView{}, err
	}
	snapshot, err := q.catalogs.GetCurrent(ctx, rootID)
	if err != nil {
		return CatalogView{}, err
	}
	if err := q.ensureCurrent(ctx, workspaceValue, snapshot); err != nil {
		return CatalogView{}, err
	}

	records, err := q.records.ListByRoot(ctx, rootID)
	if err != nil {
		return CatalogView{}, err
	}
	occurrencesByKey := make(map[occurrenceKindKey]catalog.Occurrence)
	for _, occurrence := range snapshot.Occurrences {
		if occurrence.Kind == "" {
			continue
		}
		key := occurrenceKindIdentity(
			occurrence.Key,
			occurrence.Kind,
		)
		occurrencesByKey[key] = occurrence
	}

	sourcesByID := make(map[artifactstore.SourceID]source.Source)
	for _, value := range workspaceValue.Sources {
		sourcesByID[value.ID] = value
	}

	recorded := make(map[occurrenceKindKey]struct{}, len(records))
	view := CatalogView{
		Workspace: workspaceValue,
		Catalog:   snapshot,
	}
	for _, localRecord := range records {
		key := occurrenceKindIdentity(
			localRecord.Occurrence,
			localRecord.Kind,
		)
		recorded[key] = struct{}{}

		if localRecord.ResolvedDefinition == nil {
			view.UnresolvedRecords = append(
				view.UnresolvedRecords,
				localRecord,
			)
			continue
		}
		definitionValue, err := q.definitions.Get(
			ctx,
			*localRecord.ResolvedDefinition,
		)
		if err != nil {
			return CatalogView{}, err
		}
		var occurrencePointer *catalog.Occurrence
		occurrence, found := occurrencesByKey[key]
		if found {
			copyValue := occurrence
			occurrencePointer = &copyValue
		}
		sourceValue, exists := sourcesByID[localRecord.Occurrence.SourceID]
		if !exists {
			return CatalogView{}, fmt.Errorf(
				"%w: record source %q is unavailable",
				ErrInvalidWorkspace,
				localRecord.Occurrence.SourceID,
			)
		}
		current := occurrencePointer != nil &&
			occurrencePointer.State == catalog.OccurrenceValid &&
			occurrencePointer.DefinitionDigest != nil &&
			*occurrencePointer.DefinitionDigest ==
				*localRecord.ResolvedDefinition

		view.Resources = append(view.Resources, Resource{
			Record:         localRecord,
			Definition:     definitionValue,
			Occurrence:     occurrencePointer,
			Source:         sourceValue,
			CatalogCurrent: current,
		})
	}

	for _, occurrence := range snapshot.Occurrences {
		if occurrence.Kind == "" {
			continue
		}
		key := occurrenceKindIdentity(
			occurrence.Key,
			occurrence.Kind,
		)
		if _, exists := recorded[key]; !exists {
			view.Unrecorded = append(view.Unrecorded, occurrence)
		}
	}
	sort.Slice(view.Resources, func(left, right int) bool {
		if view.Resources[left].Record.Kind !=
			view.Resources[right].Record.Kind {
			return view.Resources[left].Record.Kind <
				view.Resources[right].Record.Kind
		}
		if view.Resources[left].Record.Name !=
			view.Resources[right].Record.Name {
			return view.Resources[left].Record.Name <
				view.Resources[right].Record.Name
		}
		return view.Resources[left].Record.ID <
			view.Resources[right].Record.ID
	})
	view.Groups = groupCatalogResources(view.Resources, view.Unrecorded)
	return view, nil
}

func (q *QueryService) ensureCurrent(
	ctx context.Context,
	workspaceValue Workspace,
	snapshot catalog.Snapshot,
) error {
	if snapshot.RootRevision != workspaceValue.Root.Revision {
		return artifactstore.ErrCatalogStale
	}
	currentRevisions := make(map[artifactstore.SourceID]uint64)
	for _, sourceValue := range workspaceValue.Sources {
		currentRevisions[sourceValue.ID] = sourceValue.Revision
	}
	if !maps.Equal(currentRevisions, snapshot.SourceRevisions) {
		return artifactstore.ErrCatalogStale
	}
	return nil
}

func matchesSelector(
	value definition.Definition,
	selector definition.Selector,
) bool {
	if value.Kind != selector.Kind {
		return false
	}
	if selector.LogicalName != "" &&
		value.LogicalName != selector.LogicalName {
		return false
	}
	for key, expected := range selector.Labels {
		if value.Labels[key] != expected {
			return false
		}
	}
	constraint := strings.TrimSpace(selector.VersionConstraint)
	if constraint == "" {
		return true
	}
	constraint = strings.TrimSpace(strings.TrimPrefix(
		constraint,
		exactVersionConstraintOp,
	))
	return constraint == string(value.LogicalVersion)
}

func groupCatalogResources(
	resources []Resource,
	unrecorded []catalog.Occurrence,
) []ResourceGroup {
	values := make(map[artifactstore.ArtifactKind]*ResourceGroup)
	for _, resourceValue := range resources {
		kind := resourceValue.Definition.Kind
		group := values[kind]
		if group == nil {
			group = &ResourceGroup{Kind: kind}
			values[kind] = group
		}
		group.Resources = append(group.Resources, resourceValue)
	}
	for _, occurrence := range unrecorded {
		if occurrence.Kind == "" {
			continue
		}
		group := values[occurrence.Kind]
		if group == nil {
			group = &ResourceGroup{Kind: occurrence.Kind}
			values[occurrence.Kind] = group
		}
		group.Unrecorded = append(group.Unrecorded, occurrence)
	}

	output := make([]ResourceGroup, 0, len(values))
	for _, group := range values {
		output = append(output, *group)
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].Kind < output[right].Kind
	})
	return output
}
