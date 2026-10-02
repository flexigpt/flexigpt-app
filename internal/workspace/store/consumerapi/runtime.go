package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local/consumerutil"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/mcp"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/prompt"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/skill"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

func (a *StoreAPI) ComposeWorkspacePrompt(
	ctx context.Context,
	workspace artifact.ArtifactRef,
	artifacts []artifact.ArtifactRef,
) (WorkspacePromptPlan, error) {
	if a == nil || a.resources == nil {
		return WorkspacePromptPlan{}, spec.ErrClosed
	}
	return consumerutil.WithResourceVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (WorkspacePromptPlan, error) {
			value, err := a.composeWorkspacePrompt(
				sessionCtx,
				workspace,
				artifacts,
			)
			if err != nil {
				return WorkspacePromptPlan{}, err
			}
			return projectWorkspacePromptPlan(value), nil
		},
	)
}

func (a *StoreAPI) composeWorkspacePrompt(
	ctx context.Context,
	workspace artifact.ArtifactRef,
	artifacts []artifact.ArtifactRef,
) (prompt.Plan, error) {
	value, capabilities, err := a.resolveWorkspaceCapabilities(
		ctx,
		workspace,
	)
	if err != nil {
		return prompt.Plan{}, err
	}
	selected, err := selectedWorkspaceArtifacts(
		artifacts,
		workspaceArtifactRefs(
			capabilities,
			declaration.TypeText,
		),
	)
	if err != nil {
		return prompt.Plan{}, err
	}
	return a.promptAdapter.ComposeSelected(
		ctx,
		value,
		selected,
	)
}

func (a *StoreAPI) LoadWorkspaceSkills(
	ctx context.Context,
	workspace artifact.ArtifactRef,
	artifacts []artifact.ArtifactRef,
) (WorkspaceSkillLoadPlan, error) {
	if a == nil || a.resources == nil {
		return WorkspaceSkillLoadPlan{}, spec.ErrClosed
	}
	return consumerutil.WithResourceVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (WorkspaceSkillLoadPlan, error) {
			value, err := a.loadWorkspaceSkills(
				sessionCtx,
				workspace,
				artifacts,
			)
			if err != nil {
				return WorkspaceSkillLoadPlan{}, err
			}
			return projectWorkspaceSkillLoadPlan(value), nil
		},
	)
}

func (a *StoreAPI) loadWorkspaceSkills(
	ctx context.Context,
	workspace artifact.ArtifactRef,
	artifacts []artifact.ArtifactRef,
) (skill.LoadPlan, error) {
	value, capabilities, err := a.resolveWorkspaceCapabilities(
		ctx,
		workspace,
	)
	if err != nil {
		return skill.LoadPlan{}, err
	}
	selected, err := selectedWorkspaceArtifacts(
		artifacts,
		workspaceArtifactRefs(
			capabilities,
			declaration.TypeSkill,
		),
	)
	if err != nil {
		return skill.LoadPlan{}, err
	}
	return a.skillAdapter.LoadSelected(
		ctx,
		value,
		selected,
	)
}

func (a *StoreAPI) LoadWorkspaceMCPServers(
	ctx context.Context,
	workspace artifact.ArtifactRef,
	artifacts []artifact.ArtifactRef,
) (WorkspaceMCPServerLoadPlan, error) {
	if a == nil || a.resources == nil {
		return WorkspaceMCPServerLoadPlan{}, spec.ErrClosed
	}
	return consumerutil.WithResourceVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (WorkspaceMCPServerLoadPlan, error) {
			value, err := a.loadWorkspaceMCPServersForRuntime(
				sessionCtx,
				workspace,
				artifacts,
			)
			if err != nil {
				return WorkspaceMCPServerLoadPlan{}, err
			}
			return projectWorkspaceMCPServerLoadPlan(value), nil
		},
	)
}

func (a *StoreAPI) loadWorkspaceMCPServersForRuntime(
	ctx context.Context,
	workspace artifact.ArtifactRef,
	artifacts []artifact.ArtifactRef,
) (mcp.LoadPlan, error) {
	value, capabilities, err := a.resolveWorkspaceCapabilities(
		ctx,
		workspace,
	)
	if err != nil {
		return mcp.LoadPlan{}, err
	}
	selected, err := selectedWorkspaceArtifacts(
		artifacts,
		workspaceArtifactRefs(
			capabilities,
			declaration.TypeMCP,
		),
	)
	if err != nil {
		return mcp.LoadPlan{}, err
	}
	return a.loadWorkspaceMCPServers(
		ctx,
		value,
		selected,
	)
}

