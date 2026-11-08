package sessionv3runtime

import (
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

func (b *Binder) durableLLMDispatchPlan(
	request sessionv3Runtime.WorkDispatchRequest,
	plan llmExecutionPlan,
	model sessionv3Runtime.ArtifactPin,
	choices []preparedChoice,
) (sessionv3Runtime.ModelDispatchPlan, error) {
	output := sessionv3Runtime.ModelDispatchPlan{
		Model: sessionv3Runtime.ModelDispatchTarget{
			Artifact: model,
		},
		Ports:      clonePlanPortValues(request.WorkItem.Ports),
		PortInputs: clonePlanPortInputs(request.WorkItem.PortInputs),
	}

	if plan.defaults.Agent != nil {
		agent, found := pinForSelection(plan.artifacts, *plan.defaults.Agent)
		if !found {
			return sessionv3Runtime.ModelDispatchPlan{}, errors.New(
				"agent plan has no durable artifact pin",
			)
		}
		output.Agent = &sessionv3Runtime.AgentDispatchPlan{
			Artifact: agent,
		}
	}
	if plan.defaults.Workspace != nil {
		workspace, found := pinForSelection(
			plan.artifacts,
			*plan.defaults.Workspace,
		)
		if !found {
			return sessionv3Runtime.ModelDispatchPlan{}, errors.New(
				"workspace plan has no durable artifact pin",
			)
		}
		output.Workspace = &sessionv3Runtime.WorkspaceDispatchPlan{
			Artifact: workspace,
		}
	}

	for _, instruction := range plan.systemInstructions {
		output.Instructions = append(
			output.Instructions,
			sessionv3Runtime.InstructionDispatchPlan{
				ID:      instruction.ID,
				Label:   instruction.Label,
				Content: instruction.Content,
			},
		)
	}
	for _, selection := range plan.defaults.Skills {
		pin, found := pinForSelection(plan.artifacts, selection)
		if !found {
			return sessionv3Runtime.ModelDispatchPlan{}, fmt.Errorf(
				"skill plan %q has no durable artifact pin",
				selection.ArtifactID,
			)
		}
		output.Skills = append(
			output.Skills,
			sessionv3Runtime.SkillDispatchPlan{
				Artifact: pin,
				Active: containsSelection(
					plan.defaults.ActiveSkills,
					selection,
				),
			},
		)
	}
	for _, selection := range plan.defaults.MCP {
		pin, found := pinForSelection(plan.artifacts, selection.Server)
		if !found {
			return sessionv3Runtime.ModelDispatchPlan{}, fmt.Errorf(
				"mcp plan %q has no durable artifact pin",
				selection.Server.ArtifactID,
			)
		}
		output.MCP = append(
			output.MCP,
			sessionv3Runtime.MCPDispatchPlan{
				Server:                    pin,
				IncludeServerInstructions: selection.IncludeServerInstructions,
				ToolExposure:              selection.ToolExposure,
			},
		)
	}
	for _, choice := range choices {
		output.Tools = append(
			output.Tools,
			sessionv3Runtime.ToolDispatchPlan{
				Call: choice.Call,
				ProviderManaged: choice.Call.Binding.Kind ==
					sessionv3Runtime.ToolBindingKindProvider,
			},
		)
	}
	for index, value := range plan.context {
		output.Context = append(
			output.Context,
			sessionv3Runtime.ContextDispatchPlan{
				Name:  fmt.Sprintf("context-%d", index),
				Value: sessionv3Runtime.MustCanonicalValue(value),
			},
		)
	}

	for _, port := range request.WorkItem.Ports {
		appendLLMDispatchChannel(
			&output,
			port.Channel,
			port.Access,
			0,
		)
	}
	for _, input := range request.WorkItem.PortInputs {
		appendLLMDispatchChannel(
			&output,
			input.Channel,
			sessionv3Runtime.WorkflowPortAccessRead,
			input.Cursor,
		)
	}

	if err := enrichDurableLLMDispatchPlan(
		&output,
		request,
		plan,
		model,
	); err != nil {
		return sessionv3Runtime.ModelDispatchPlan{}, err
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.ModelDispatchPlan{}, err
	}
	return output, nil
}

func appendLLMDispatchChannel(
	output *sessionv3Runtime.ModelDispatchPlan,
	name string,
	access sessionv3Runtime.WorkflowPortAccess,
	cursor uint64,
) {
	if output == nil {
		return
	}
	for index := range output.Channels {
		channel := &output.Channels[index]
		if channel.Name != name {
			continue
		}
		if cursor > channel.Cursor {
			channel.Cursor = cursor
		}
		if channel.Access == access {
			return
		}
		channel.Access = sessionv3Runtime.WorkflowPortAccessReadWrite
		return
	}
	output.Channels = append(
		output.Channels,
		sessionv3Runtime.ChannelDispatchPlan{
			Name:   name,
			Cursor: cursor,
			Access: access,
		},
	)
}

func pinForSelection(
	pins []sessionv3Runtime.ArtifactPin,
	selection sessionv3Runtime.ArtifactSelection,
) (sessionv3Runtime.ArtifactPin, bool) {
	for _, pin := range pins {
		if pin.RootID == selection.RootID &&
			pin.ArtifactID == selection.ArtifactID {
			return pin, true
		}
	}
	return sessionv3Runtime.ArtifactPin{}, false
}

func containsSelection(
	values []sessionv3Runtime.ArtifactSelection,
	want sessionv3Runtime.ArtifactSelection,
) bool {
	for _, value := range values {
		if value.RootID == want.RootID &&
			value.ArtifactID == want.ArtifactID {
			return true
		}
	}
	return false
}

func clonePlanPortValues(
	values []sessionv3Runtime.PortPlan,
) []sessionv3Runtime.PortPlan {
	if len(values) == 0 {
		return nil
	}
	output := make([]sessionv3Runtime.PortPlan, len(values))
	for index := range values {
		output[index] = values[index]
		output[index].Schema = values[index].Schema.Clone()
		output[index].ChannelSchema = values[index].ChannelSchema.Clone()
	}
	return output
}
