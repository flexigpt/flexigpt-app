// Package workflow exposes read-only Workflow declaration planning.
//
// It does not create invocations or execute control flow.
package workflow

import (
	"context"
	"fmt"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

type Service struct {
	resolver *composition.Resolver
}

// BindingRequirement identifies one declared callable or capability
// requirement that needs binding preparation before use.
//
// UsesComposition reports whether the declaration also has a relationship in
// Capabilities. That relationship can resolve to an Artifact or an explicitly
// supplied direct capability. It does not establish runtime binding readiness.
type BindingRequirement struct {
	Path            string            `json:"path"`
	Target          workflowv1.Target `json:"target"`
	UsesComposition bool              `json:"usesComposition"`
}

type Plan struct {
	Workflow artifactModel.ArtifactRef `json:"workflow"`

	// Document is the canonical declaration. Dispatch-only apiVersion is not
	// included in an admitted Definition.
	Document workflowv1.WorkflowDocument `json:"document"`

	// Capabilities contains recursively resolved declaration relationships.
	Capabilities composition.CapabilityPlan `json:"capabilities"`

	// BindingRequirements inventories all declared targets and requirements,
	// including service and activity bindings and MCP Tool operation bindings.
	// Inclusion does not mean that a binding is unavailable.
	BindingRequirements []BindingRequirement `json:"bindingRequirements"`

	// CompositionComplete covers only relationships represented in
	// Capabilities. Direct capabilities count as available composition
	// results. This field does not establish binding readiness, authorization,
	// dynamic target availability, or runtime support.
	CompositionComplete bool `json:"compositionComplete"`
}

func NewService(resolver *composition.Resolver) (*Service, error) {
	if resolver == nil {
		return nil, fmt.Errorf(
			"%w: Workflow composition resolver is required",
			storeSpec.ErrInvalid,
		)
	}
	return &Service{resolver: resolver}, nil
}

func (s *Service) ResolvePlan(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (Plan, error) {
	if s == nil || s.resolver == nil {
		return Plan{}, storeSpec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return Plan{}, err
	}

	resolved, err := s.resolver.ResolveWorkflow(ctx, ref)
	if err != nil {
		return Plan{}, err
	}
	if resolved == nil ||
		resolved.Type != declaration.TypeWorkflow ||
		resolved.Definition == nil {
		return Plan{}, fmt.Errorf(
			"%w: Artifact did not resolve as a Workflow",
			storeSpec.ErrReferenceUnresolved,
		)
	}

	workflowRef, found := resolved.ArtifactRef()
	if !found {
		return Plan{}, fmt.Errorf(
			"%w: Workflow plan has no source-backed root",
			storeSpec.ErrReferenceUnresolved,
		)
	}

	document, err := workflowv1.FromDefinition(*resolved.Definition)
	if err != nil {
		return Plan{}, err
	}
	capabilities, err := composition.CapabilityPlanForResolvedEntry(resolved)
	if err != nil {
		return Plan{}, err
	}
	bindings, err := bindingRequirements(document)
	if err != nil {
		return Plan{}, err
	}

	return Plan{
		Workflow:            workflowRef,
		Document:            document,
		Capabilities:        capabilities,
		BindingRequirements: bindings,
		CompositionComplete: capabilities.Complete,
	}, nil
}

func bindingRequirements(
	document workflowv1.WorkflowDocument,
) ([]BindingRequirement, error) {
	groups := []struct {
		path   string
		values map[string]workflowv1.Target
	}{
		{
			path:   "targets",
			values: document.Targets,
		},
		{
			path:   "requirements",
			values: document.Requirements,
		},
	}

	output := make(
		[]BindingRequirement,
		0,
		len(document.Targets)+len(document.Requirements),
	)
	for _, group := range groups {
		names := make([]string, 0, len(group.values))
		for name := range group.values {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			target := group.values[name]
			_, usesComposition, err := target.DeclarationMember()
			if err != nil {
				return nil, fmt.Errorf(
					"workflow %s/%s: %w",
					group.path,
					name,
					err,
				)
			}

			// The plan's dependency inventory and document must not share
			// mutable pointers or slices with one another.
			owned, err := jsonutil.CloneJSON(target)
			if err != nil {
				return nil, fmt.Errorf(
					"clone Workflow %s/%s: %w",
					group.path,
					name,
					err,
				)
			}

			output = append(output, BindingRequirement{
				Path:            group.path + "/" + name,
				Target:          owned,
				UsesComposition: usesComposition,
			})
		}
	}
	return output, nil
}
