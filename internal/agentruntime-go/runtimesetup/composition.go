package runtimesetup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

const kindWorkspaceContext = "workspaceContext"

func (c *coordinator) resolveAgent(
	ctx context.Context,
	selection *sessionRuntime.ArtifactSelection,
	plan *operationPlan,
) error {
	if selection == nil {
		return nil
	}

	ref, err := artifactRef(*selection)
	if err != nil {
		return operationPreparationError(
			"agent_selection_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}

	resolution, err := c.agents.ResolveAgent(ctx, ref)
	if err != nil {
		return operationPreparationError(
			"agent_unavailable",
			err.Error(),
			plan.summary,
			err,
		)
	}

	used := resolvedArtifact(
		resolution.Agent.Ref,
		resolution.Agent.Revision,
		resolution.Agent.DefinitionDigest,
	)
	if err := requireSelectionPin(*selection, used); err != nil {
		plan.summary.Agent = &sessionRuntime.ResolutionItem{
			Key:      selection.CanonicalKey(),
			Kind:     "agent",
			Owner:    sessionRuntime.ConfigurationOwnerSession,
			State:    sessionRuntime.ResolutionStateStale,
			Selected: selection,
			Used:     used,
			Reason:   err.Error(),
		}
		return operationPreparationError(
			"agent_selection_stale",
			err.Error(),
			plan.summary,
			err,
		)
	}

	plan.summary.Agent = &sessionRuntime.ResolutionItem{
		Key:      selection.CanonicalKey(),
		Kind:     "agent",
		Owner:    sessionRuntime.ConfigurationOwnerSession,
		State:    sessionRuntime.ResolutionStateIncluded,
		Selected: selection,
		Used:     used,
	}

	for _, occurrence := range resolution.Capabilities.Occurrences {
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Target == nil {
			diagnostic := agentOccurrenceDiagnostic(occurrence)
			plan.summary.Diagnostics = append(
				plan.summary.Diagnostics,
				diagnostic,
			)
			if occurrence.Required {
				return operationPreparationError(
					"agent_required_capability_unavailable",
					diagnostic.Message,
					plan.summary,
					nil,
				)
			}
			continue
		}

		switch occurrence.Type {
		case declaration.TypeModel:
			if plan.model != nil {
				plan.summary.Diagnostics = append(
					plan.summary.Diagnostics,
					sessionRuntime.ResolutionDiagnostic{
						Severity: sessionRuntime.DiagnosticSeverityWarning,
						Code:     "agent_multiple_models",
						Message: "Agent contributes more than one Model; " +
							"later Model contributions are ignored",
						Key: occurrence.Path,
					},
				)
				continue
			}

			value, found := selectionFromTarget(occurrence.Target)
			if !found {
				plan.summary.Diagnostics = append(
					plan.summary.Diagnostics,
					unavailableTargetDiagnostic(
						occurrence.Path,
						"Agent Model is not Artifact-backed",
					),
				)
				continue
			}
			plan.model = value
			plan.modelOwner = sessionRuntime.ConfigurationOwnerAgent
			if occurrence.IncludeSystemPrompt != nil {
				includeSystemPrompt := *occurrence.IncludeSystemPrompt
				plan.includeModelSystemPrompt = &includeSystemPrompt
			}

		case declaration.TypeText:
			value, found := selectionFromTarget(occurrence.Target)
			if !found {
				plan.summary.Diagnostics = append(
					plan.summary.Diagnostics,
					unavailableTargetDiagnostic(
						occurrence.Path,
						"Agent instruction is not Artifact-backed",
					),
				)
				continue
			}
			if err := c.appendArtifactInstruction(
				ctx,
				plan,
				*value,
				sessionRuntime.ConfigurationOwnerAgent,
				value.CanonicalKey(),
				occurrence.Required,
			); err != nil {
				return err
			}

		case declaration.TypeTool:
			target := occurrence.Target.Clone()
			plan.tools = append(plan.tools, toolContribution{
				key:         capabilityTargetKey(target),
				owner:       sessionRuntime.ConfigurationOwnerAgent,
				target:      &target,
				autoExecute: occurrence.AutoExecute,
				required:    occurrence.Required,
				item: sessionRuntime.ResolutionItem{
					Key:   capabilityTargetKey(target),
					Kind:  resolutionItemKindTool,
					Owner: sessionRuntime.ConfigurationOwnerAgent,
					State: sessionRuntime.ResolutionStateIncluded,
				},
			})

		case declaration.TypeSkill:
			value, found := selectionFromTarget(occurrence.Target)
			if !found {
				plan.summary.Diagnostics = append(
					plan.summary.Diagnostics,
					unavailableTargetDiagnostic(
						occurrence.Path,
						"Agent Skill is not Artifact-backed",
					),
				)
				continue
			}

			key := value.CanonicalKey()
			plan.skills = append(plan.skills, skillContribution{
				key:         key,
				owner:       sessionRuntime.ConfigurationOwnerAgent,
				selection:   *value,
				instruction: occurrence.SkillUseMode == agentAPI.AgentSkillUseModeInstructions,
				item: sessionRuntime.ResolutionItem{
					Key:      key,
					Kind:     resolutionItemKindSkill,
					Owner:    sessionRuntime.ConfigurationOwnerAgent,
					State:    sessionRuntime.ResolutionStateIncluded,
					Selected: value,
				},
				required: occurrence.Required,
			})
			if occurrence.SkillUseMode == agentAPI.AgentSkillUseModeActive {
				plan.activeSkills[key] = struct{}{}
			}

		case declaration.TypeMCP:
			selection, found := selectionFromTarget(occurrence.Target)
			if !found {
				diagnostic := unavailableTargetDiagnostic(
					occurrence.Path,
					"Agent MCP capability is not Artifact-backed",
				)
				if occurrence.Required {
					diagnostic.Severity = sessionRuntime.DiagnosticSeverityError
				}
				plan.summary.Diagnostics = append(plan.summary.Diagnostics, diagnostic)
				if occurrence.Required {
					return operationPreparationError(
						"agent_required_mcp_unavailable",
						diagnostic.Message,
						plan.summary,
						nil,
					)
				}
				continue
			}

			used, resolveErr := c.resolveArtifactSelection(ctx, *selection)
			if resolveErr != nil {
				diagnostic := sessionRuntime.ResolutionDiagnostic{
					Severity: sessionRuntime.DiagnosticSeverityWarning,
					Code:     "agent_mcp_unavailable",
					Message:  resolveErr.Error(),
					Key:      selection.CanonicalKey(),
				}
				if occurrence.Required {
					diagnostic.Severity = sessionRuntime.DiagnosticSeverityError
				}
				plan.summary.Diagnostics = append(plan.summary.Diagnostics, diagnostic)
				plan.summary.MCP = append(
					plan.summary.MCP,
					sessionRuntime.ResolutionItem{
						Key:      selection.CanonicalKey(),
						Kind:     resolutionItemKindMCP,
						Owner:    sessionRuntime.ConfigurationOwnerAgent,
						State:    sessionRuntime.ResolutionStateUnavailable,
						Selected: selection,
						Reason:   resolveErr.Error(),
					},
				)
				if occurrence.Required {
					return operationPreparationError(
						"agent_required_mcp_unavailable",
						resolveErr.Error(),
						plan.summary,
						resolveErr,
					)
				}
				continue
			}

			if plan.mcpUses == nil {
				plan.mcpUses = make(map[string]*sessionRuntime.ResolvedArtifact)
			}
			if plan.mcpOwners == nil {
				plan.mcpOwners = make(map[string]sessionRuntime.ConfigurationOwner)
			}
			if plan.mcpRequired == nil {
				plan.mcpRequired = make(map[string]bool)
			}

			key := selection.CanonicalKey()
			plan.mcp = append(plan.mcp, sessionRuntime.MCPServerSelection{
				Server:                    *selection,
				ToolExposure:              sessionRuntime.MCPToolExposureAll,
				IncludeServerInstructions: true,
			})
			plan.mcpUses[key] = used
			plan.mcpOwners[key] = sessionRuntime.ConfigurationOwnerAgent
			if occurrence.Required {
				plan.mcpRequired[key] = true
			}

		case declaration.TypeModelProvider,
			declaration.TypeMCPPolicy,
			declaration.TypePlugin,
			declaration.TypeAgent,
			declaration.TypeTeam,
			declaration.TypeLoop,
			declaration.TypeWorkflow,
			declaration.TypeWorkspace:
			// These declaration families do not contribute executable Agent
			// operation inputs.
			continue
		}
	}

	return nil
}

func (c *coordinator) resolveWorkspace(
	ctx context.Context,
	selection *sessionRuntime.WorkspaceSelectionIntent,
	plan *operationPlan,
) error {
	if selection == nil {
		return nil
	}

	workspaceRef, err := artifactRef(selection.Workspace)
	if err != nil {
		return operationPreparationError(
			"workspace_selection_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}

	capabilities, err := c.workspaces.ResolveWorkspaceCapabilities(
		ctx,
		workspaceRef,
	)
	if err != nil {
		return operationPreparationError(
			"workspace_unavailable",
			err.Error(),
			plan.summary,
			err,
		)
	}

	contextSelections, err := applyArtifactOverrideSelections(
		artifactSelectionsForCapabilities(
			capabilities,
			declaration.TypeText,
		),
		selection.Context,
	)
	if err != nil {
		return operationPreparationError(
			"workspace_context_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}

	skillSelections, err := applyArtifactOverrideSelections(
		artifactSelectionsForCapabilities(
			capabilities,
			declaration.TypeSkill,
		),
		selection.Skills,
	)
	if err != nil {
		return operationPreparationError(
			"workspace_skill_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}

	mcpSelections, err := workspaceMCPSelections(
		artifactSelectionsForCapabilities(
			capabilities,
			declaration.TypeMCP,
		),
		selection.MCP,
	)
	if err != nil {
		return operationPreparationError(
			"workspace_mcp_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}

	contextRefs, err := artifactRefs(contextSelections)
	if err != nil {
		return operationPreparationError(
			"workspace_context_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}
	skillRefs, err := artifactRefs(skillSelections)
	if err != nil {
		return operationPreparationError(
			"workspace_skill_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}

	mcpRefs := make([]artifactModel.ArtifactRef, 0, len(mcpSelections))
	for _, value := range mcpSelections {
		ref, err := artifactRef(value.Server)
		if err != nil {
			return operationPreparationError(
				"workspace_mcp_invalid",
				err.Error(),
				plan.summary,
				err,
			)
		}
		mcpRefs = append(mcpRefs, ref)
	}

	runtimePlan, err := c.workspaces.ResolveWorkspaceRuntimePlan(
		ctx,
		workspaceRef,
		workspaceAPI.WorkspaceRuntimeSelection{
			PromptArtifacts: contextRefs,
			SkillArtifacts:  skillRefs,
			MCPArtifacts:    mcpRefs,
		},
	)
	if err != nil {
		return operationPreparationError(
			"workspace_unavailable",
			err.Error(),
			plan.summary,
			err,
		)
	}

	usedWorkspace := resolvedArtifact(
		runtimePlan.Workspace.Artifact.Ref(),
		runtimePlan.Workspace.Artifact.Revision,
		digestFromArtifact(runtimePlan.Workspace.Artifact),
	)
	if err := requireSelectionPin(
		selection.Workspace,
		usedWorkspace,
	); err != nil {
		plan.summary.Workspace = &sessionRuntime.ResolutionItem{
			Key:      selection.Workspace.CanonicalKey(),
			Kind:     "workspace",
			Owner:    sessionRuntime.ConfigurationOwnerSession,
			State:    sessionRuntime.ResolutionStateStale,
			Selected: &selection.Workspace,
			Used:     usedWorkspace,
			Reason:   err.Error(),
		}
		return operationPreparationError(
			"workspace_selection_stale",
			err.Error(),
			plan.summary,
			err,
		)
	}

	plan.summary.Workspace = &sessionRuntime.ResolutionItem{
		Key:      selection.Workspace.CanonicalKey(),
		Kind:     "workspace",
		Owner:    sessionRuntime.ConfigurationOwnerSession,
		State:    sessionRuntime.ResolutionStateIncluded,
		Selected: &selection.Workspace,
		Used:     usedWorkspace,
	}

	if strings.TrimSpace(runtimePlan.Prompt.Instructions) != "" {
		plan.instructions = append(
			plan.instructions,
			instructionContribution{
				key: "workspace:" +
					selection.Workspace.CanonicalKey() +
					":instructions",
				owner: sessionRuntime.ConfigurationOwnerWorkspace,
				label: runtimePlan.Workspace.Artifact.DisplayName,
				text:  runtimePlan.Prompt.Instructions,
				item: sessionRuntime.ResolutionItem{
					Key: "workspace:" +
						selection.Workspace.CanonicalKey() +
						":instructions",
					Kind:  resolutionItemKindInstruction,
					Owner: sessionRuntime.ConfigurationOwnerWorkspace,
					State: sessionRuntime.ResolutionStateIncluded,
				},
			},
		)
	}

	if strings.TrimSpace(runtimePlan.Prompt.UserMessage) != "" {
		plan.userContexts = append(
			plan.userContexts,
			runtimePlan.Prompt.UserMessage,
		)
	}

	contextByRef := make(
		map[string]sessionRuntime.ArtifactSelection,
		len(contextSelections),
	)
	for _, value := range contextSelections {
		contextByRef[value.CanonicalKey()] = value
	}

	includedWorkspaceContexts := make(map[string]struct{}, len(runtimePlan.Prompt.Contributions))
	for _, contribution := range runtimePlan.Prompt.Contributions {
		key := artifactKey(contribution.Artifact)
		selected, found := contextByRef[key]
		if !found {
			continue
		}

		used := resolvedArtifact(
			contribution.Artifact,
			contribution.ArtifactRevision,
			contribution.DefinitionDigest,
		)
		if err := requireSelectionPin(selected, used); err != nil {
			selectedCopy := selected
			plan.summary.Instructions = append(
				plan.summary.Instructions,
				sessionRuntime.ResolutionItem{
					Key:      key,
					Kind:     kindWorkspaceContext,
					Owner:    sessionRuntime.ConfigurationOwnerWorkspace,
					State:    sessionRuntime.ResolutionStateStale,
					Selected: &selectedCopy,
					Used:     used,
					Reason:   err.Error(),
				},
			)
			return operationPreparationError(
				"workspace_context_stale",
				err.Error(),
				plan.summary,
				err,
			)
		}

		plan.summary.Instructions = append(
			plan.summary.Instructions,
			sessionRuntime.ResolutionItem{
				Key:      key,
				Kind:     kindWorkspaceContext,
				Owner:    sessionRuntime.ConfigurationOwnerWorkspace,
				State:    sessionRuntime.ResolutionStateIncluded,
				Selected: &selected,
				Used:     used,
			},
		)
		includedWorkspaceContexts[key] = struct{}{}
	}

	for _, selected := range contextSelections {
		key := selected.CanonicalKey()
		if _, included := includedWorkspaceContexts[key]; included {
			continue
		}

		selectedCopy := selected
		plan.summary.Instructions = append(
			plan.summary.Instructions,
			sessionRuntime.ResolutionItem{
				Key:      key,
				Kind:     kindWorkspaceContext,
				Owner:    sessionRuntime.ConfigurationOwnerWorkspace,
				State:    sessionRuntime.ResolutionStateExcluded,
				Selected: &selectedCopy,
				Reason:   "Workspace Context was not included in the resolved runtime plan",
			},
		)
		plan.summary.Diagnostics = append(
			plan.summary.Diagnostics,
			sessionRuntime.ResolutionDiagnostic{
				Severity: sessionRuntime.DiagnosticSeverityWarning,
				Code:     "workspace_context_excluded",
				Message:  "Selected Workspace Context was not included in the resolved runtime plan",
				Key:      key,
			},
		)
	}

	skillByRef := make(
		map[string]sessionRuntime.ArtifactSelection,
		len(skillSelections),
	)
	for _, value := range skillSelections {
		skillByRef[value.CanonicalKey()] = value
	}

	includedWorkspaceSkills := make(map[string]struct{}, len(runtimePlan.Skills.Skills))
	for _, skill := range runtimePlan.Skills.Skills {
		key := artifactKey(skill.Artifact)
		selected, found := skillByRef[key]
		if !found {
			continue
		}

		used := resolvedArtifact(
			skill.Artifact,
			skill.ArtifactRevision,
			skill.DefinitionDigest,
		)
		if err := requireSelectionPin(selected, used); err != nil {
			selectedCopy := selected
			plan.summary.Skills = append(
				plan.summary.Skills,
				sessionRuntime.ResolutionItem{
					Key:      key,
					Kind:     resolutionItemKindSkill,
					Owner:    sessionRuntime.ConfigurationOwnerWorkspace,
					State:    sessionRuntime.ResolutionStateStale,
					Selected: &selectedCopy,
					Used:     used,
					Reason:   err.Error(),
				},
			)
			return operationPreparationError(
				"workspace_skill_stale",
				err.Error(),
				plan.summary,
				err,
			)
		}

		plan.skills = append(plan.skills, skillContribution{
			key:       selected.CanonicalKey(),
			owner:     sessionRuntime.ConfigurationOwnerWorkspace,
			selection: selected,
			item: sessionRuntime.ResolutionItem{
				Key:      selected.CanonicalKey(),
				Kind:     resolutionItemKindSkill,
				Owner:    sessionRuntime.ConfigurationOwnerWorkspace,
				State:    sessionRuntime.ResolutionStateIncluded,
				Selected: &selected,
				Used:     used,
			},
		})
		includedWorkspaceSkills[selected.CanonicalKey()] = struct{}{}
	}

	for _, selected := range skillSelections {
		key := selected.CanonicalKey()
		if _, included := includedWorkspaceSkills[key]; included {
			continue
		}

		selectedCopy := selected
		plan.summary.Skills = append(
			plan.summary.Skills,
			sessionRuntime.ResolutionItem{
				Key:      key,
				Kind:     resolutionItemKindSkill,
				Owner:    sessionRuntime.ConfigurationOwnerWorkspace,
				State:    sessionRuntime.ResolutionStateExcluded,
				Selected: &selectedCopy,
				Reason:   "Workspace Skill was not included in the resolved runtime plan",
			},
		)
		plan.summary.Diagnostics = append(
			plan.summary.Diagnostics,
			sessionRuntime.ResolutionDiagnostic{
				Severity: sessionRuntime.DiagnosticSeverityWarning,
				Code:     "workspace_skill_excluded",
				Message:  "Selected Workspace Skill was not included in the resolved runtime plan",
				Key:      key,
			},
		)
	}

	for _, target := range artifactTargetsForCapabilities(
		capabilities,
		declaration.TypeTool,
	) {
		targetCopy := target.Clone()
		plan.tools = append(plan.tools, toolContribution{
			key:    capabilityTargetKey(targetCopy),
			owner:  sessionRuntime.ConfigurationOwnerWorkspace,
			target: &targetCopy,
			item: sessionRuntime.ResolutionItem{
				Key:   capabilityTargetKey(targetCopy),
				Kind:  resolutionItemKindTool,
				Owner: sessionRuntime.ConfigurationOwnerWorkspace,
				State: sessionRuntime.ResolutionStateIncluded,
			},
		})
	}

	mcpByRef := make(
		map[string]sessionRuntime.MCPServerSelection,
		len(mcpSelections),
	)
	for _, value := range mcpSelections {
		mcpByRef[value.Server.CanonicalKey()] = cloneMCPSelection(value)
	}

	includedWorkspaceMCP := make(map[string]struct{}, len(runtimePlan.MCPServers.Servers))
	for _, server := range runtimePlan.MCPServers.Servers {
		key := artifactKey(server.Artifact)
		selected, found := mcpByRef[key]
		if !found {
			continue
		}

		used := resolvedArtifact(
			server.Artifact,
			server.ArtifactRevision,
			server.DefinitionDigest,
		)
		if err := requireSelectionPin(selected.Server, used); err != nil {
			selectedServer := selected.Server
			plan.summary.MCP = append(
				plan.summary.MCP,
				sessionRuntime.ResolutionItem{
					Key:      selected.CanonicalKey(),
					Kind:     resolutionItemKindMCP,
					Owner:    sessionRuntime.ConfigurationOwnerWorkspace,
					State:    sessionRuntime.ResolutionStateStale,
					Selected: &selectedServer,
					Used:     used,
					Reason:   err.Error(),
				},
			)
			return operationPreparationError(
				"workspace_mcp_stale",
				err.Error(),
				plan.summary,
				err,
			)
		}

		plan.mcp = append(plan.mcp, selected)
		plan.mcpUses[selected.CanonicalKey()] = used
		plan.mcpOwners[selected.CanonicalKey()] = sessionRuntime.ConfigurationOwnerWorkspace
		includedWorkspaceMCP[selected.CanonicalKey()] = struct{}{}
	}

	for _, selected := range mcpSelections {
		key := selected.CanonicalKey()
		if _, included := includedWorkspaceMCP[key]; included {
			continue
		}

		selectedServer := selected.Server
		plan.summary.MCP = append(
			plan.summary.MCP,
			sessionRuntime.ResolutionItem{
				Key:      key,
				Kind:     resolutionItemKindMCP,
				Owner:    sessionRuntime.ConfigurationOwnerWorkspace,
				State:    sessionRuntime.ResolutionStateExcluded,
				Selected: &selectedServer,
				Reason:   "Workspace MCP server was not included in the resolved runtime plan",
			},
		)
		plan.summary.Diagnostics = append(
			plan.summary.Diagnostics,
			sessionRuntime.ResolutionDiagnostic{
				Severity: sessionRuntime.DiagnosticSeverityWarning,
				Code:     "workspace_mcp_excluded",
				Message:  "Selected Workspace MCP server was not included in the resolved runtime plan",
				Key:      key,
			},
		)
	}

	for _, decision := range runtimePlan.Prompt.Decisions {
		if decision.Status == "" ||
			decision.Status == "included" {
			continue
		}

		plan.summary.Diagnostics = append(
			plan.summary.Diagnostics,
			sessionRuntime.ResolutionDiagnostic{
				Severity: sessionRuntime.DiagnosticSeverityWarning,
				Code: firstNonEmpty(
					decision.Code,
					"workspace_context_unavailable",
				),
				Message: "Workspace Context was not fully included",
				Key:     artifactKey(decision.Artifact),
			},
		)
	}

	return nil
}

func (c *coordinator) applySessionOverrides(
	ctx context.Context,
	configuration sessionRuntime.SessionConfiguration,
	plan *operationPlan,
) error {
	if configuration.Model != nil {
		plan.model = configuration.Model
		plan.modelOwner = sessionRuntime.ConfigurationOwnerSession
		plan.includeModelSystemPrompt = nil
	}

	instructions, err := c.applyInstructionOverrides(
		ctx,
		plan.instructions,
		configuration.Instructions,
	)
	if err != nil {
		return operationPreparationError(
			"instruction_override_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}
	plan.instructions = instructions

	tools, err := c.applyToolOverrides(
		ctx,
		plan.tools,
		configuration.Tools,
	)
	if err != nil {
		return operationPreparationError(
			"tool_override_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}
	plan.tools = tools

	skills, err := applySkillOverrides(
		plan.skills,
		configuration.Skills,
	)
	if err != nil {
		return operationPreparationError(
			"skill_override_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}
	plan.skills = skills

	active, activeSelections, err := applyActiveSkillOverrides(
		plan.activeSkills,
		plan.activeSkillSelections,
		configuration.ActiveSkills,
	)
	if err != nil {
		return operationPreparationError(
			"active_skill_override_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}
	if err := validateActiveSkills(plan.skills, active); err != nil {
		return operationPreparationError(
			"active_skill_override_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}
	plan.activeSkills = active
	plan.activeSkillSelections = activeSelections

	mcp, err := applyMCPOverrides(
		plan.mcp,
		configuration.MCP,
	)
	if err != nil {
		return operationPreparationError(
			"mcp_override_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}

	sessionSelections := sessionMCPOverrideSelections(configuration.MCP)
	nextMCPUses := make(
		map[string]*sessionRuntime.ResolvedArtifact,
		len(mcp),
	)
	nextMCPOwners := make(
		map[string]sessionRuntime.ConfigurationOwner,
		len(mcp),
	)
	for _, selection := range mcp {
		key := selection.CanonicalKey()
		used := plan.mcpUses[key]
		owner := plan.mcpOwners[key]

		if selected, overridden := sessionSelections[key]; overridden {
			selectedServer := selected.Server
			current, resolveErr := c.resolveArtifactSelection(
				ctx,
				selectedServer,
			)
			if resolveErr != nil {
				plan.summary.MCP = append(
					plan.summary.MCP,
					sessionRuntime.ResolutionItem{
						Key:      key,
						Kind:     resolutionItemKindMCP,
						Owner:    sessionRuntime.ConfigurationOwnerSession,
						State:    sessionRuntime.ResolutionStateUnavailable,
						Selected: &selectedServer,
						Reason:   resolveErr.Error(),
					},
				)
				return operationPreparationError(
					"mcp_selection_unavailable",
					resolveErr.Error(),
					plan.summary,
					resolveErr,
				)
			}
			if pinErr := requireSelectionPin(selectedServer, current); pinErr != nil {
				plan.summary.MCP = append(
					plan.summary.MCP,
					sessionRuntime.ResolutionItem{
						Key:      key,
						Kind:     resolutionItemKindMCP,
						Owner:    sessionRuntime.ConfigurationOwnerSession,
						State:    sessionRuntime.ResolutionStateStale,
						Selected: &selectedServer,
						Used:     current,
						Reason:   pinErr.Error(),
					},
				)
				return operationPreparationError(
					"mcp_selection_stale",
					pinErr.Error(),
					plan.summary,
					pinErr,
				)
			}
			used = current
			owner = sessionRuntime.ConfigurationOwnerSession
		}

		if owner == "" {
			owner = sessionRuntime.ConfigurationOwnerSession
		}
		nextMCPUses[key] = used
		nextMCPOwners[key] = owner
	}

	plan.mcp = mcp
	plan.mcpUses = nextMCPUses
	plan.mcpOwners = nextMCPOwners
	return nil
}

func (c *coordinator) appendArtifactInstruction(
	ctx context.Context,
	plan *operationPlan,
	selection sessionRuntime.ArtifactSelection,
	owner sessionRuntime.ConfigurationOwner,
	key string,
	required bool,
) error {
	ref, err := artifactRef(selection)
	if err != nil {
		return operationPreparationError(
			"instruction_selection_invalid",
			err.Error(),
			plan.summary,
			err,
		)
	}
	itemKey := firstNonEmpty(key, selection.CanonicalKey())

	content, used, label, err := c.materializeText(ctx, ref)
	if err != nil {
		severity := sessionRuntime.DiagnosticSeverityWarning
		if required {
			severity = sessionRuntime.DiagnosticSeverityError
		}
		selected := selection
		plan.summary.Instructions = append(
			plan.summary.Instructions,
			sessionRuntime.ResolutionItem{
				Key:      itemKey,
				Kind:     resolutionItemKindInstruction,
				Owner:    owner,
				State:    sessionRuntime.ResolutionStateUnavailable,
				Selected: &selected,
				Reason:   err.Error(),
			},
		)
		plan.summary.Diagnostics = append(
			plan.summary.Diagnostics,
			sessionRuntime.ResolutionDiagnostic{
				Severity: severity,
				Code:     "instruction_unavailable",
				Message:  err.Error(),
				Key:      selection.CanonicalKey(),
			},
		)
		if required {
			return operationPreparationError(
				"agent_required_instruction_unavailable",
				err.Error(),
				plan.summary,
				err,
			)
		}

		return nil
	}
	if err := requireSelectionPin(selection, used); err != nil {
		return operationPreparationError(
			"instruction_selection_stale",
			err.Error(),
			plan.summary,
			err,
		)
	}

	plan.instructions = append(plan.instructions, instructionContribution{
		key:   itemKey,
		owner: owner,
		label: label,
		text:  content,
		item: sessionRuntime.ResolutionItem{
			Key:      itemKey,
			Kind:     resolutionItemKindInstruction,
			Owner:    owner,
			State:    sessionRuntime.ResolutionStateIncluded,
			Selected: &selection,
			Used:     used,
		},
	})
	return nil
}

func (c *coordinator) materializeText(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (
	text string,
	used *sessionRuntime.ResolvedArtifact,
	label string,
	err error,
) {
	value, err := c.texts.Materialize(ctx, ref)
	if err != nil {
		return "", nil, "", err
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return "", nil, "", err
	}

	var wire struct {
		Artifact         artifactModel.ArtifactRef `json:"artifact"`
		ArtifactRevision uint64                    `json:"artifactRevision"`
		DefinitionDigest cryptoutil.Digest         `json:"definitionDigest"`
		Name             string                    `json:"name"`
		Content          string                    `json:"content"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return "", nil, "", fmt.Errorf(
			"adapt Text materialization: %w",
			err,
		)
	}
	if wire.Artifact != ref {
		return "", nil, "", errors.New(
			"text materialization returned another Artifact",
		)
	}

	return wire.Content, &sessionRuntime.ResolvedArtifact{
		RootID:     string(wire.Artifact.RootID),
		ArtifactID: string(wire.Artifact.ArtifactID),
		Revision:   sessionRuntime.Revision(wire.ArtifactRevision),
		Digest:     string(wire.DefinitionDigest),
	}, wire.Name, nil
}

func selectionFromTarget(
	target *composition.CapabilityTarget,
) (*sessionRuntime.ArtifactSelection, bool) {
	if target == nil ||
		target.Form != composition.TargetFormArtifact ||
		target.Artifact == nil {
		return nil, false
	}

	return &sessionRuntime.ArtifactSelection{
		RootID:     string(target.Artifact.RootID),
		ArtifactID: string(target.Artifact.ArtifactID),
	}, true
}

func artifactSelectionsForCapabilities(
	plan composition.CapabilityPlan,
	kind declaration.Type,
) []sessionRuntime.ArtifactSelection {
	refs := artifactRefsForCapabilities(plan, kind)
	output := make(
		[]sessionRuntime.ArtifactSelection,
		0,
		len(refs),
	)
	for _, ref := range refs {
		output = append(output, sessionRuntime.ArtifactSelection{
			RootID:     string(ref.RootID),
			ArtifactID: string(ref.ArtifactID),
		})
	}
	return output
}

func artifactRefsForCapabilities(
	plan composition.CapabilityPlan,
	kind declaration.Type,
) []artifactModel.ArtifactRef {
	seen := make(map[artifactModel.ArtifactRef]struct{})
	output := make([]artifactModel.ArtifactRef, 0)

	for _, occurrence := range plan.Occurrences {
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Type != kind ||
			occurrence.Target == nil ||
			occurrence.Target.Form != composition.TargetFormArtifact ||
			occurrence.Target.Artifact == nil {
			continue
		}

		ref := *occurrence.Target.Artifact
		if _, duplicate := seen[ref]; duplicate {
			continue
		}
		seen[ref] = struct{}{}
		output = append(output, ref)
	}
	return output
}

func artifactTargetsForCapabilities(
	plan composition.CapabilityPlan,
	kind declaration.Type,
) []composition.CapabilityTarget {
	output := make([]composition.CapabilityTarget, 0)
	seen := make(map[string]struct{})

	for _, occurrence := range plan.Occurrences {
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Type != kind ||
			occurrence.Target == nil {
			continue
		}

		target := occurrence.Target.Clone()
		key := capabilityTargetKey(target)
		if _, duplicate := seen[key]; duplicate {
			continue
		}

		seen[key] = struct{}{}
		output = append(output, target)
	}
	return output
}

func artifactRefs(
	selections []sessionRuntime.ArtifactSelection,
) ([]artifactModel.ArtifactRef, error) {
	output := make(
		[]artifactModel.ArtifactRef,
		0,
		len(selections),
	)
	for _, selection := range selections {
		ref, err := artifactRef(selection)
		if err != nil {
			return nil, err
		}
		output = append(output, ref)
	}
	return output, nil
}

func workspaceMCPSelections(
	available []sessionRuntime.ArtifactSelection,
	overrides []sessionRuntime.MCPServerOverride,
) ([]sessionRuntime.MCPServerSelection, error) {
	allowed := make(map[string]struct{}, len(available))
	for _, selection := range available {
		allowed[selection.CanonicalKey()] = struct{}{}
	}

	values, err := applyMCPOverrides(nil, overrides)
	if err != nil {
		return nil, err
	}

	for _, selection := range values {
		if _, found := allowed[selection.Server.CanonicalKey()]; !found {
			return nil, fmt.Errorf(
				"workspace MCP %q is not an available Workspace capability",
				selection.Server.ArtifactID,
			)
		}
	}
	return values, nil
}

func applyArtifactOverrideSelections(
	baseline []sessionRuntime.ArtifactSelection,
	overrides []sessionRuntime.ArtifactOverride,
) ([]sessionRuntime.ArtifactSelection, error) {
	output := append(
		[]sessionRuntime.ArtifactSelection(nil),
		baseline...,
	)
	replacementStarted := false

	for _, override := range overrides {
		switch override.Mode {
		case sessionRuntime.CollectionOverrideInherit:
			continue

		case sessionRuntime.CollectionOverrideRemove:
			output = slices.DeleteFunc(
				output,
				func(selection sessionRuntime.ArtifactSelection) bool {
					return selection.CanonicalKey() == override.Key
				},
			)

		case sessionRuntime.CollectionOverrideReplace:
			if !replacementStarted {
				output = output[:0]
				replacementStarted = true
			}
			fallthrough

		case sessionRuntime.CollectionOverrideAdd:
			if override.Value == nil {
				return nil, errors.New(
					"artifact override has no value",
				)
			}
			if !containsArtifactSelection(output, *override.Value) {
				output = append(output, *override.Value)
			}

		default:
			return nil, fmt.Errorf(
				"unsupported Artifact override mode %q",
				override.Mode,
			)
		}
	}

	return output, nil
}

func applySkillOverrides(
	baseline []skillContribution,
	overrides []sessionRuntime.ArtifactOverride,
) ([]skillContribution, error) {
	byKey := make(map[string]skillContribution, len(baseline))
	order := make([]string, 0, len(baseline))
	replacementStarted := false

	for _, value := range baseline {
		if _, found := byKey[value.key]; !found {
			order = append(order, value.key)
		}
		byKey[value.key] = value
	}

	for _, override := range overrides {
		switch override.Mode {
		case sessionRuntime.CollectionOverrideInherit:
			continue

		case sessionRuntime.CollectionOverrideRemove:
			delete(byKey, override.Key)
			order = slices.DeleteFunc(order, func(key string) bool {
				return key == override.Key
			})

		case sessionRuntime.CollectionOverrideReplace:
			if !replacementStarted {
				clear(byKey)
				order = order[:0]
				replacementStarted = true
			}
			fallthrough

		case sessionRuntime.CollectionOverrideAdd:
			if override.Value == nil {
				return nil, errors.New("skill override has no value")
			}

			value := *override.Value
			key := value.CanonicalKey()
			if _, found := byKey[key]; !found {
				order = append(order, key)
			}
			byKey[key] = skillContribution{
				key:       key,
				owner:     sessionRuntime.ConfigurationOwnerSession,
				selection: value,
				item: sessionRuntime.ResolutionItem{
					Key:      key,
					Kind:     resolutionItemKindSkill,
					Owner:    sessionRuntime.ConfigurationOwnerSession,
					State:    sessionRuntime.ResolutionStateIncluded,
					Selected: &value,
				},
			}

		default:
			return nil, fmt.Errorf(
				"unsupported Skill override mode %q",
				override.Mode,
			)
		}
	}

	output := make([]skillContribution, 0, len(order))
	for _, key := range order {
		if value, found := byKey[key]; found {
			output = append(output, value)
		}
	}
	return output, nil
}

func applyActiveSkillOverrides(
	baseline map[string]struct{},
	baselineSelections map[string]sessionRuntime.ArtifactSelection,
	overrides []sessionRuntime.ActiveSkillOverride,
) (
	active map[string]struct{},
	activeSelection map[string]sessionRuntime.ArtifactSelection,
	err error,
) {
	output := make(map[string]struct{}, len(baseline))
	maps.Copy(output, baseline)
	selections := make(
		map[string]sessionRuntime.ArtifactSelection,
		len(baselineSelections),
	)
	maps.Copy(selections, baselineSelections)

	for _, override := range overrides {
		switch override.Mode {
		case sessionRuntime.ActivityOverrideInherit:
			continue

		case sessionRuntime.ActivityOverrideDeactivate:
			delete(output, override.Key)
			delete(selections, override.Key)

		case sessionRuntime.ActivityOverrideActivate:
			if override.Value == nil {
				return nil, nil, errors.New(
					"skill activation override has no value",
				)
			}
			key := override.Value.CanonicalKey()
			output[key] = struct{}{}
			selections[key] = *override.Value

		default:
			return nil, nil, fmt.Errorf(
				"unsupported active Skill override mode %q",
				override.Mode,
			)
		}
	}

	return output, selections, nil
}

// validateActiveSkills keeps the Session invariant explicit: an active Skill
// must also belong to the effective enabled Skill collection. Without this
// check an activate override for a removed or never-enabled Skill would be
// silently ignored later by Skill-session preparation.
func validateActiveSkills(
	skills []skillContribution,
	active map[string]struct{},
) error {
	enabled := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		enabled[skill.key] = struct{}{}
	}

	for key := range active {
		if _, found := enabled[key]; !found {
			return fmt.Errorf(
				"active Skill %q is not part of the effective enabled Skills",
				key,
			)
		}
	}
	return nil
}

func applyMCPOverrides(
	baseline []sessionRuntime.MCPServerSelection,
	overrides []sessionRuntime.MCPServerOverride,
) ([]sessionRuntime.MCPServerSelection, error) {
	byKey := make(map[string]sessionRuntime.MCPServerSelection)
	order := make([]string, 0)
	replacementStarted := false

	for _, value := range baseline {
		key := value.CanonicalKey()
		if _, found := byKey[key]; !found {
			order = append(order, key)
		}
		byKey[key] = cloneMCPSelection(value)
	}

	for _, override := range overrides {
		switch override.Mode {
		case sessionRuntime.CollectionOverrideInherit:
			continue

		case sessionRuntime.CollectionOverrideRemove:
			delete(byKey, override.Key)
			order = slices.DeleteFunc(order, func(key string) bool {
				return key == override.Key
			})

		case sessionRuntime.CollectionOverrideReplace:
			if !replacementStarted {
				clear(byKey)
				order = order[:0]
				replacementStarted = true
			}
			fallthrough

		case sessionRuntime.CollectionOverrideAdd:
			if override.Value == nil {
				return nil, errors.New("mcp override has no value")
			}

			key := override.Value.CanonicalKey()
			if _, found := byKey[key]; !found {
				order = append(order, key)
			}
			byKey[key] = cloneMCPSelection(*override.Value)

		default:
			return nil, fmt.Errorf(
				"unsupported MCP override mode %q",
				override.Mode,
			)
		}
	}

	output := make(
		[]sessionRuntime.MCPServerSelection,
		0,
		len(order),
	)
	for _, key := range order {
		if value, found := byKey[key]; found {
			output = append(output, value)
		}
	}
	return output, nil
}

func sessionMCPOverrideSelections(
	overrides []sessionRuntime.MCPServerOverride,
) map[string]sessionRuntime.MCPServerSelection {
	output := make(map[string]sessionRuntime.MCPServerSelection)
	replacementStarted := false

	for _, override := range overrides {
		switch override.Mode {
		case sessionRuntime.CollectionOverrideInherit:
			continue

		case sessionRuntime.CollectionOverrideRemove:
			delete(output, override.Key)

		case sessionRuntime.CollectionOverrideReplace:
			if !replacementStarted {
				clear(output)
				replacementStarted = true
			}
			fallthrough

		case sessionRuntime.CollectionOverrideAdd:
			if override.Value != nil {
				output[override.Value.CanonicalKey()] = cloneMCPSelection(*override.Value)
			}
		}
	}

	return output
}

func (c *coordinator) applyInstructionOverrides(
	ctx context.Context,
	baseline []instructionContribution,
	overrides []sessionRuntime.InstructionOverride,
) ([]instructionContribution, error) {
	byKey := make(
		map[string]instructionContribution,
		len(baseline),
	)
	order := make([]string, 0, len(baseline))
	replacementStarted := false

	for _, value := range baseline {
		if _, found := byKey[value.key]; !found {
			order = append(order, value.key)
		}
		byKey[value.key] = value
	}

	for _, override := range overrides {
		switch override.Mode {
		case sessionRuntime.CollectionOverrideInherit:
			continue

		case sessionRuntime.CollectionOverrideRemove:
			delete(byKey, override.Key)
			order = slices.DeleteFunc(order, func(key string) bool {
				return key == override.Key
			})

		case sessionRuntime.CollectionOverrideReplace:
			if !replacementStarted {
				clear(byKey)
				order = order[:0]
				replacementStarted = true
			}
			fallthrough

		case sessionRuntime.CollectionOverrideAdd:
			if override.Value == nil {
				return nil, errors.New(
					"instruction override has no value",
				)
			}

			value := *override.Value
			key := value.CanonicalKey()
			if _, found := byKey[key]; !found {
				order = append(order, key)
			}

			if value.Artifact != nil {
				temp := operationPlan{
					summary: sessionRuntime.OperationResolutionSummary{
						Version: operationResolutionVersion,
					},
				}
				if err := c.appendArtifactInstruction(
					ctx,
					&temp,
					*value.Artifact,
					sessionRuntime.ConfigurationOwnerSession,
					key,
					false,
				); err != nil {
					return nil, err
				}
				if len(temp.instructions) == 0 {
					return nil, errors.New(
						"instruction Artifact has no usable text",
					)
				}

				byKey[key] = temp.instructions[0]
				continue
			}

			byKey[key] = instructionContribution{
				key:   key,
				owner: sessionRuntime.ConfigurationOwnerSession,
				label: value.Label,
				text:  value.Text,
				item: sessionRuntime.ResolutionItem{
					Key:   key,
					Kind:  resolutionItemKindInstruction,
					Owner: sessionRuntime.ConfigurationOwnerSession,
					State: sessionRuntime.ResolutionStateIncluded,
				},
			}

		default:
			return nil, fmt.Errorf(
				"unsupported instruction override mode %q",
				override.Mode,
			)
		}
	}

	output := make(
		[]instructionContribution,
		0,
		len(order),
	)
	for _, key := range order {
		if value, found := byKey[key]; found {
			output = append(output, value)
		}
	}
	return output, nil
}

func (c *coordinator) applyToolOverrides(
	ctx context.Context,
	baseline []toolContribution,
	overrides []sessionRuntime.ToolOverride,
) ([]toolContribution, error) {
	byKey := make(map[string]toolContribution, len(baseline))
	order := make([]string, 0, len(baseline))
	replacementStarted := false

	for _, value := range baseline {
		if _, found := byKey[value.key]; !found {
			order = append(order, value.key)
		}
		byKey[value.key] = value
	}

	for _, override := range overrides {
		switch override.Mode {
		case sessionRuntime.CollectionOverrideInherit:
			continue

		case sessionRuntime.CollectionOverrideRemove:
			delete(byKey, override.Key)
			order = slices.DeleteFunc(order, func(key string) bool {
				return key == override.Key
			})

		case sessionRuntime.CollectionOverrideReplace:
			if !replacementStarted {
				clear(byKey)
				order = order[:0]
				replacementStarted = true
			}
			fallthrough

		case sessionRuntime.CollectionOverrideAdd:
			if override.Value == nil {
				return nil, errors.New("tool override has no value")
			}

			target, err := c.toolTargetFromIntent(
				ctx,
				*override.Value,
			)
			if err != nil {
				return nil, err
			}

			key := capabilityTargetKey(target)
			if _, found := byKey[key]; !found {
				order = append(order, key)
			}

			value := *override.Value
			var selected *sessionRuntime.ArtifactSelection
			if value.Capability.Artifact != nil {
				selectedValue := *value.Capability.Artifact
				selected = &selectedValue
			}
			byKey[key] = toolContribution{
				key:         key,
				owner:       sessionRuntime.ConfigurationOwnerSession,
				target:      &target,
				selection:   &value,
				autoExecute: value.AutoExecute,
				item: sessionRuntime.ResolutionItem{
					Key:      key,
					Kind:     resolutionItemKindTool,
					Owner:    sessionRuntime.ConfigurationOwnerSession,
					State:    sessionRuntime.ResolutionStateIncluded,
					Selected: selected,
				},
			}

		default:
			return nil, fmt.Errorf(
				"unsupported Tool override mode %q",
				override.Mode,
			)
		}
	}

	output := make([]toolContribution, 0, len(order))
	for _, key := range order {
		if value, found := byKey[key]; found {
			output = append(output, value)
		}
	}
	return output, nil
}

func (c *coordinator) toolTargetFromIntent(
	ctx context.Context,
	intent sessionRuntime.ToolSelectionIntent,
) (composition.CapabilityTarget, error) {
	if intent.Capability.Artifact != nil {
		ref, err := artifactRef(*intent.Capability.Artifact)
		if err != nil {
			return composition.CapabilityTarget{}, err
		}
		return c.tools.MapToolTarget(ctx, ref)
	}

	if intent.Capability.Direct != nil {
		direct := intent.Capability.Direct
		return composition.CapabilityTarget{
			Form:             "direct",
			Type:             declaration.Type(direct.Type),
			Name:             spec.LogicalName(direct.Name),
			ProviderIdentity: direct.ProviderIdentity,
			ProviderLocalID:  direct.ProviderLocalID,
		}, nil
	}

	return composition.CapabilityTarget{}, errors.New(
		"tool capability selection is required",
	)
}

func agentOccurrenceDiagnostic(
	occurrence agentAPI.AgentCapabilityOccurrence,
) sessionRuntime.ResolutionDiagnostic {
	severity := sessionRuntime.DiagnosticSeverityWarning
	if occurrence.Required {
		severity = sessionRuntime.DiagnosticSeverityError
	}

	return sessionRuntime.ResolutionDiagnostic{
		Severity: severity,
		Code: firstNonEmpty(
			occurrence.Code,
			"agent_capability_unavailable",
		),
		Message: firstNonEmpty(
			occurrence.Message,
			"Agent capability is unavailable",
		),
		Key: occurrence.Path,
	}
}

func unavailableTargetDiagnostic(
	key string,
	message string,
) sessionRuntime.ResolutionDiagnostic {
	return sessionRuntime.ResolutionDiagnostic{
		Severity: sessionRuntime.DiagnosticSeverityWarning,
		Code:     "capability_target_unavailable",
		Message:  message,
		Key:      key,
	}
}

func capabilityTargetKey(
	target composition.CapabilityTarget,
) string {
	if target.Form == composition.TargetFormArtifact &&
		target.Artifact != nil {
		return artifactKey(*target.Artifact)
	}

	return "direct:" +
		target.ProviderIdentity + ":" +
		target.ProviderLocalID + ":" +
		string(target.Type) + ":" +
		string(target.Name)
}

func artifactKey(ref artifactModel.ArtifactRef) string {
	return "artifact:" +
		string(ref.RootID) + ":" +
		string(ref.ArtifactID)
}

func containsArtifactSelection(
	values []sessionRuntime.ArtifactSelection,
	want sessionRuntime.ArtifactSelection,
) bool {
	for _, value := range values {
		if value.CanonicalKey() == want.CanonicalKey() {
			return true
		}
	}
	return false
}

func digestFromArtifact(
	value artifactModel.Artifact,
) cryptoutil.Digest {
	if value.ResolvedDefinition == nil {
		return ""
	}
	return *value.ResolvedDefinition
}

func cloneMCPSelection(
	value sessionRuntime.MCPServerSelection,
) sessionRuntime.MCPServerSelection {
	output := value
	output.SelectedTools = append(
		[]sessionRuntime.MCPToolSelection(nil),
		value.SelectedTools...,
	)
	output.Resources = append(
		[]sessionRuntime.MCPResourceSelection(nil),
		value.Resources...,
	)
	output.ResourceTemplates = append(
		[]sessionRuntime.MCPResourceTemplateSelection(nil),
		value.ResourceTemplates...,
	)
	output.Prompts = append(
		[]sessionRuntime.MCPPromptSelection(nil),
		value.Prompts...,
	)
	for index := range output.ResourceTemplates {
		output.ResourceTemplates[index].Arguments = maps.Clone(
			value.ResourceTemplates[index].Arguments,
		)
	}
	for index := range output.Prompts {
		output.Prompts[index].Arguments = maps.Clone(
			value.Prompts[index].Arguments,
		)
	}
	return output
}
