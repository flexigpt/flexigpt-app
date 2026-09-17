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

func (a *StoreAPI) ResolveWorkspaceCapabilities(
	ctx context.Context,
	ref WorkspaceRef,
) (WorkspaceCapabilityPlan, error) {
	_, plan, err := a.resolveWorkspaceCapabilities(ctx, ref)
	return plan, err
}

func (a *StoreAPI) resolveWorkspaceCapabilities(
	ctx context.Context,
	ref WorkspaceRef,
) (
	workspaceDomain.Workspace,
	WorkspaceCapabilityPlan,
	error,
) {
	if a == nil || a.resolver == nil {
		return workspaceDomain.Workspace{},
			WorkspaceCapabilityPlan{},
			basespec.ErrClosed
	}
	workspace, err := a.GetWorkspace(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{},
			WorkspaceCapabilityPlan{},
			err
	}
	plan, err := a.resolver.ResolveWorkspaceCapabilities(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{},
			WorkspaceCapabilityPlan{},
			err
	}

	output := WorkspaceCapabilityPlan{
		Workspace:       workspace.Ref(),
		Occurrences:     append([]resolve.CapabilityOccurrence(nil), plan.Occurrences...),
		PromptArtifacts: make([]artifact.ArtifactRef, 0),
		SkillArtifacts:  make([]artifact.ArtifactRef, 0),
		MCPArtifacts:    make([]artifact.ArtifactRef, 0),
		Complete:        plan.Complete,
	}
	promptSeen := make(map[artifact.ArtifactRef]struct{})
	skillSeen := make(map[artifact.ArtifactRef]struct{})
	mcpSeen := make(map[artifact.ArtifactRef]struct{})

	for _, occurrence := range output.Occurrences {
		if occurrence.Status != resolve.ResolutionAvailable ||
			occurrence.Artifact == nil {
			continue
		}

		switch occurrence.Type {
		case declaration.TypeText:
			appendWorkspaceArtifact(
				&output.PromptArtifacts,
				promptSeen,
				*occurrence.Artifact,
			)

		case declaration.TypeSkill:
			appendWorkspaceArtifact(
				&output.SkillArtifacts,
				skillSeen,
				*occurrence.Artifact,
			)

		case declaration.TypeMCP:
			appendWorkspaceArtifact(
				&output.MCPArtifacts,
				mcpSeen,
				*occurrence.Artifact,
			)
		default:
		}
	}

	sortWorkspaceArtifactRefs(output.PromptArtifacts)
	sortWorkspaceArtifactRefs(output.SkillArtifacts)
	sortWorkspaceArtifactRefs(output.MCPArtifacts)
	return workspace, output, nil
}

func requireCompleteWorkspaceCapabilities(
	value WorkspaceCapabilityPlan,
) error {
	return resolve.RequireComplete(value.Occurrences)
}

func appendWorkspaceArtifact(
	values *[]artifact.ArtifactRef,
	seen map[artifact.ArtifactRef]struct{},
	ref artifact.ArtifactRef,
) {
	if _, found := seen[ref]; found {
		return
	}
	seen[ref] = struct{}{}
	*values = append(*values, ref)
}

func sortWorkspaceArtifactRefs(values []artifact.ArtifactRef) {
	sort.Slice(values, func(left, right int) bool {
		leftKey := string(values[left].RootID) + "\x00" +
			string(values[left].ArtifactID)
		rightKey := string(values[right].RootID) + "\x00" +
			string(values[right].ArtifactID)
		return leftKey < rightKey
	})
}
