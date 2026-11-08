// Package workflow exposes read-only Workflow declaration planning.
//
// It does not create invocations or execute control flow.
package workflow

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

type Service struct {
	resolver *composition.Resolver
}

// BindingRequirement is retained as a compatibility alias. New consumers use
// Plan.Inventory.Targets, which also contains planning-owned resolution facts.
type BindingRequirement = TargetPlan

type Plan struct {
	Workflow artifactModel.ArtifactRef `json:"workflow"`
	Revision uint64                    `json:"revision"`
	Digest   cryptoutil.Digest         `json:"digest"`

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

	// Inventory exposes the typed declaration facts required for runtime
	// lowering and current binding selection. It is built by Workflow
	// planning, not rediscovered by setup admission.
	Inventory Inventory `json:"inventory"`
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
	if resolved.Artifact == nil || !resolved.Artifact.Enabled {
		return Plan{}, fmt.Errorf(
			"%w: selected Workflow is disabled",
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
	inventory, err := buildWorkflowInventory(document, capabilities)
	if err != nil {
		return Plan{}, err
	}

	return Plan{
		Workflow:            workflowRef,
		Revision:            resolved.Artifact.Revision,
		Digest:              resolved.Definition.Digest,
		Document:            document,
		Capabilities:        capabilities,
		CompositionComplete: capabilities.Complete,
		Inventory:           inventory,
		BindingRequirements: append([]BindingRequirement(nil), inventory.Targets...),
	}, nil
}
