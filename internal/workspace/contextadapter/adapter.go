package contextadapter

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/artifactadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type ContextContribution struct {
	Artifact         artifact.ArtifactRef                      `json:"artifact"`
	ArtifactRevision uint64                                    `json:"artifactRevision"`
	DefinitionDigest cryptoutil.Digest                         `json:"definitionDigest"`
	SourceID         basespec.SourceID                         `json:"sourceID"`
	Locator          basespec.Locator                          `json:"locator"`
	Name             string                                    `json:"name"`
	Role             artifactbuiltin.WorkspaceContextRole      `json:"role"`
	MediaType        artifactbuiltin.WorkspaceContextMediaType `json:"mediaType"`
	Content          string                                    `json:"content"`
	ConventionOrder  int                                       `json:"conventionOrder"`
	OriginalBytes    int                                       `json:"originalBytes"`
	IncludedBytes    int                                       `json:"includedBytes"`
	Truncated        bool                                      `json:"truncated"`
}

type ContextLoadPlan struct {
	Workspace       collection.CollectionRef `json:"workspace"`
	CatalogRevision uint64                   `json:"catalogRevision"`
	Contributions   []ContextContribution    `json:"contributions"`
	Prompt          string                   `json:"prompt"`
	Diagnostics     []providerapi.Diagnostic `json:"diagnostics,omitempty"`
	Decisions       []CompositionDecision    `json:"decisions"`
	PromptBytes     int                      `json:"promptBytes"`
}

type ContextDocument struct {
	Artifact         artifact.ArtifactRef                      `json:"artifact"`
	ArtifactRevision uint64                                    `json:"artifactRevision"`
	DefinitionDigest cryptoutil.Digest                         `json:"definitionDigest"`
	SourceID         basespec.SourceID                         `json:"sourceID"`
	Locator          basespec.Locator                          `json:"locator"`
	Name             string                                    `json:"name"`
	Role             artifactbuiltin.WorkspaceContextRole      `json:"role"`
	MediaType        artifactbuiltin.WorkspaceContextMediaType `json:"mediaType"`
	Enabled          bool                                      `json:"enabled"`
	State            artifact.State                            `json:"state"`
	CatalogCurrent   bool                                      `json:"catalogCurrent"`
	ProjectionValid  bool                                      `json:"projectionValid"`
	RuntimeDisabled  bool                                      `json:"runtimeDisabled"`
	Diagnostics      []providerapi.Diagnostic                  `json:"diagnostics,omitempty"`
}

type ContextInspection struct {
	Workspace       collection.CollectionRef `json:"workspace"`
	CatalogRevision uint64                   `json:"catalogRevision"`
	Contributions   []ContextContribution    `json:"contributions"`
	Diagnostics     []providerapi.Diagnostic `json:"diagnostics,omitempty"`
}

type Adapter struct {
	query             *artifactadapter.QueryService
	runtimePolicy     artifactadapter.SourceUsePolicy
	compositionPolicy CompositionPolicy
}

func NewAdapter(
	query *artifactadapter.QueryService,
	runtimePolicy artifactadapter.SourceUsePolicy,
	compositionPolicy CompositionPolicy,
) (*Adapter, error) {
	if query == nil || runtimePolicy == nil {
		return nil, fmt.Errorf(
			"%w: Workspace context adapter query is nil",
			spec.ErrInvalidWorkspace,
		)
	}
	compositionPolicy = compositionPolicy.Normalized()
	if err := compositionPolicy.Validate(); err != nil {
		return nil, err
	}
	return &Adapter{
		query:             query,
		runtimePolicy:     runtimePolicy,
		compositionPolicy: compositionPolicy,
	}, nil
}

