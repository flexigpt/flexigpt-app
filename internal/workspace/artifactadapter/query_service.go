package artifactadapter

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type occurrenceKindKey struct {
	Occurrence catalog.OccurrenceKey
	Kind       basespec.ArtifactKind
}

type QueryService struct {
	workspaces *Service
	catalogs   refresh.CollectionCatalogInspector
	artifacts  artifactLookup
	validators map[basespec.ArtifactKind]spec.DefinitionValidator
}

func NewQueryService(
	workspaces *Service,
	artifacts artifactLookup,
	catalogs refresh.CollectionCatalogInspector,
	supports ...spec.ArtifactSupport,
) (*QueryService, error) {
	if workspaces == nil ||
		catalogs == nil ||
		artifacts == nil {
		return nil, fmt.Errorf(
			"%w: Workspace query dependencies are incomplete",
			spec.ErrInvalidWorkspace,
		)
	}
	if len(supports) == 0 {
		return nil, fmt.Errorf(
			"%w: Workspace query requires at least one supported Artifact kind",
			spec.ErrInvalidWorkspace,
		)
	}
	validators := make(
		map[basespec.ArtifactKind]spec.DefinitionValidator,
		len(supports),
	)
	for _, support := range supports {
		if err := support.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := validators[support.Kind]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate query validator for %q",
				spec.ErrInvalidWorkspace,
				support.Kind,
			)
		}
		validators[support.Kind] = support.Validator
	}
	return &QueryService{
		workspaces: workspaces,
		catalogs:   catalogs,
		artifacts:  artifacts,
		validators: validators,
	}, nil
}

func (q *QueryService) GetWorkspace(
	ctx context.Context,
	workspace collection.CollectionRef,
) (spec.Workspace, error) {
	return q.workspaces.Get(ctx, workspace)
}

func (q *QueryService) ResolveArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (spec.Workspace, spec.Resource, error) {
	if err := ref.Validate(); err != nil {
		return spec.Workspace{}, spec.Resource{}, err
	}
	value, err := q.artifacts.Get(ctx, ref)
	if err != nil {
		return spec.Workspace{}, spec.Resource{}, err
	}
	workspaceRef := collection.CollectionRef{
		RootID:       value.RootID,
		CollectionID: value.CollectionID,
	}
	workspace, err := q.workspaces.Get(ctx, workspaceRef)
	if err != nil {
		return spec.Workspace{}, spec.Resource{}, err
	}
	view, err := q.Catalog(ctx, workspaceRef)
	if err != nil {
		return spec.Workspace{}, spec.Resource{}, err
	}
	for _, resourceValue := range view.Resources {
		if resourceValue.Artifact.ID == value.ID {
			return workspace, resourceValue, nil
		}
	}
	return workspace, spec.Resource{}, fmt.Errorf(
		"%w: Artifact %q is not a current Workspace resource",
		spec.ErrReferenceUnresolved,
		ref.ArtifactID,
	)
}

