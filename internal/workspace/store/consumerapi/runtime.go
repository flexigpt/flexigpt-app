package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/mcp"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/prompt"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/skill"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

func (a *StoreAPI) ComposeWorkspacePrompt(
	ctx context.Context,
	workspace WorkspaceRef,
	artifacts []artifact.ArtifactRef,
) (prompt.Plan, error) {
	value, capabilities, err := a.resolveWorkspaceCapabilities(
		ctx,
		workspace,
	)
	if err != nil {
		return prompt.Plan{}, err
	}
	return a.promptAdapter.ComposeSelected(
		ctx,
		value,
		selectedWorkspaceArtifacts(
			artifacts,
			capabilities.PromptArtifacts,
		),
	)
}

func (a *StoreAPI) LoadWorkspaceSkills(
	ctx context.Context,
	workspace WorkspaceRef,
	artifacts []artifact.ArtifactRef,
) (skill.LoadPlan, error) {
	value, capabilities, err := a.resolveWorkspaceCapabilities(
		ctx,
		workspace,
	)
	if err != nil {
		return skill.LoadPlan{}, err
	}
	return a.skillAdapter.LoadSelected(
		ctx,
		value,
		selectedWorkspaceArtifacts(
			artifacts,
			capabilities.SkillArtifacts,
		),
	)
}

func (a *StoreAPI) ListWorkspaceSkills(
	ctx context.Context,
	workspace WorkspaceRef,
) ([]skill.WorkspaceSkill, error) {
	plan, err := a.LoadWorkspaceSkills(ctx, workspace, nil)
	if err != nil {
		return nil, err
	}
	return plan.Skills, nil
}

func (a *StoreAPI) LoadWorkspaceMCPServers(
	ctx context.Context,
	workspace WorkspaceRef,
	artifacts []artifact.ArtifactRef,
) (mcp.LoadPlan, error) {
	value, capabilities, err := a.resolveWorkspaceCapabilities(
		ctx,
		workspace,
	)
	if err != nil {
		return mcp.LoadPlan{}, err
	}
	return a.loadWorkspaceMCPServers(
		ctx,
		value,
		selectedWorkspaceArtifacts(
			artifacts,
			capabilities.MCPArtifacts,
		),
	)
}

func (a *StoreAPI) ResolveWorkspaceRuntimePlan(
	ctx context.Context,
	workspace WorkspaceRef,
	selection WorkspaceRuntimeSelection,
) (WorkspaceRuntimePlan, error) {
	value, capabilities, err := a.resolveWorkspaceCapabilities(
		ctx,
		workspace,
	)
	if err != nil {
		return WorkspaceRuntimePlan{}, err
	}

	promptPlan, err := a.promptAdapter.ComposeSelected(
		ctx,
		value,
		selectedWorkspaceArtifacts(
			selection.PromptArtifacts,
			capabilities.PromptArtifacts,
		),
	)
	if err != nil {
		return WorkspaceRuntimePlan{}, err
	}
	skillPlan, err := a.skillAdapter.LoadSelected(
		ctx,
		value,
		selectedWorkspaceArtifacts(
			selection.SkillArtifacts,
			capabilities.SkillArtifacts,
		),
	)
	if err != nil {
		return WorkspaceRuntimePlan{}, err
	}
	mcpPlan, err := a.loadWorkspaceMCPServers(
		ctx,
		value,
		selectedWorkspaceArtifacts(
			selection.MCPArtifacts,
			capabilities.MCPArtifacts,
		),
	)
	if err != nil {
		return WorkspaceRuntimePlan{}, err
	}

	return WorkspaceRuntimePlan{
		Workspace:    value,
		Capabilities: capabilities,
		Prompt:       promptPlan,
		Skills:       skillPlan,
		MCPServers:   mcpPlan,
	}, nil
}

func (a *StoreAPI) loadWorkspaceMCPServers(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifact.ArtifactRef,
) (mcp.LoadPlan, error) {
	if len(refs) == 0 {
		return mcp.LoadPlan{
			Workspace: workspace.Ref(),
			Servers:   []mcp.WorkspaceServer{},
		}, nil
	}
	if a == nil || a.mcpAdapter == nil {
		return mcp.LoadPlan{}, fmt.Errorf(
			"%w: Workspace MCP resolver is unavailable",
			basespec.ErrUnsupported,
		)
	}
	return a.mcpAdapter.Load(ctx, workspace, refs)
}

func selectedWorkspaceArtifacts(
	explicit []artifact.ArtifactRef,
	defaults []artifact.ArtifactRef,
) []artifact.ArtifactRef {
	if explicit != nil {
		return append([]artifact.ArtifactRef(nil), explicit...)
	}
	return append([]artifact.ArtifactRef(nil), defaults...)
}
