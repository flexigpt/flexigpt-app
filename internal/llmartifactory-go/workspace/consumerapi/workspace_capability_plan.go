package consumerapi

import (
	"context"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
)

func (a *StoreAPI) ResolveWorkspaceCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (composition.CapabilityPlan, error) {
	_, plan, err := a.resolveWorkspaceCapabilities(ctx, ref)
	return plan, err
}

func (a *StoreAPI) resolveWorkspaceCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (
	workspaceDomain.Workspace,
	composition.CapabilityPlan,
	error,
) {
	workspace, resolved, err := a.resolveCurrentWorkspace(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{},
			composition.CapabilityPlan{},
			err
	}
	plan, err := composition.CapabilityPlanForResolvedEntry(resolved)
	if err != nil {
		return workspaceDomain.Workspace{},
			composition.CapabilityPlan{},
			err
	}
	return workspace, plan, nil
}

func requireCompleteWorkspaceCapabilities(
	value composition.CapabilityPlan,
) error {
	return composition.RequireComplete(value.Occurrences)
}

// workspaceArtifactRefs returns the available Artifact-backed capabilities of
// one type. Mapped targets intentionally remain in CapabilityPlan.Occurrences
// and are translated by ToolStore or ModelStore consumers.
func workspaceArtifactRefs(
	plan composition.CapabilityPlan,
	declarationType declaration.Type,
) []artifactModel.ArtifactRef {
	output := make([]artifactModel.ArtifactRef, 0)
	seen := make(map[artifactModel.ArtifactRef]struct{})

	for _, occurrence := range plan.Occurrences {
		if occurrence.Status != composition.ResolutionAvailable ||
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