func (p *Adapter) Compose(
	ctx context.Context,
	workspace collection.CollectionRef,
	artifactRefs []artifact.ArtifactRef,
) (ContextLoadPlan, error) {
	if err := workspace.Validate(); err != nil {
		return ContextLoadPlan{}, err
	}
	if len(artifactRefs) == 0 {
		values, err := p.List(ctx, workspace)
		if err != nil {
			return ContextLoadPlan{}, err
		}
		for _, value := range values {
			if value.Enabled && value.State == artifact.StateAvailable {
				artifactRefs = append(artifactRefs, value.Artifact)
			}
		}
	}

	loadPlan, err := p.query.ComposeLoadPlan(ctx, workspace, artifactRefs)
	if err != nil {
		return ContextLoadPlan{}, err
	}

	workspaceValue, err := p.query.GetWorkspace(ctx, workspace)
	if err != nil {
		return ContextLoadPlan{}, err
	}
	output := ContextLoadPlan{
		Workspace:       workspace,
		CatalogRevision: loadPlan.CatalogRevision,
		Diagnostics:     providerapi.CloneDiagnostics(loadPlan.Diagnostics),
	}
	handled := make(map[basespec.ArtifactID]struct{}, len(loadPlan.Items))
	for _, item := range loadPlan.Items {
		handled[item.Artifact.ID] = struct{}{}
		if err := ValidateContextDefinition(item.Definition); err != nil {
			output.Diagnostics = providerapi.AppendDiagnostics(
				output.Diagnostics,
				contextProjectionDiagnostic(item.Artifact, err),
			)
			output.Decisions = append(output.Decisions, CompositionDecision{
				Artifact: item.Artifact.Ref(),
				Status:   CompositionUnavailable,
				Code:     artifactadapter.DiagnosticCodeProjectionInvalid,
			})
			continue
		}
		decision := p.runtimePolicy.Decide(ctx, artifactadapter.RuntimePolicyRequest{
			Use:              artifactadapter.RuntimeUseContextPrompt,
			Workspace:        workspaceValue,
			Artifact:         item.Artifact,
			DefinitionDigest: item.Definition.Digest,
			SourceID:         item.Source.ID,
		})
		if err := decision.Validate(); err != nil {
			return ContextLoadPlan{}, err
		}
		if decision.Disposition != artifactadapter.RuntimeAllowed {
			output.Diagnostics = providerapi.AppendDiagnostics(
				output.Diagnostics,
				artifactadapter.RuntimeDecisionDiagnostic(decision, item.Artifact),
			)
			status := CompositionDenied
			if decision.Disposition == artifactadapter.RuntimeUnavailable {
				status = CompositionUnavailable
			}
			output.Decisions = append(output.Decisions, CompositionDecision{
				Artifact: item.Artifact.Ref(),
				Status:   status,
				Code:     decision.Code,
			})
			continue
		}
		body, err := providerapi.DecodeBody[contextDefinition](
			item.Definition.Body,
		)
		if err != nil {
			handled[item.Artifact.ID] = struct{}{}
			output.Diagnostics = providerapi.AppendDiagnostics(
				output.Diagnostics,
				contextProjectionDiagnostic(item.Artifact, err),
			)
			output.Decisions = append(output.Decisions, CompositionDecision{
				Artifact: item.Artifact.Ref(),
				Status:   CompositionUnavailable,
				Code:     artifactadapter.DiagnosticCodeProjectionInvalid,
			})
			continue
		}
		output.Contributions = append(
			output.Contributions,
			ContextContribution{
				ConventionOrder: contextRuntimeOrder(
					item.Artifact.Binding.Locator,
				),
				ArtifactRevision: item.Artifact.Revision,
				Artifact:         item.Artifact.Ref(),
				DefinitionDigest: item.Definition.Digest,
				SourceID:         item.Source.ID,
				Locator:          item.Artifact.Binding.Locator,
				Name:             body.Name,
				Role:             body.Role,
				MediaType:        body.MediaType,
				Content:          body.Content,
			},
		)
	}
	for _, ref := range artifactRefs {
		if _, found := handled[ref.ArtifactID]; found {
			continue
		}
		output.Decisions = append(output.Decisions, CompositionDecision{
			Artifact: ref,
			Status:   CompositionUnavailable,
			Code:     artifactadapter.DiagnosticCodeArtifactUnresolved,
		})
	}

	sortContextContributions(output.Contributions)
	output.Contributions,
		output.Prompt,
		output.Diagnostics,
		output.Decisions = applyCompositionPolicy(
		p.compositionPolicy,
		output.Contributions,
		output.Diagnostics,
		output.Decisions,
	)
	output.PromptBytes = len(output.Prompt)
	return output, nil
}

func (p *Adapter) List(
	ctx context.Context,
	workspace collection.CollectionRef,
) ([]ContextDocument, error) {
	view, err := p.query.Catalog(ctx, workspace)
	if err != nil {
		return nil, err
	}
	output := make([]ContextDocument, 0)
	for _, resourceValue := range view.Resources {
		if resourceValue.Definition.Kind != artifactbuiltin.WorkspaceContextArtifactKind ||
			resourceValue.Definition.SchemaID != artifactbuiltin.WorkspaceContextSchemaID {
			continue
		}
		value, err := projectContextDocument(resourceValue)
		if err != nil {
			value.Diagnostics = providerapi.AppendDiagnostics(
				value.Diagnostics,
				contextProjectionDiagnostic(resourceValue.Artifact, err),
			)
		}
		output = append(output, value)
	}
	sort.Slice(output, func(left, right int) bool {
		leftOrder := contextRuntimeOrder(output[left].Locator)
		rightOrder := contextRuntimeOrder(output[right].Locator)
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		return output[left].Artifact.ArtifactID < output[right].Artifact.ArtifactID
	})
	return output, nil
}

