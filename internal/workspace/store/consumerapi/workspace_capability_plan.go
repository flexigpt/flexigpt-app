package consumerapi

import (
	"context"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

// ResolveArtifactCapabilities exposes the complete generic declaration
// capability plan. It can resolve Plugin, Agent, Team, Loop, Workflow,
// Workspace, Skill, Tool, Model, and other contract declaration Artifacts.
func (a *StoreAPI) ResolveArtifactCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (resolve.CapabilityPlan, error) {
	if a == nil || a.resolver == nil {
		return resolve.CapabilityPlan{}, basespec.ErrClosed
	}
	return a.resolver.ResolveCapabilities(ctx, ref)
}

func (a *StoreAPI) ResolveWorkspaceCapabilities(
	ctx context.Context,
	ref WorkspaceRef,
) (resolve.CapabilityPlan, error) {
	_, plan, err := a.resolveWorkspaceCapabilities(ctx, ref)
	return plan, err
}

func (a *StoreAPI) resolveWorkspaceCapabilities(
	ctx context.Context,
	ref WorkspaceRef,
) (
	workspaceDomain.Workspace,
	resolve.CapabilityPlan,
	error,
) {
	workspace, resolved, err := a.resolveCurrentWorkspace(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{},
			resolve.CapabilityPlan{},
			err
	}
	plan, err := resolve.CapabilityPlanForResolvedEntry(resolved)
	if err != nil {
		return workspaceDomain.Workspace{},
			resolve.CapabilityPlan{},
			err
	}
	return workspace, plan, nil
}

func requireCompleteWorkspaceCapabilities(
	value resolve.CapabilityPlan,
) error {
	return resolve.RequireComplete(value.Occurrences)
}

// workspaceArtifactRefs returns the available Artifact-backed capabilities of
// one type. Mapped targets intentionally remain in CapabilityPlan.Occurrences
// and are translated by ToolStore or ModelPresetStore consumers.
func workspaceArtifactRefs(
	plan resolve.CapabilityPlan,
	declarationType declaration.Type,
) []artifact.ArtifactRef {
	output := make([]artifact.ArtifactRef, 0)
	seen := make(map[artifact.ArtifactRef]struct{})

	for _, occurrence := range plan.Occurrences {
		if occurrence.Status != resolve.ResolutionAvailable ||
			occurrence.Type != declarationType ||
			occurrence.Artifact == nil {
			continue
		}
		if _, duplicate := seen[*occurrence.Artifact]; duplicate {
			continue
		}
		seen[*occurrence.Artifact] = struct{}{}
		output = append(output, *occurrence.Artifact)
	}

	sort.Slice(output, func(left, right int) bool {
		leftKey := string(output[left].RootID) + "\x00" +
			string(output[left].ArtifactID)
		rightKey := string(output[right].RootID) + "\x00" +
			string(output[right].ArtifactID)
		return leftKey < rightKey
	})
	return output
}
