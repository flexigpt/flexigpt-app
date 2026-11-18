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

const workflowPlanStabilityPasses = 3

type Plan struct {
	Workflow artifactModel.ArtifactRef `json:"workflow"`
	Revision uint64                    `json:"revision"`

	// DefinitionDigest is the current admitted root Workflow Definition
	// digest. It is a source fact, not a runtime Artifact pin.
	DefinitionDigest cryptoutil.Digest `json:"definitionDigest"`

	// Digest remains available for existing planning consumers. It is always
	// identical to DefinitionDigest.
	Digest cryptoutil.Digest `json:"digest"`

	// Document is the canonical declaration. Dispatch-only apiVersion is not
	// included in an admitted Definition.
	Document workflowv1.WorkflowDocument `json:"document"`

	// Closure is the deterministic complete static source-backed Artifact
	// closure. It contains no setup or runtime types.
	Closure ArtifactClosure `json:"closure"`

	// Capabilities contains recursively resolved declaration relationships.
	// It remains diagnostic composition information. Runtime setup must use
	// Inventory rather than rebuilding dispatch lookup from this value.
	Capabilities composition.CapabilityPlan `json:"capabilities"`

	// CompositionComplete covers generic source relationship resolution.
	// Workflow-relevant direct composition results are rejected during closure
	// planning and cannot be lowered to runtime state. This field does not
	// establish binding readiness, authorization, or runtime support.
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

	var previous cryptoutil.Digest
	for attempt := range workflowPlanStabilityPasses {
		snapshot, err := s.resolvePlanSnapshot(ctx, ref)
		if err != nil {
			return Plan{}, err
		}
		fingerprint, err := planStabilityFingerprint(snapshot)
		if err != nil {
			return Plan{}, err
		}
		if attempt != 0 && fingerprint == previous {
			return snapshot.Clone()
		}
		previous = fingerprint
	}

	return Plan{}, &PlanRefreshError{
		Reason: "Workflow Artifact closure did not stabilize during plan construction",
	}
}

func (s *Service) resolvePlanSnapshot(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (Plan, error) {
	resolved, err := s.resolver.ResolveWorkflow(ctx, ref)
	if err != nil {
		return Plan{}, err
	}
	if resolved == nil ||
		resolved.Type != declaration.TypeWorkflow ||
		resolved.Definition == nil ||
		resolved.Artifact == nil {
		return Plan{}, fmt.Errorf(
			"%w: Artifact did not resolve as a source-backed Workflow",
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

	closure, err := buildArtifactClosure(resolved)
	if err != nil {
		return Plan{}, err
	}
	root, found := closure.rootNode()
	if !found || root.Workflow == nil {
		return Plan{}, &PlanConflictError{
			Artifact: workflowRef,
			Reason:   "Workflow closure root has no typed admitted Workflow facts",
		}
	}
	if root.Artifact != workflowRef {
		return Plan{}, &PlanConflictError{
			Artifact: workflowRef,
			Reason:   "Workflow closure root differs from selected Workflow",
		}
	}
	if root.Revision != resolved.Artifact.Revision ||
		root.DefinitionDigest != resolved.Definition.Digest {
		return Plan{}, &PlanRefreshError{
			Artifact: workflowRef,
			Reason:   "Workflow root changed during closure construction",
		}
	}

	return Plan{
		Workflow:            workflowRef,
		Revision:            root.Revision,
		DefinitionDigest:    root.DefinitionDigest,
		Digest:              root.DefinitionDigest,
		Document:            root.Workflow.Document,
		Closure:             closure,
		Capabilities:        root.Workflow.Capabilities,
		CompositionComplete: root.Workflow.CompositionComplete,
		Inventory:           root.Workflow.Inventory,
	}, nil
}

func planStabilityFingerprint(
	plan Plan,
) (cryptoutil.Digest, error) {
	return cryptoutil.CanonicalDigest(struct {
		Workflow            artifactModel.ArtifactRef   `json:"workflow"`
		Revision            uint64                      `json:"revision"`
		DefinitionDigest    cryptoutil.Digest           `json:"definitionDigest"`
		Digest              cryptoutil.Digest           `json:"digest"`
		Document            workflowv1.WorkflowDocument `json:"document"`
		Closure             ArtifactClosure             `json:"closure"`
		Capabilities        composition.CapabilityPlan  `json:"capabilities"`
		CompositionComplete bool                        `json:"compositionComplete"`
		Inventory           Inventory                   `json:"inventory"`
	}{
		Workflow:            plan.Workflow,
		Revision:            plan.Revision,
		DefinitionDigest:    plan.DefinitionDigest,
		Digest:              plan.Digest,
		Document:            plan.Document,
		Closure:             plan.Closure,
		Capabilities:        plan.Capabilities,
		CompositionComplete: plan.CompositionComplete,
		Inventory:           plan.Inventory,
	})
}