func (p *Adapter) Load(
	ctx context.Context,
	workspace collection.CollectionRef,
	artifactRefs []artifact.ArtifactRef,
) (ContextInspection, error) {
	view, err := p.query.Catalog(ctx, workspace)
	if err != nil {
		return ContextInspection{}, err
	}
	requested := make(
		map[basespec.ArtifactID]struct{},
		len(artifactRefs),
	)
	for _, ref := range artifactRefs {
		if err := ref.Validate(); err != nil {
			return ContextInspection{}, err
		}
		if ref.RootID != workspace.RootID {
			return ContextInspection{}, fmt.Errorf(
				"%w: Context Artifact belongs to another Root",
				spec.ErrInvalidWorkspace,
			)
		}
		if _, duplicate := requested[ref.ArtifactID]; duplicate {
			return ContextInspection{}, fmt.Errorf(
				"%w: duplicate Context Artifact %q",
				spec.ErrInvalidWorkspace,
				ref.ArtifactID,
			)
		}
		requested[ref.ArtifactID] = struct{}{}
	}
	output := ContextInspection{
		Workspace:       workspace,
		CatalogRevision: view.Catalog.Revision,
	}
	for _, resourceValue := range view.Resources {
		if resourceValue.Definition.Kind != artifactbuiltin.WorkspaceContextArtifactKind ||
			resourceValue.Definition.SchemaID != artifactbuiltin.WorkspaceContextSchemaID {
			continue
		}
		if len(requested) != 0 {
			if _, selected := requested[resourceValue.Artifact.ID]; !selected {
				continue
			}
		}
		contribution, err := projectContext(resourceValue)
		if err != nil {
			output.Diagnostics = providerapi.AppendDiagnostics(
				output.Diagnostics,
				contextProjectionDiagnostic(resourceValue.Artifact, err),
			)
			continue
		}
		output.Contributions = append(output.Contributions, contribution)
		output.Diagnostics = providerapi.AppendDiagnostics(
			output.Diagnostics,
			resourceValue.Artifact.Diagnostics...,
		)
	}
	sortContextContributions(output.Contributions)
	if len(requested) != 0 &&
		len(output.Contributions) != len(requested) {
		output.Diagnostics = providerapi.AppendDiagnostics(
			output.Diagnostics,
			providerapi.Diagnostic{
				Severity: providerapi.DiagnosticError,
				Code:     artifactadapter.DiagnosticCodeArtifactUnresolved,
				Message:  "one or more requested Context Artifacts were not available for inspection",
			},
		)
	}
	return output, nil
}

func projectContextDocument(
	value spec.Resource,
) (ContextDocument, error) {
	runtimeDisabled, dataErr := artifactadapter.ArtifactRuntimeDisabled(value.Artifact)
	output := ContextDocument{
		Artifact:         value.Artifact.Ref(),
		ArtifactRevision: value.Artifact.Revision,
		DefinitionDigest: value.Definition.Digest,
		SourceID:         value.Source.ID,
		Locator:          value.Artifact.Binding.Locator,
		Name:             value.Artifact.Name,
		Enabled:          value.Artifact.Enabled,
		State:            value.Artifact.State,
		CatalogCurrent:   value.CatalogCurrent,
		RuntimeDisabled:  runtimeDisabled,
		Diagnostics: providerapi.AppendDiagnostics(
			value.Artifact.Diagnostics,
			value.Diagnostics...,
		),
	}
	if dataErr != nil {
		return output, dataErr
	}
	if err := ValidateContextDefinition(value.Definition); err != nil {
		return output, err
	}
	body, err := providerapi.DecodeBody[contextDefinition](
		value.Definition.Body,
	)
	if err != nil {
		return output, err
	}
	output.Name = body.Name
	output.Role = body.Role
	output.MediaType = body.MediaType
	output.ProjectionValid = true
	return output, nil
}

func projectContext(
	value spec.Resource,
) (ContextContribution, error) {
	if err := ValidateContextDefinition(value.Definition); err != nil {
		return ContextContribution{}, err
	}
	body, err := providerapi.DecodeBody[contextDefinition](value.Definition.Body)
	if err != nil {
		return ContextContribution{}, err
	}
	return ContextContribution{
		Artifact:         value.Artifact.Ref(),
		ArtifactRevision: value.Artifact.Revision,
		DefinitionDigest: value.Definition.Digest,
		SourceID:         value.Source.ID,
		Locator:          value.Artifact.Binding.Locator,
		ConventionOrder:  contextRuntimeOrder(value.Artifact.Binding.Locator),
		Name:             body.Name,
		Role:             body.Role,
		MediaType:        body.MediaType,
		Content:          body.Content,
	}, nil
}

func sortContextContributions(values []ContextContribution) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].ConventionOrder != values[right].ConventionOrder {
			return values[left].ConventionOrder <
				values[right].ConventionOrder
		}
		if values[left].SourceID != values[right].SourceID {
			return values[left].SourceID < values[right].SourceID
		}
		return values[left].Locator < values[right].Locator
	})
}

func contextRuntimeOrder(locator basespec.Locator) int {
	if convention, found := contextConventionFor(locator); found {
		return convention.RuntimeOrder
	}
	return 10_000
}

func contextProjectionDiagnostic(
	value artifact.Artifact,
	err error,
) providerapi.Diagnostic {
	return providerapi.Diagnostic{
		Severity: providerapi.DiagnosticError,
		Code:     artifactadapter.DiagnosticCodeProjectionInvalid,
		Message:  providerapi.BoundedDiagnosticMessage(err.Error()),
		Location: &providerapi.DiagnosticLocation{
			Locator:            value.Binding.Locator,
			SubresourceLocator: value.Binding.SubresourceLocator,
		},
	}
}