func (q *QueryService) ComposeLoadPlan(
	ctx context.Context,
	workspace collection.CollectionRef,
	artifactRefs []artifact.ArtifactRef,
) (spec.LoadPlan, error) {
	view, err := q.Catalog(ctx, workspace)
	if err != nil {
		return spec.LoadPlan{}, err
	}
	requested := make(map[basespec.ArtifactID]struct{}, len(artifactRefs))
	for _, ref := range artifactRefs {
		if err := ref.Validate(); err != nil {
			return spec.LoadPlan{}, err
		}
		if ref.RootID != workspace.RootID {
			return spec.LoadPlan{}, fmt.Errorf(
				"%w: ArtifactRef belongs to another Root",
				spec.ErrReferenceUnresolved,
			)
		}
		if _, duplicate := requested[ref.ArtifactID]; duplicate {
			return spec.LoadPlan{}, fmt.Errorf(
				"%w: duplicate load-plan Artifact %q",
				spec.ErrInvalidWorkspace,
				ref.ArtifactID,
			)
		}
		requested[ref.ArtifactID] = struct{}{}
	}

	plan := spec.LoadPlan{
		Workspace:       workspace,
		CatalogRevision: view.Catalog.Revision,
		Diagnostics: providerapi.AppendDiagnostics(
			view.Catalog.Diagnostics,
			view.FreshnessDiagnostics...,
		),
	}
	resources := make(map[basespec.ArtifactID]spec.Resource, len(view.Resources))
	for _, value := range view.Resources {
		resources[value.Artifact.ID] = value
	}
	unresolved := make(
		map[basespec.ArtifactID]artifact.Artifact,
		len(view.UnresolvedArtifacts),
	)
	for _, value := range view.UnresolvedArtifacts {
		unresolved[value.ID] = value
	}

	ordered := make([]basespec.ArtifactID, 0, len(requested))
	for artifactID := range requested {
		ordered = append(ordered, artifactID)
	}
	slices.Sort(ordered)

	for _, artifactID := range ordered {
		resourceValue, found := resources[artifactID]
		if !found {
			if unresolvedValue, exists := unresolved[artifactID]; exists {
				plan.Diagnostics = providerapi.AppendDiagnostics(
					plan.Diagnostics,
					unresolvedValue.Diagnostics...,
				)
				plan.Diagnostics = providerapi.AppendDiagnostics(
					plan.Diagnostics,
					recordAvailabilityDiagnostic(
						unresolvedValue,
						DiagnosticCodeArtifactUnresolved,
						"the Workspace Artifact is unavailable for loading",
					),
				)
			} else {
				plan.Diagnostics = providerapi.AppendDiagnostics(
					plan.Diagnostics,
					providerapi.Diagnostic{
						Severity: providerapi.DiagnosticError,
						Code:     DiagnosticCodeArtifactUnresolved,
						Message:  "the requested Workspace Artifact was not found",
					},
				)
			}
			continue
		}

		switch {
		case !view.CatalogCurrent:
			plan.Diagnostics = providerapi.AppendDiagnostics(
				plan.Diagnostics,
				recordAvailabilityDiagnostic(
					resourceValue.Artifact,
					DiagnosticCodeArtifactUnavailable,
					"the Workspace catalog is stale and must be refreshed",
				),
			)
			continue

		case !resourceValue.Artifact.Enabled:
			plan.Diagnostics = providerapi.AppendDiagnostics(
				plan.Diagnostics,
				recordAvailabilityDiagnostic(
					resourceValue.Artifact,
					DiagnosticCodeArtifactUnavailable,
					"the Workspace Artifact is disabled",
				),
			)
			continue

		case resourceValue.Artifact.State != artifact.StateAvailable:
			plan.Diagnostics = providerapi.AppendDiagnostics(
				plan.Diagnostics,
				recordAvailabilityDiagnostic(
					resourceValue.Artifact,
					DiagnosticCodeArtifactUnavailable,
					"the Workspace Artifact is not available",
				),
			)
			continue

		case !resourceValue.CatalogCurrent:
			plan.Diagnostics = providerapi.AppendDiagnostics(
				plan.Diagnostics,
				recordAvailabilityDiagnostic(
					resourceValue.Artifact,
					DiagnosticCodeArtifactUnavailable,
					"the linked Workspace Artifact is not catalog-current",
				),
			)
			continue

		case !resourceValue.ProjectionValid:
			plan.Diagnostics = providerapi.AppendDiagnostics(
				plan.Diagnostics,
				resourceValue.Diagnostics...,
			)
			continue
		}

		occurrenceDefinitionDigest := cryptoutil.Digest("")
		sourceContentDigest := cryptoutil.Digest("")
		if resourceValue.Occurrence != nil {
			if resourceValue.Occurrence.DefinitionDigest != nil {
				occurrenceDefinitionDigest = *resourceValue.Occurrence.DefinitionDigest
			}
			if resourceValue.Occurrence.SourceContentDigest != nil {
				sourceContentDigest = *resourceValue.Occurrence.SourceContentDigest
			}
		}
		plan.Items = append(plan.Items, spec.LoadPlanItem{
			Artifact:                   resourceValue.Artifact,
			Definition:                 resourceValue.Definition,
			Source:                     resourceValue.Source,
			CatalogCurrent:             resourceValue.CatalogCurrent,
			OccurrenceDefinitionDigest: occurrenceDefinitionDigest,
			SourceContentDigest:        sourceContentDigest,
			SourceGeneration:           view.Catalog.SourceGenerations[resourceValue.Source.ID],
		})
		plan.Diagnostics = providerapi.AppendDiagnostics(
			plan.Diagnostics,
			resourceValue.Artifact.Diagnostics...,
		)
	}
	sort.Slice(plan.Items, func(left, right int) bool {
		return plan.Items[left].Artifact.ID < plan.Items[right].Artifact.ID
	})
	return plan, nil
}