func (a *StoreAPI) ResolveWorkspaceRuntimePlan(
	ctx context.Context,
	workspace artifact.ArtifactRef,
	selection WorkspaceRuntimeSelection,
) (WorkspaceRuntimePlan, error) {
	if a == nil || a.resources == nil {
		return WorkspaceRuntimePlan{}, spec.ErrClosed
	}
	return consumerutil.WithResourceVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (WorkspaceRuntimePlan, error) {
			return a.resolveWorkspaceRuntimePlan(
				sessionCtx,
				workspace,
				selection,
			)
		},
	)
}

func (a *StoreAPI) resolveWorkspaceRuntimePlan(
	ctx context.Context,
	workspace artifact.ArtifactRef,
	selection WorkspaceRuntimeSelection,
) (WorkspaceRuntimePlan, error) {
	value, capabilities, err := a.resolveWorkspaceCapabilities(
		ctx,
		workspace,
	)
	if err != nil {
		return WorkspaceRuntimePlan{}, err
	}
	if selection.RequireComplete {
		if err := requireCompleteWorkspaceCapabilities(capabilities); err != nil {
			return WorkspaceRuntimePlan{}, err
		}
	}

	availablePromptArtifacts := workspaceArtifactRefs(
		capabilities,
		declaration.TypeText,
	)
	availableSkillArtifacts := workspaceArtifactRefs(
		capabilities,
		declaration.TypeSkill,
	)
	availableMCPArtifacts := workspaceArtifactRefs(
		capabilities,
		declaration.TypeMCP,
	)

	promptArtifacts, err := selectedWorkspaceArtifacts(
		selection.PromptArtifacts,
		availablePromptArtifacts,
	)
	if err != nil {
		return WorkspaceRuntimePlan{}, err
	}
	skillArtifacts, err := selectedWorkspaceArtifacts(
		selection.SkillArtifacts,
		availableSkillArtifacts,
	)
	if err != nil {
		return WorkspaceRuntimePlan{}, err
	}
	mcpArtifacts, err := selectedWorkspaceArtifacts(
		selection.MCPArtifacts,
		availableMCPArtifacts,
	)
	if err != nil {
		return WorkspaceRuntimePlan{}, err
	}

	promptPlan, err := a.promptAdapter.ComposeSelected(
		ctx,
		value,
		promptArtifacts,
	)
	if err != nil {
		return WorkspaceRuntimePlan{}, err
	}
	skillPlan, err := a.skillAdapter.LoadSelected(
		ctx,
		value,
		skillArtifacts,
	)
	if err != nil {
		return WorkspaceRuntimePlan{}, err
	}
	mcpPlan, err := a.loadWorkspaceMCPServers(
		ctx,
		value,
		mcpArtifacts,
	)
	if err != nil {
		return WorkspaceRuntimePlan{}, err
	}

	return projectWorkspaceRuntimePlan(
		value,
		capabilities,
		promptPlan,
		skillPlan,
		mcpPlan,
	), nil
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
			spec.ErrUnsupported,
		)
	}
	return a.mcpAdapter.Load(ctx, workspace, refs)
}

func selectedWorkspaceArtifacts(
	explicit []artifact.ArtifactRef,
	defaults []artifact.ArtifactRef,
) ([]artifact.ArtifactRef, error) {
	if explicit != nil {
		allowed := make(
			map[artifact.ArtifactRef]struct{},
			len(defaults),
		)
		for _, ref := range defaults {
			allowed[ref] = struct{}{}
		}

		output := make([]artifact.ArtifactRef, 0, len(explicit))
		seen := make(map[artifact.ArtifactRef]struct{}, len(explicit))
		for _, ref := range explicit {
			if err := ref.Validate(); err != nil {
				return nil, err
			}
			if _, found := allowed[ref]; !found {
				return nil, fmt.Errorf(
					"%w: Artifact %q is not a resolved Workspace capability",
					workspaceDomain.ErrReferenceUnresolved,
					ref.ArtifactID,
				)
			}
			if _, duplicate := seen[ref]; duplicate {
				continue
			}
			seen[ref] = struct{}{}
			output = append(output, ref)
		}
		return output, nil
	}
	return append([]artifact.ArtifactRef(nil), defaults...), nil
}
