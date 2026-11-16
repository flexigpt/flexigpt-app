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

type Plan struct {
	Workflow artifactModel.ArtifactRef `json:"workflow"`
	Revision uint64                    `json:"revision"`
	Digest   cryptoutil.Digest         `json:"digest"`

	// Document is the canonical declaration. Dispatch-only apiVersion is not
	// included in an admitted Definition.
	Document workflowv1.WorkflowDocument `json:"document"`

	// Capabilities contains recursively resolved declaration relationships.
	Capabilities composition.CapabilityPlan `json:"capabilities"`

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

// ResolvePlan reads an available Workflow Artifact and its current admitted
// declaration plan. Local Artifact enablement is an execution-admission
// concern owned by downstream runtime setup.
func (s *Service) ResolvePlan(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (Plan, error) {
	if s == nil || s.resolver == nil {
		return Plan{}, storeSpec.ErrClosed
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
	if resolved.Artifact == nil {
		return Plan{}, fmt.Errorf(
			"%w: selected Workflow has no source-backed Artifact",
			storeSpec.ErrReferenceUnresolved,
		)
	}

	entry, found := resolved.AdmittedEntry()
	if !found {
		return Plan{}, fmt.Errorf(
			"%w: Workflow plan has no admitted declaration entry",
			storeSpec.ErrReferenceUnresolved,
		)
	}
	document, err := workflowv1.DecodeResolvedWorkflowDefinition(
		*resolved.Definition,
		entry,
	)
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
	}, nil
}