func (q *QueryService) Catalog(
	ctx context.Context,
	workspace collection.CollectionRef,
) (spec.CatalogView, error) {
	if err := workspace.Validate(); err != nil {
		return spec.CatalogView{}, err
	}
	workspaceValue, err := q.workspaces.Get(ctx, workspace)
	if err != nil {
		return spec.CatalogView{}, err
	}
	inspection, err := q.catalogs.InspectCollectionCatalog(
		ctx,
		workspace,
	)
	if err != nil {
		return spec.CatalogView{}, err
	}
	snapshot := inspection.Catalog
	catalogCurrent := inspection.IsCurrent()

	freshnessDiagnostics := make([]providerapi.Diagnostic, 0)
	if inspection.MetadataChanged {
		freshnessDiagnostics = providerapi.AppendDiagnostics(
			freshnessDiagnostics,
			providerapi.Diagnostic{
				Severity: providerapi.DiagnosticWarning,
				Code:     DiagnosticCodeCatalogStale,
				Message:  "the Workspace catalog no longer matches current collection metadata",
			},
		)
	}
	if inspection.DecoderChanged {
		freshnessDiagnostics = providerapi.AppendDiagnostics(
			freshnessDiagnostics,
			providerapi.Diagnostic{
				Severity: providerapi.DiagnosticWarning,
				Code:     DiagnosticCodeCatalogDecoderStale,
				Message:  "the Workspace decoder capability set changed after this catalog was published",
			},
		)
	}
	if inspection.PlanChanged {
		freshnessDiagnostics = providerapi.AppendDiagnostics(
			freshnessDiagnostics,
			providerapi.Diagnostic{
				Severity: providerapi.DiagnosticWarning,
				Code:     DiagnosticCodeCatalogPlanStale,
				Message:  "the Workspace provider discovery behavior changed after this catalog was published",
			},
		)
	}

	artifacts, err := q.artifacts.ListByCollection(ctx, workspace)
	if err != nil {
		return spec.CatalogView{}, err
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

	sourcesByID := make(map[basespec.SourceID]source.Summary)
	for _, value := range workspaceValue.Sources {
		sourcesByID[value.ID] = value
	}

	recorded := make(map[occurrenceKindKey]struct{}, len(artifacts))
	view := spec.CatalogView{
		Workspace:            workspaceValue,
		Catalog:              snapshot,
		CatalogCurrent:       catalogCurrent,
		FreshnessDiagnostics: freshnessDiagnostics,
	}
	for _, localArtifact := range artifacts {
		key := occurrenceKindIdentity(
			catalog.OccurrenceKey{
				CollectionID:       localArtifact.CollectionID,
				SourceID:           localArtifact.Binding.SourceID,
				Locator:            localArtifact.Binding.Locator,
				SubresourceLocator: localArtifact.Binding.SubresourceLocator,
			},
			localArtifact.Kind,
		)
		recorded[key] = struct{}{}

		sourceValue, exists := sourcesByID[localArtifact.Binding.SourceID]
		if !exists {
			view.UnresolvedArtifacts = append(
				view.UnresolvedArtifacts,
				recordWithDiagnostic(
					localArtifact,
					recordSourceUnavailableDiagnostic(localArtifact),
				),
			)
			continue
		}

		occurrence, found := occurrencesByKey[key]
		if !found {
			view.UnresolvedArtifacts = append(
				view.UnresolvedArtifacts,
				recordWithDiagnostic(
					localArtifact,
					recordDefinitionUnavailableDiagnostic(
						localArtifact,
						errors.New("current catalog has no matching occurrence"),
					),
				),
			)
			continue
		}

		if localArtifact.ResolvedDefinition == nil {
			view.UnresolvedArtifacts = append(view.UnresolvedArtifacts, localArtifact)
			continue
		}

		definitionValue, err := catalog.DefinitionForOccurrence(
			snapshot,
			occurrence.Key,
		)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return spec.CatalogView{}, ctxErr
			}
			view.UnresolvedArtifacts = append(
				view.UnresolvedArtifacts,
				recordWithDiagnostic(
					localArtifact,
					recordDefinitionUnavailableDiagnostic(localArtifact, err),
				),
			)
			continue
		}
		if definitionValue.Digest != *localArtifact.ResolvedDefinition {
			view.UnresolvedArtifacts = append(
				view.UnresolvedArtifacts,
				recordWithDiagnostic(
					localArtifact,
					recordDefinitionUnavailableDiagnostic(
						localArtifact,
						errors.New("catalog definition fingerprint differs from artifact state"),
					),
				),
			)
			continue
		}

		projectionValid := true
		projectionDiagnostics := make([]providerapi.Diagnostic, 0)
		if _, dataErr := DecodeArtifactData(localArtifact.Data); dataErr != nil {
			projectionValid = false
			projectionDiagnostics = append(
				projectionDiagnostics,
				projectionDiagnostic(localArtifact, dataErr),
			)
		}

		if definitionValue.Kind != localArtifact.Kind {
			projectionValid = false
			projectionDiagnostics = append(
				projectionDiagnostics,
				projectionDiagnostic(
					localArtifact,
					fmt.Errorf(
						"artifact kind %q does not match resolved definition kind %q",
						localArtifact.Kind,
						definitionValue.Kind,
					),
				),
			)

		} else if validator, supported := q.validators[localArtifact.Kind]; !supported {
			projectionValid = false
			projectionDiagnostics = append(
				projectionDiagnostics,
				projectionDiagnostic(
					localArtifact,
					fmt.Errorf(
						"artifact kind %q has no Workspace validator",
						localArtifact.Kind,
					),
				),
			)
		} else if err := validator(definitionValue); err != nil {
			projectionValid = false
			projectionDiagnostics = append(
				projectionDiagnostics,
				projectionDiagnostic(localArtifact, err),
			)
		}
		occurrenceCopy := catalog.CloneOccurrence(occurrence)
		occurrencePointer := &occurrenceCopy
		current := catalogCurrent &&
			occurrencePointer.State == catalog.OccurrenceValid &&
			occurrencePointer.DefinitionDigest != nil &&
			*occurrencePointer.DefinitionDigest ==
				*localArtifact.ResolvedDefinition

		view.Resources = append(view.Resources, spec.Resource{
			Artifact:        localArtifact,
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
		if view.Resources[left].Artifact.Kind !=
			view.Resources[right].Artifact.Kind {
			return view.Resources[left].Artifact.Kind <
				view.Resources[right].Artifact.Kind
		}
		if view.Resources[left].Artifact.Name !=
			view.Resources[right].Artifact.Name {
			return view.Resources[left].Artifact.Name <
				view.Resources[right].Artifact.Name
		}
		return view.Resources[left].Artifact.ID <
			view.Resources[right].Artifact.ID
	})
	view.Groups = groupCatalogResources(view.Resources, view.Unrecorded)
	return view, nil
}

