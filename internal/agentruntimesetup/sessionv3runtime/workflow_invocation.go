package sessionv3runtime

import (
	"errors"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	preparedWorkspaceInstrPrefix = "workspace:"
	preparedWorkspaceInstrSuffix = ":instructions"
	preparedTextInstrPrefix      = "prepared-text:"
	preparedSkillInstrPrefix     = "prepared-skill:"
	preparedIDSeparator          = ":"
)

// applyPreparedContributionPlan consumes the neutral source-pinned
// contribution contract. It never traverses current Agent or Workspace
// capability graphs. Text, Workspace prompt, instruction Skill, and MCP
// semantic values are consumed directly from captured facts.
func (b *Binder) applyPreparedContributionPlan(
	plan *llmExecutionPlan,
	value sessionv3Runtime.PreparedContributionPlan,
) error {
	if plan == nil {
		return errors.New("prepared contribution plan is unavailable")
	}
	if err := value.Validate(); err != nil {
		return err
	}

	modelSelectedFromPreparedPlan := false
	modelOwner := sessionv3Runtime.ConfigurationOwner("")
	if value.Model != nil {
		selection := value.Model.Selection()
		if plan.model == nil {
			plan.model = &selection
			plan.defaults.Model = &selection
			modelSelectedFromPreparedPlan = true
			for _, order := range value.ContributionOrder {
				if order.Kind == sessionv3Runtime.PreparedContributionKindModel &&
					order.Key == sessionv3Runtime.PreparedContributionKey(
						sessionv3Runtime.PreparedContributionKindModel,
						*value.Model,
					) {
					modelOwner = order.Owner
					break
				}
			}
		}
		if err := plan.addArtifact(*value.Model); err != nil {
			return err
		}
	}
	if value.Agent != nil {
		selection := value.Agent.Selection()
		plan.defaults.Agent = &selection
		if err := plan.addArtifact(*value.Agent); err != nil {
			return err
		}
	}
	if value.Workspace != nil {
		selection := value.Workspace.Selection()
		plan.defaults.Workspace = &selection
		if err := plan.addArtifact(*value.Workspace); err != nil {
			return err
		}
	}
	if modelSelectedFromPreparedPlan &&
		modelOwner == sessionv3Runtime.ConfigurationOwnerAgent {
		plan.agentModelIncludeSystemPrompt = cloneBool(value.IncludeModelSystemPrompt)
	}

	if value.Workspace != nil && !value.WorkspacePromptPrepared {
		return errors.New(
			"prepared workspace has no immutable prompt and context facts",
		)
	}
	if len(value.ContributionOrder) == 0 &&
		(value.Model != nil || value.Agent != nil || value.Workspace != nil ||
			len(value.Texts) != 0 || len(value.Tools) != 0 ||
			len(value.Skills) != 0 || len(value.MCP) != 0) {
		return errors.New("prepared contribution plan has no contribution order")
	}

	texts := make(map[string]sessionv3Runtime.WorkflowPreparedText, len(value.Texts))
	for _, prepared := range value.Texts {
		texts[sessionv3Runtime.PreparedContributionKey(
			sessionv3Runtime.PreparedContributionKindText,
			prepared.Artifact,
		)] = prepared
	}
	tools := make(map[string]sessionv3Runtime.WorkflowPreparedTool, len(value.Tools))
	for _, prepared := range value.Tools {
		tools[sessionv3Runtime.PreparedContributionKey(
			sessionv3Runtime.PreparedContributionKindTool,
			prepared.Artifact,
		)] = prepared
	}
	skills := make(map[string]sessionv3Runtime.WorkflowPreparedSkill, len(value.Skills))
	for _, prepared := range value.Skills {
		skills[sessionv3Runtime.PreparedContributionKey(
			sessionv3Runtime.PreparedContributionKindSkill,
			prepared.Artifact,
		)] = prepared
	}
	mcp := make(map[string]sessionv3Runtime.WorkflowPreparedMCP, len(value.MCP))
	for _, prepared := range value.MCP {
		mcp[sessionv3Runtime.PreparedContributionKey(
			sessionv3Runtime.PreparedContributionKindMCP,
			prepared.Server,
		)] = prepared
	}

	appliedWorkspacePrompt := false
	for _, order := range value.ContributionOrder {
		switch order.Kind {
		case sessionv3Runtime.PreparedContributionKindModel,
			sessionv3Runtime.PreparedContributionKindAgent,
			sessionv3Runtime.PreparedContributionKindWorkspace:
			continue

		case sessionv3Runtime.PreparedContributionKindWorkspacePrompt:
			if appliedWorkspacePrompt || value.Workspace == nil ||
				value.WorkspacePrompt == nil ||
				order.Key != sessionv3Runtime.PreparedWorkspacePromptKey(
					*value.Workspace,
				) {
				return errors.New("prepared workspace prompt order is invalid")
			}
			appliedWorkspacePrompt = true
			if strings.TrimSpace(value.WorkspacePrompt.Instructions) != "" {
				plan.addInstruction(sessionv3Runtime.Instruction{
					ID: preparedWorkspaceInstrPrefix + value.Workspace.RootID + preparedIDSeparator +
						value.Workspace.ArtifactID + preparedWorkspaceInstrSuffix,
					Content:   value.WorkspacePrompt.Instructions,
					Owner:     sessionv3Runtime.ConfigurationOwnerWorkspace,
					SourceKey: order.Key,
				})
			}
			if strings.TrimSpace(value.WorkspacePrompt.UserContext) != "" {
				plan.addContext(value.WorkspacePrompt.UserContext)
			}

		case sessionv3Runtime.PreparedContributionKindText:
			prepared, found := texts[order.Key]
			if !found {
				return errors.New("prepared text contribution is unavailable")
			}
			if err := plan.addArtifact(prepared.Artifact); err != nil {
				return err
			}
			if !prepared.Included {
				continue
			}
			plan.addText(sessionv3Runtime.TextDispatchPlan{
				Artifact: prepared.Artifact,
				Insert:   prepared.Insert,
				Content:  prepared.Content,
			})
			workspacePromptOwnsInstructions :=
				value.WorkspacePromptPrepared &&
					prepared.Owner ==
						sessionv3Runtime.ConfigurationOwnerWorkspace
			if prepared.Insert ==
				sessionv3Runtime.TextDispatchInsertInstructions &&
				!workspacePromptOwnsInstructions {
				plan.addInstruction(sessionv3Runtime.Instruction{
					ID: preparedTextInstrPrefix + prepared.Artifact.RootID + preparedIDSeparator +
						prepared.Artifact.ArtifactID,
					Label:     prepared.Label,
					Content:   prepared.Content,
					Owner:     prepared.Owner,
					SourceKey: order.Key,
				})
			}

		case sessionv3Runtime.PreparedContributionKindTool:
			prepared, found := tools[order.Key]
			if !found {
				return errors.New("prepared tool contribution is unavailable")
			}
			if err := plan.addArtifact(prepared.Artifact); err != nil {
				return err
			}
			selection := prepared.Artifact.Selection()
			plan.defaults.ToolSelections = appendUniqueLLMToolSelection(
				plan.defaults.ToolSelections,
				sessionv3Runtime.ToolSelection{
					Artifact:      selection,
					UserArguments: prepared.UserArguments.Clone(),
					AutoExecute:   cloneBool(prepared.AutoExecute),
				},
			)

		case sessionv3Runtime.PreparedContributionKindSkill:
			prepared, found := skills[order.Key]
			if !found {
				return errors.New("prepared skill contribution is unavailable")
			}
			if err := plan.addArtifact(prepared.Artifact); err != nil {
				return err
			}
			selection := prepared.Artifact.Selection()
			if prepared.Instructions {
				if strings.TrimSpace(prepared.InstructionContent) == "" {
					return errors.New(
						"prepared instruction skill has no captured instruction content",
					)
				}
				plan.addInstruction(sessionv3Runtime.Instruction{
					ID: preparedSkillInstrPrefix + prepared.Artifact.RootID + preparedIDSeparator +
						prepared.Artifact.ArtifactID,
					Label:     prepared.Artifact.ArtifactID,
					Content:   prepared.InstructionContent,
					Owner:     prepared.Owner,
					SourceKey: order.Key,
				})
				if !prepared.Enabled {
					continue
				}
			}
			plan.defaults.Skills = appendUniqueLLMSelection(
				plan.defaults.Skills,
				selection,
			)
			if prepared.Active {
				plan.defaults.ActiveSkills = appendUniqueLLMSelection(
					plan.defaults.ActiveSkills,
					selection,
				)
			}

		case sessionv3Runtime.PreparedContributionKindMCP:
			prepared, found := mcp[order.Key]
			if !found {
				return errors.New("prepared mcp contribution is unavailable")
			}
			if !prepared.Prepared {
				return errors.New(
					"prepared mcp contribution has no immutable discovery and tool facts",
				)
			}
			if err := plan.addArtifact(prepared.Server); err != nil {
				return err
			}
			plan.defaults.MCP = appendUniqueMCPSelection(
				plan.defaults.MCP,
				sessionv3Runtime.MCPServerSelection{
					Server:                    prepared.Server.Selection(),
					IncludeServerInstructions: prepared.IncludeServerInstructions,
					ToolExposure:              prepared.ToolExposure,
					SelectedTools:             jsonutil.CloneJSONInValue(prepared.Tools),
					Resources:                 jsonutil.CloneJSONInValue(prepared.Resources),
					ResourceTemplates:         jsonutil.CloneJSONInValue(prepared.ResourceTemplates),
					Prompts:                   jsonutil.CloneJSONInValue(prepared.Prompts),
				},
			)
			plan.setPreparedMCP(prepared)

		default:
			return errors.New("prepared contribution kind is unavailable")
		}
	}
	if value.Workspace != nil && !appliedWorkspacePrompt {
		return errors.New("prepared workspace prompt was not applied in contribution order")
	}
	return nil
}
