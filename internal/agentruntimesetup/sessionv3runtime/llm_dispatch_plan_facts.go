package sessionv3runtime

import (
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func enrichDurableLLMDispatchPlan(
	output *sessionv3Runtime.ModelDispatchPlan,
	request sessionv3Runtime.WorkDispatchRequest,
	plan llmExecutionPlan,
	model sessionv3Runtime.ArtifactPin,
) error {
	if output == nil {
		return errors.New("model dispatch plan is unavailable")
	}

	output.Texts = cloneTextDispatchPlans(plan.texts)
	for index := range output.Tools {
		output.Tools[index].ProviderManaged =
			output.Tools[index].Call.Binding.Kind ==
				sessionv3Runtime.ToolBindingKindProvider
	}

	if len(output.MCP) != len(plan.defaults.MCP) {
		return errors.New("model dispatch plan mcp selections are incomplete")
	}
	for index := range plan.defaults.MCP {
		output.MCP[index] = mcpDispatchPlan(
			output.MCP[index].Server,
			plan.defaults.MCP[index],
		)
	}

	payload := request.WorkItem.Input.LLM
	if payload != nil {
		for _, activity := range selectHistory(request.Session, payload) {
			appendDurableActivityMessageDispatchPlan(output, activity)
		}
		if current := findActivity(request.Session, payload.InputActivityID); current != nil {
			appendDurableActivityMessageDispatchPlan(output, *current)
		}
	}

	if output.Workspace != nil {
		output.Workspace.Instructions = cloneInstructionDispatchPlans(
			output.Instructions,
		)
		output.Workspace.Texts = cloneTextDispatchPlans(output.Texts)
		output.Workspace.Skills = cloneSkillDispatchPlans(output.Skills)
		output.Workspace.MCP = cloneMCPDispatchPlans(output.MCP)
		output.Workspace.Context = cloneContextDispatchPlans(output.Context)
	}

	if output.Agent != nil {
		modelTarget := sessionv3Runtime.ModelDispatchTarget{Artifact: model}
		output.Agent.Model = &modelTarget
		output.Agent.Workspace = cloneWorkspaceDispatchPlan(output.Workspace)
		output.Agent.Instructions = cloneInstructionDispatchPlans(
			output.Instructions,
		)
		output.Agent.Texts = cloneTextDispatchPlans(output.Texts)
		output.Agent.Tools = cloneToolDispatchPlans(output.Tools)
		output.Agent.Skills = cloneSkillDispatchPlans(output.Skills)
		output.Agent.MCP = cloneMCPDispatchPlans(output.MCP)
	}

	return nil
}

func mcpDispatchPlan(
	pin sessionv3Runtime.ArtifactPin,
	selection sessionv3Runtime.MCPServerSelection,
) sessionv3Runtime.MCPDispatchPlan {
	output := sessionv3Runtime.MCPDispatchPlan{
		Server:                    pin,
		IncludeServerInstructions: selection.IncludeServerInstructions,
		ToolExposure:              selection.ToolExposure,
		Tools: make(
			[]sessionv3Runtime.MCPToolDispatchPlan,
			0,
			len(selection.SelectedTools),
		),
		Resources: make(
			[]sessionv3Runtime.MCPResourceDispatchPlan,
			0,
			len(selection.Resources),
		),
		ResourceTemplates: make(
			[]sessionv3Runtime.MCPResourceTemplateDispatchPlan,
			0,
			len(selection.ResourceTemplates),
		),
		Prompts: make(
			[]sessionv3Runtime.MCPPromptDispatchPlan,
			0,
			len(selection.Prompts),
		),
	}
	for _, value := range selection.SelectedTools {
		output.Tools = append(output.Tools, sessionv3Runtime.MCPToolDispatchPlan{
			Name:           value.ToolName,
			ExpectedDigest: value.ExpectedDigest,
			ApprovalRule:   cloneString(value.ApprovalRule),
			ExecutionMode:  cloneString(value.ExecutionMode),
		})
	}
	for _, value := range selection.Resources {
		output.Resources = append(output.Resources, sessionv3Runtime.MCPResourceDispatchPlan(value))
	}
	for _, value := range selection.ResourceTemplates {
		output.ResourceTemplates = append(
			output.ResourceTemplates,
			sessionv3Runtime.MCPResourceTemplateDispatchPlan{
				URITemplate:    value.URITemplate,
				ExpectedDigest: value.ExpectedDigest,
				Arguments:      cloneStringMap(value.Arguments),
			},
		)
	}
	for _, value := range selection.Prompts {
		output.Prompts = append(output.Prompts, sessionv3Runtime.MCPPromptDispatchPlan{
			PromptName:     value.PromptName,
			ExpectedDigest: value.ExpectedDigest,
			Arguments:      cloneStringMap(value.Arguments),
		})
	}
	return output
}

func appendDurableActivityMessageDispatchPlan(
	output *sessionv3Runtime.ModelDispatchPlan,
	activity sessionv3Runtime.Activity,
) {
	if output == nil || activity.Message == nil {
		return
	}
	appendDurableMessageDispatchPlan(
		output,
		sessionv3Runtime.MessageDispatchPlan{
			ID:          string(activity.ID),
			Role:        activity.Message.Role,
			Text:        activity.Message.Text,
			Attachments: append([]sessionv3Runtime.Attachment(nil), activity.Message.Attachments...),
		},
	)
}

func appendDurableMessageDispatchPlan(
	output *sessionv3Runtime.ModelDispatchPlan,
	message sessionv3Runtime.MessageDispatchPlan,
) {
	if output == nil {
		return
	}
	output.Messages = append(output.Messages, message)
	for _, attachment := range message.Attachments {
		output.Attachments = append(
			output.Attachments,
			sessionv3Runtime.AttachmentDispatchPlan{
				MessageID:  message.ID,
				Attachment: attachment,
			},
		)
	}
}

func cloneInstructionDispatchPlans(
	values []sessionv3Runtime.InstructionDispatchPlan,
) []sessionv3Runtime.InstructionDispatchPlan {
	return jsonutil.CloneJSONInValue(values)
}

func cloneTextDispatchPlans(
	values []sessionv3Runtime.TextDispatchPlan,
) []sessionv3Runtime.TextDispatchPlan {
	return jsonutil.CloneJSONInValue(values)
}

func cloneToolDispatchPlans(
	values []sessionv3Runtime.ToolDispatchPlan,
) []sessionv3Runtime.ToolDispatchPlan {
	return jsonutil.CloneJSONInValue(values)
}

func cloneSkillDispatchPlans(
	values []sessionv3Runtime.SkillDispatchPlan,
) []sessionv3Runtime.SkillDispatchPlan {
	return jsonutil.CloneJSONInValue(values)
}

func cloneMCPDispatchPlans(
	values []sessionv3Runtime.MCPDispatchPlan,
) []sessionv3Runtime.MCPDispatchPlan {
	return jsonutil.CloneJSONInValue(values)
}

func cloneContextDispatchPlans(
	values []sessionv3Runtime.ContextDispatchPlan,
) []sessionv3Runtime.ContextDispatchPlan {
	return jsonutil.CloneJSONInValue(values)
}

func cloneWorkspaceDispatchPlan(
	value *sessionv3Runtime.WorkspaceDispatchPlan,
) *sessionv3Runtime.WorkspaceDispatchPlan {
	if value == nil {
		return nil
	}
	output := jsonutil.CloneJSONInValue(*value)
	return &output
}

func cloneString(
	value *string,
) *string {
	if value == nil {
		return nil
	}
	output := *value
	return &output
}

func cloneStringMap(
	value map[string]string,
) map[string]string {
	if len(value) == 0 {
		return nil
	}
	return jsonutil.CloneJSONInValue(value)
}

func clonePlanPortInputs(
	values []sessionv3Runtime.PortInput,
) []sessionv3Runtime.PortInput {
	if len(values) == 0 {
		return nil
	}
	return jsonutil.CloneJSONInValue(values)
}