func recordAvailabilityDiagnostic(
	value artifact.Artifact,
	code string,
	message string,
) providerapi.Diagnostic {
	return providerapi.Diagnostic{
		Severity: providerapi.DiagnosticError,
		Code:     code,
		Message:  message,
		Location: &providerapi.DiagnosticLocation{
			Locator:            value.Binding.Locator,
			SubresourceLocator: value.Binding.SubresourceLocator,
		},
	}
}

func projectionDiagnostic(
	value artifact.Artifact,
	err error,
) providerapi.Diagnostic {
	return providerapi.Diagnostic{
		Severity: providerapi.DiagnosticError,
		Code:     DiagnosticCodeProjectionInvalid,
		Message:  providerapi.BoundedDiagnosticMessage(err.Error()),
		Location: &providerapi.DiagnosticLocation{
			Locator:            value.Binding.Locator,
			SubresourceLocator: value.Binding.SubresourceLocator,
		},
	}
}

func recordWithDiagnostic(
	value artifact.Artifact,
	d providerapi.Diagnostic,
) artifact.Artifact {
	output := value
	output.Diagnostics = providerapi.AppendDiagnostics(
		[]providerapi.Diagnostic{d},
		value.Diagnostics...,
	)
	return output
}

func recordSourceUnavailableDiagnostic(
	value artifact.Artifact,
) providerapi.Diagnostic {
	return providerapi.Diagnostic{
		Severity: providerapi.DiagnosticError,
		Code:     DiagnosticCodeArtifactUnavailable,
		Message:  "the Artifact Source is no longer attached to this Workspace",
		Location: &providerapi.DiagnosticLocation{
			Locator:            value.Binding.Locator,
			SubresourceLocator: value.Binding.SubresourceLocator,
		},
	}
}

func recordDefinitionUnavailableDiagnostic(
	value artifact.Artifact,
	cause error,
) providerapi.Diagnostic {
	return providerapi.Diagnostic{
		Severity: providerapi.DiagnosticError,
		Code:     DiagnosticCodeArtifactUnavailable,
		Message: providerapi.BoundedDiagnosticMessage(
			fmt.Sprintf(
				"the resolved Workspace Artifact definition could not be read: %v",
				cause,
			),
		),
		Location: &providerapi.DiagnosticLocation{
			Locator:            value.Binding.Locator,
			SubresourceLocator: value.Binding.SubresourceLocator,
		},
	}
}

func groupCatalogResources(
	resources []spec.Resource,
	unrecorded []catalog.Occurrence,
) []spec.ResourceGroup {
	values := make(map[basespec.ArtifactKind]*spec.ResourceGroup)
	for _, resourceValue := range resources {
		kind := resourceValue.Artifact.Kind
		group := values[kind]
		if group == nil {
			group = &spec.ResourceGroup{Kind: kind}
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
			group = &spec.ResourceGroup{Kind: occurrence.Kind}
			values[occurrence.Kind] = group
		}
		group.Unrecorded = append(group.Unrecorded, occurrence)
	}

	output := make([]spec.ResourceGroup, 0, len(values))
	for _, group := range values {
		output = append(output, *group)
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].Kind < output[right].Kind
	})
	return output
}

func occurrenceKindIdentity(
	key catalog.OccurrenceKey,
	kind basespec.ArtifactKind,
) occurrenceKindKey {
	return occurrenceKindKey{Occurrence: key, Kind: kind}
}
