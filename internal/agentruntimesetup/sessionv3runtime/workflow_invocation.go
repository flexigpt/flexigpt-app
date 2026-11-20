package sessionv3runtime

import (
	"context"
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// applyPreparedWorkflowInvocation consumes source-backed contribution facts
// prepared before Runtime received the Workflow ProgramSnapshot. It never
// resolves the current Agent or Workspace contribution graph.
func (b *Binder) applyPreparedWorkflowInvocation(
	ctx context.Context,
	plan *llmExecutionPlan,
	value sessionv3Runtime.WorkflowPreparedInvocation,
) error {
	if plan == nil {
		return errors.New("workflow invocation plan is unavailable")
	}
	if err := value.Validate(); err != nil {
		return err
	}

	if value.Model != nil {
		selection := selectionForPreparedPin(*value.Model)
		plan.defaults.Model = &selection
		plan.model = &selection
		if err := plan.addArtifact(*value.Model); err != nil {
			return err
		}
	}
	if value.Agent != nil {
		if err := plan.addArtifact(*value.Agent); err != nil {
			return err
		}
	}
	if value.Workspace != nil {
		if err := plan.addArtifact(*value.Workspace); err != nil {
			return err
		}
	}
	plan.agentModelIncludeSystemPrompt = cloneBool(value.IncludeModelSystemPrompt)

	for _, prepared := range value.Texts {
		selection := selectionForPreparedPin(prepared.Artifact)
		if err := plan.addArtifact(prepared.Artifact); err != nil {
			return err
		}
		contribution, err := b.prepareLLMTextContribution(
			ctx,
			selection,
			prepared.Artifact,
		)
		if err != nil {
			return err
		}
		plan.addText(contribution.Dispatch)
		if contribution.Dispatch.Insert ==
			sessionv3Runtime.TextDispatchInsertInstructions {
			plan.addInstruction(sessionv3Runtime.Instruction{
				ID:      "workflow-text:" + prepared.Artifact.RootID + ":" + prepared.Artifact.ArtifactID,
				Label:   contribution.Label,
				Content: contribution.Dispatch.Content,
			})
		}
	}

	for _, prepared := range value.Tools {
		selection := selectionForPreparedPin(prepared.Artifact)
		if err := plan.addArtifact(prepared.Artifact); err != nil {
			return err
		}
		plan.defaults.ToolSelections = appendUniqueLLMToolSelection(
			plan.defaults.ToolSelections,
			sessionv3Runtime.ToolSelection{
				Artifact:    selection,
				AutoExecute: cloneBool(prepared.AutoExecute),
			},
		)
	}

	for _, prepared := range value.Skills {
		selection := selectionForPreparedPin(prepared.Artifact)
		if err := plan.addArtifact(prepared.Artifact); err != nil {
			return err
		}
		if prepared.Instructions {
			if b.skillTools == nil {
				return errors.New("session v3 skill runtime is unavailable")
			}
			instruction, err := b.skillTools.Instruction(ctx, selection)
			if err != nil {
				return err
			}
			plan.addInstruction(sessionv3Runtime.Instruction{
				ID:      "workflow-skill:" + prepared.Artifact.RootID + ":" + prepared.Artifact.ArtifactID,
				Label:   prepared.Artifact.ArtifactID,
				Content: instruction,
			})
			continue
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
	}

	for _, prepared := range value.MCP {
		selection := selectionForPreparedPin(prepared.Server)
		if err := plan.addArtifact(prepared.Server); err != nil {
			return err
		}
		plan.defaults.MCP = appendUniqueMCPSelection(
			plan.defaults.MCP,
			sessionv3Runtime.MCPServerSelection{
				Server:                    selection,
				IncludeServerInstructions: prepared.IncludeServerInstructions,
				ToolExposure:              prepared.ToolExposure,
			},
		)
	}
	return nil
}

func selectionForPreparedPin(
	value sessionv3Runtime.ArtifactPin,
) sessionv3Runtime.ArtifactSelection {
	return sessionv3Runtime.ArtifactSelection{
		RootID:           value.RootID,
		ArtifactID:       value.ArtifactID,
		ExpectedRevision: value.Revision,
		ExpectedDigest:   value.Digest,
	}
}
