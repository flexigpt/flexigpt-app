package engine

import (
	"context"
	"fmt"
	"maps"
	"slices"
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
	records     recordLister
	definitions definitionLookup
	validators  map[artifactstore.ArtifactKind]DefinitionValidator
}

func NewQueryService(
	workspaces *Service,
	catalogs catalogSnapshotReader,
	records recordLister,
	definitions definitionLookup,
	supports ...ArtifactSupport,
) (*QueryService, error) {
	if workspaces == nil ||
		catalogs == nil ||
		records == nil ||
		definitions == nil {
		return nil, fmt.Errorf(
			"%w: Workspace query dependencies are incomplete",
			ErrInvalidWorkspace,
		)
	}
	validators := make(
		map[artifactstore.ArtifactKind]DefinitionValidator,
		len(supports),
	)
	for _, support := range supports {
		if err := support.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := validators[support.Kind]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate query validator for %q",
				ErrInvalidWorkspace,
				support.Kind,
			)
		}
		validators[support.Kind] = support.Validator
	}
	return &QueryService{
		workspaces:  workspaces,
		catalogs:    catalogs,
		records:     records,
		definitions: definitions,
		validators:  validators,
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
	if !view.CatalogCurrent {
		return Resource{}, artifactstore.ErrCatalogStale
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

	var selected *Resource

	for index := range view.Resources {
		resourceValue := &view.Resources[index]
		if !resourceValue.Record.Enabled ||
			resourceValue.Record.State != record.StateAvailable ||
			!matchesSelector(resourceValue.Definition, selector) {
			continue
		}
		if selected != nil {
			return Resource{}, ErrReferenceAmbiguous
		}
		copyValue := *resourceValue
		selected = &copyValue
	}
	if selected == nil {
		return Resource{}, ErrReferenceUnresolved
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
	resources := make(map[artifactstore.RecordID]Resource, len(view.Resources))
	for _, value := range view.Resources {
		resources[value.Record.ID] = value
	}
	unresolved := make(map[artifactstore.RecordID]record.Record, len(view.UnresolvedRecords))
	for _, value := range view.UnresolvedRecords {
		unresolved[value.ID] = value
	}

	ordered := make([]artifactstore.RecordID, 0, len(requested))
	for recordID := range requested {
		ordered = append(ordered, recordID)
	}
	slices.Sort(ordered)

	for _, recordID := range ordered {
		resourceValue, found := resources[recordID]
		if !found {
			if unresolvedValue, exists := unresolved[recordID]; exists {
				plan.Diagnostics = artifactstore.AppendDiagnostics(
					plan.Diagnostics,
					recordAvailabilityDiagnostic(
						unresolvedValue,
						DiagnosticCodeRecordUnresolved,
						"the Workspace record has no resolved definition",
					),
				)
			} else {
				plan.Diagnostics = artifactstore.AppendDiagnostics(
					plan.Diagnostics,
					artifactstore.Diagnostic{
						Severity: artifactstore.DiagnosticError,
						Code:     DiagnosticCodeRecordUnresolved,
						Message:  "the requested Workspace record was not found",
					},
				)
			}
			continue
		}

		switch {
		case !view.CatalogCurrent:
			plan.Diagnostics = artifactstore.AppendDiagnostics(
				plan.Diagnostics,
				recordAvailabilityDiagnostic(
					resourceValue.Record,
					DiagnosticCodeRecordUnavailable,
					"the Workspace catalog is stale and must be refreshed",
				),
			)
			continue

		case !resourceValue.Record.Enabled:
			plan.Diagnostics = artifactstore.AppendDiagnostics(
				plan.Diagnostics,
				recordAvailabilityDiagnostic(
					resourceValue.Record,
					DiagnosticCodeRecordUnavailable,
					"the Workspace record is disabled",
				),
			)
			continue

		case resourceValue.Record.State != record.StateAvailable:
			plan.Diagnostics = artifactstore.AppendDiagnostics(
				plan.Diagnostics,
				recordAvailabilityDiagnostic(
					resourceValue.Record,
					DiagnosticCodeRecordUnavailable,
					"the Workspace record is not available",
				),
			)
			continue

		case !resourceValue.CatalogCurrent:
			plan.Diagnostics = artifactstore.AppendDiagnostics(
				plan.Diagnostics,
				recordAvailabilityDiagnostic(
					resourceValue.Record,
					DiagnosticCodeRecordUnavailable,
					"the linked Workspace record is not catalog-current",
				),
			)
			continue

		case !resourceValue.ProjectionValid:
			plan.Diagnostics = artifactstore.AppendDiagnostics(
				plan.Diagnostics,
				resourceValue.Diagnostics...,
			)
			continue
		}

		occurrenceDefinitionDigest := artifactstore.Digest("")
		sourceContentDigest := artifactstore.Digest("")
		if resourceValue.Occurrence != nil {
			if resourceValue.Occurrence.DefinitionDigest != nil {
				occurrenceDefinitionDigest = *resourceValue.Occurrence.DefinitionDigest
			}
			if resourceValue.Occurrence.SourceContentDigest != nil {
				sourceContentDigest = *resourceValue.Occurrence.SourceContentDigest
			}
		}
		plan.Items = append(plan.Items, LoadPlanItem{
			Record:                     resourceValue.Record,
			Definition:                 resourceValue.Definition,
			Source:                     resourceValue.Source,
			CatalogCurrent:             resourceValue.CatalogCurrent,
			OccurrenceDefinitionDigest: occurrenceDefinitionDigest,
			SourceContentDigest:        sourceContentDigest,
		})
		plan.Diagnostics = artifactstore.AppendDiagnostics(
			plan.Diagnostics,
			resourceValue.Record.Diagnostics...,
		)
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
	catalogCurrent := q.catalogIsCurrent(workspaceValue, snapshot)

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

	sourcesByID := make(map[artifactstore.SourceID]source.Summary)
	for _, value := range workspaceValue.Sources {
		sourcesByID[value.ID] = value
	}

	recorded := make(map[occurrenceKindKey]struct{}, len(records))
	view := CatalogView{
		Workspace:      workspaceValue,
		Catalog:        snapshot,
		CatalogCurrent: catalogCurrent,
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
		definitionValue, err := definition.ReadCanonical(
			ctx,
			q.definitions,
			*localRecord.ResolvedDefinition,
		)
		if err != nil {
			return CatalogView{}, err
		}
		projectionValid := true
		var projectionDiagnostics []artifactstore.Diagnostic
		validator, supported := q.validators[definitionValue.Kind]
		if !supported {
			projectionValid = false
			projectionDiagnostics = append(
				projectionDiagnostics,
				projectionDiagnostic(
					localRecord,
					fmt.Errorf("artifact kind %q has no Workspace validator", definitionValue.Kind),
				),
			)
		} else if err := validator(definitionValue); err != nil {
			projectionValid = false
			projectionDiagnostics = append(
				projectionDiagnostics,
				projectionDiagnostic(localRecord, err),
			)
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
		current := catalogCurrent &&
			occurrencePointer != nil &&
			occurrencePointer.State == catalog.OccurrenceValid &&
			occurrencePointer.DefinitionDigest != nil &&
			*occurrencePointer.DefinitionDigest ==
				*localRecord.ResolvedDefinition

		view.Resources = append(view.Resources, Resource{
			Record:          localRecord,
			Definition:      definitionValue,
			Occurrence:      occurrencePointer,
			Source:          sourceValue,
			CatalogCurrent:  current,
			ProjectionValid: projectionValid,
			Diagnostics:     projectionDiagnostics,
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

func (q *QueryService) catalogIsCurrent(
	workspaceValue Workspace,
	snapshot catalog.Snapshot,
) bool {
	if snapshot.RootRevision != workspaceValue.Root.Revision {
		return false
	}
	currentRevisions := make(map[artifactstore.SourceID]uint64)
	for _, sourceValue := range workspaceValue.Sources {
		currentRevisions[sourceValue.ID] = sourceValue.Revision
	}
	return maps.Equal(currentRevisions, snapshot.SourceRevisions)
}

func recordAvailabilityDiagnostic(
	value record.Record,
	code string,
	message string,
) artifactstore.Diagnostic {
	return artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticError,
		Code:     code,
		Message:  message,
		Location: &artifactstore.DiagnosticLocation{
			Locator:            value.Occurrence.Locator,
			SubresourceLocator: value.Occurrence.SubresourceLocator,
		},
	}
}

func projectionDiagnostic(
	value record.Record,
	err error,
) artifactstore.Diagnostic {
	return artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticError,
		Code:     DiagnosticCodeProjectionInvalid,
		Message:  diagnosticMessage(err.Error()),
		Location: &artifactstore.DiagnosticLocation{
			Locator:            value.Occurrence.Locator,
			SubresourceLocator: value.Occurrence.SubresourceLocator,
		},
	}
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
