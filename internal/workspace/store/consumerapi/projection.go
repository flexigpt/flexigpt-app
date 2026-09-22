package consumerapi

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/mcp"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/prompt"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/skill"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

func projectWorkspacePromptPlan(
	value prompt.Plan,
) WorkspacePromptPlan {
	output := WorkspacePromptPlan{
		Workspace:     value.Workspace,
		Contributions: make([]WorkspacePromptContribution, 0, len(value.Contributions)),
		Instructions:  value.Instructions,
		UserMessage:   value.UserMessage,
		Diagnostics:   diagnostic.Clone(value.Diagnostics),
		Decisions:     make([]WorkspacePromptDecision, 0, len(value.Decisions)),
	}
	for _, contribution := range value.Contributions {
		output.Contributions = append(
			output.Contributions,
			WorkspacePromptContribution{
				Artifact:         contribution.Artifact,
				ArtifactRevision: contribution.ArtifactRevision,
				DefinitionDigest: contribution.DefinitionDigest,
				Kind:             contribution.Kind,
				Name:             contribution.Name,
				Insert:           contribution.Insert,
				MediaType:        contribution.MediaType,
				Locator:          contribution.Locator,
				OriginalBytes:    contribution.OriginalBytes,
				IncludedBytes:    contribution.IncludedBytes,
				Truncated:        contribution.Truncated,
			},
		)
	}
	for _, decision := range value.Decisions {
		output.Decisions = append(
			output.Decisions,
			WorkspacePromptDecision{
				Artifact:      decision.Artifact,
				Status:        decision.Status,
				Code:          decision.Code,
				OriginalBytes: decision.OriginalBytes,
				IncludedBytes: decision.IncludedBytes,
			},
		)
	}
	return output
}

func projectWorkspaceSkillLoadPlan(
	value skill.LoadPlan,
) WorkspaceSkillLoadPlan {
	output := WorkspaceSkillLoadPlan{
		Workspace: value.Workspace,
		Skills:    make([]WorkspaceSkill, 0, len(value.Skills)),
	}
	for _, skill := range value.Skills {
		output.Skills = append(output.Skills, WorkspaceSkill{
			Artifact:         skill.Artifact,
			ArtifactRevision: skill.ArtifactRevision,
			DefinitionDigest: skill.DefinitionDigest,
			Name:             skill.Document.Name,
			DisplayName:      skill.Document.DisplayName,
			Locator:          skill.Locator,
			Version:          skill.Version,
		})
	}
	return output
}

func projectWorkspaceMCPServerLoadPlan(
	value mcp.LoadPlan,
) WorkspaceMCPServerLoadPlan {
	output := WorkspaceMCPServerLoadPlan{
		Workspace: value.Workspace,
		Servers:   make([]WorkspaceMCPServer, 0, len(value.Servers)),
	}
	for _, server := range value.Servers {
		output.Servers = append(output.Servers, WorkspaceMCPServer{
			Artifact:         server.Artifact,
			ArtifactRevision: server.Server.ArtifactRevision,
			DefinitionDigest: server.Server.DefinitionDigest,
			Name:             server.Server.Document.LogicalName,
			DisplayName:      server.Server.Document.DisplayName,
			BuiltIn:          server.Server.BuiltIn,
			Version:          server.Server.Version,
		})
	}
	return output
}

func projectWorkspaceRuntimePlan(
	workspace domain.Workspace,
	capabilities resolve.CapabilityPlan,
	p prompt.Plan,
	skills skill.LoadPlan,
	servers mcp.LoadPlan,
) WorkspaceRuntimePlan {
	return WorkspaceRuntimePlan{
		Workspace:    workspace.View(),
		Capabilities: capabilities,
		Prompt:       projectWorkspacePromptPlan(p),
		Skills:       projectWorkspaceSkillLoadPlan(skills),
		MCPServers:   projectWorkspaceMCPServerLoadPlan(servers),
	}
}
