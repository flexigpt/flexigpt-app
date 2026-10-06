package workspace

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	workspacemcp "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/mcp"
	workspaceprompt "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/prompt"
	workspaceskill "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/skill"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
)

func (a *Service) ComposeWorkspacePrompt(
	ctx context.Context,
	workspace artifactModel.ArtifactRef,
	artifacts []artifactModel.ArtifactRef,
) (WorkspacePromptPlan, error) {
	if a == nil || a.resources == nil {
		return WorkspacePromptPlan{}, spec.ErrClosed
	}
	return resourceFlow.WithVerificationSession(
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

func (a *Service) composeWorkspacePrompt(
	ctx context.Context,
	workspace artifactModel.ArtifactRef,
	artifacts []artifactModel.ArtifactRef,
) (workspaceprompt.Plan, error) {
	value, capabilities, err := a.resolveWorkspaceCapabilities(
		ctx,
		workspace,
	)
	if err != nil {
		return workspaceprompt.Plan{}, err
	}
	selected, err := selectedWorkspaceArtifacts(
		artifacts,
		workspaceArtifactRefs(
			capabilities,
			declaration.TypeText,
		),
	)
	if err != nil {
		return workspaceprompt.Plan{}, err
	}
	return a.promptAdapter.ComposeSelected(
		ctx,
		value,
		selected,
	)
}

func (a *Service) LoadWorkspaceSkills(
	ctx context.Context,
	workspace artifactModel.ArtifactRef,
	artifacts []artifactModel.ArtifactRef,
) (WorkspaceSkillLoadPlan, error) {
	if a == nil || a.resources == nil {
		return WorkspaceSkillLoadPlan{}, spec.ErrClosed
	}
	return resourceFlow.WithVerificationSession(
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

func (a *Service) loadWorkspaceSkills(
	ctx context.Context,
	workspace artifactModel.ArtifactRef,
	artifacts []artifactModel.ArtifactRef,
) (workspaceskill.LoadPlan, error) {
	value, capabilities, err := a.resolveWorkspaceCapabilities(
		ctx,
		workspace,
	)
	if err != nil {
		return workspaceskill.LoadPlan{}, err
	}
	selected, err := selectedWorkspaceArtifacts(
		artifacts,
		workspaceArtifactRefs(
			capabilities,
			declaration.TypeSkill,
		),
	)
	if err != nil {
		return workspaceskill.LoadPlan{}, err
	}
	return a.skillAdapter.LoadSelected(
		ctx,
		value,
		selected,
	)
}

func (a *Service) LoadWorkspaceMCPServers(
	ctx context.Context,
	workspace artifactModel.ArtifactRef,
	artifacts []artifactModel.ArtifactRef,
) (WorkspaceMCPServerLoadPlan, error) {
	if a == nil || a.resources == nil {
		return WorkspaceMCPServerLoadPlan{}, spec.ErrClosed
	}
	return resourceFlow.WithVerificationSession(
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

func (a *Service) loadWorkspaceMCPServersForRuntime(
	ctx context.Context,
	workspace artifactModel.ArtifactRef,
	artifacts []artifactModel.ArtifactRef,
) (workspacemcp.LoadPlan, error) {
	value, capabilities, err := a.resolveWorkspaceCapabilities(
		ctx,
		workspace,
	)
	if err != nil {
		return workspacemcp.LoadPlan{}, err
	}
	selected, err := selectedWorkspaceArtifacts(
		artifacts,
		workspaceArtifactRefs(
			capabilities,
			declaration.TypeMCP,
		),
	)
	if err != nil {
		return workspacemcp.LoadPlan{}, err
	}
	return a.loadWorkspaceMCPServers(
		ctx,
		value,
		selected,
	)
}

func (a *Service) ResolveWorkspaceRuntimePlan(
	ctx context.Context,
	workspace artifactModel.ArtifactRef,
	selection WorkspaceRuntimeSelection,
) (WorkspaceRuntimePlan, error) {
	if a == nil || a.resources == nil {
		return WorkspaceRuntimePlan{}, spec.ErrClosed
	}
	return resourceFlow.WithVerificationSession(
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

func (a *Service) resolveWorkspaceRuntimePlan(
	ctx context.Context,
	workspace artifactModel.ArtifactRef,
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

func (a *Service) loadWorkspaceMCPServers(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifactModel.ArtifactRef,
) (workspacemcp.LoadPlan, error) {
	if len(refs) == 0 {
		return workspacemcp.LoadPlan{
			Workspace: workspace.Ref(),
			Servers:   []workspacemcp.WorkspaceServer{},
		}, nil
	}
	if a == nil || a.mcpAdapter == nil {
		return workspacemcp.LoadPlan{}, fmt.Errorf(
			"%w: Workspace MCP resolver is unavailable",
			spec.ErrUnsupported,
		)
	}
	return a.mcpAdapter.Load(ctx, workspace, refs)
}

func selectedWorkspaceArtifacts(
	explicit []artifactModel.ArtifactRef,
	defaults []artifactModel.ArtifactRef,
) ([]artifactModel.ArtifactRef, error) {
	if explicit != nil {
		allowed := make(
			map[artifactModel.ArtifactRef]struct{},
			len(defaults),
		)
		for _, ref := range defaults {
			allowed[ref] = struct{}{}
		}

		output := make([]artifactModel.ArtifactRef, 0, len(explicit))
		seen := make(map[artifactModel.ArtifactRef]struct{}, len(explicit))
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
	return append([]artifactModel.ArtifactRef(nil), defaults...), nil
}
