package sessionv3runtime

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/inference-go/capabilityoverride"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	inferenceRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// bindRecoveredLLM reconstructs process-private provider, Skill, and Tool
// resources from one immutable persisted dispatch plan. It does not consult
// current Session defaults, Agent capabilities, Workspace capabilities, or
// mutable contribution assembly.
func (b *Binder) bindRecoveredLLM(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
	persisted sessionv3Runtime.SemanticBinding,
) (sessionv3Runtime.BindResult, error) {
	if persisted.Plan == nil {
		return sessionv3Runtime.BindResult{}, errors.New(
			"recovered llm work item has no dispatch plan",
		)
	}
	plan := jsonutil.CloneJSONInValue(*persisted.Plan)
	if err := plan.Validate(); err != nil {
		return sessionv3Runtime.BindResult{}, err
	}

	defaults, err := recoveredSessionDefaults(plan)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	if defaults.Model == nil {
		return sessionv3Runtime.BindResult{}, errors.New(
			"recovered llm dispatch plan has no model selection",
		)
	}

	if len(plan.RuntimeConfiguration) == 0 {
		return sessionv3Runtime.BindResult{}, errors.New(
			"recovered llm dispatch plan has no immutable runtime configuration",
		)
	}
	resolved, err := b.deps.Models.RehydrateRuntimeConfiguration(
		ctx,
		plan.RuntimeConfiguration,
	)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	if resolved.ModelParam.SystemPrompt != plan.FinalSystemPrompt {
		return sessionv3Runtime.BindResult{}, errors.New(
			"recovered llm runtime configuration differs from the captured final prompt",
		)
	}

	inputs, err := recoveredProviderInputs(plan)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}

	if len(defaults.Skills) != 0 {
		if b.skillTools == nil {
			return sessionv3Runtime.BindResult{}, errors.New(
				"session v3 skill runtime is unavailable",
			)
		}
		hydration, hydrationErr := b.skillTools.HydrateLLM(
			ctx,
			request.Execution.ID,
			request.WorkItem.FrameID,
			defaults.Skills,
			defaults.ActiveSkills,
		)
		if hydrationErr != nil {
			return sessionv3Runtime.BindResult{}, hydrationErr
		}
		_ = hydration
	}

	choices, bindings, err := recoveredToolChoices(plan)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}

	semantic := jsonutil.CloneJSONInValue(persisted)
	configuration := cloneRecoveredRuntimeConfiguration(resolved)
	return sessionv3Runtime.BindResult{
		SemanticBinding: semantic,
		Private: privateLLM{
			Configuration: configuration,
			Inputs:        inputs,
			Choices:       choices,
			Bindings:      bindings,
			Fingerprint:   configuration.Fingerprint,
		},
	}, nil
}

func recoveredProviderInputs(
	plan sessionv3Runtime.ModelDispatchPlan,
) ([]inferenceSpec.InputUnion, error) {
	if len(plan.ProviderInputs) == 0 {
		return nil, errors.New(
			"recovered llm dispatch plan has no exact captured provider inputs",
		)
	}
	output := make([]inferenceSpec.InputUnion, 0, len(plan.ProviderInputs))
	for index := range plan.ProviderInputs {
		value, err := jsonutil.DecodeCanonicalValue[inferenceSpec.InputUnion](
			plan.ProviderInputs[index],
		)
		if err != nil {
			return nil, fmt.Errorf(
				"decode recovered provider input %d: %w",
				index,
				err,
			)
		}
		output = append(output, value)
	}
	return output, nil
}

func recoveredSessionDefaults(
	plan sessionv3Runtime.ModelDispatchPlan,
) (sessionv3Runtime.SessionDefaults, error) {
	modelSelection := plan.Model.Artifact.Selection()
	defaults := sessionv3Runtime.SessionDefaults{
		History: plan.History,
		Model:   &modelSelection,
	}
	if plan.ModelOverrides != nil {
		defaults.ModelOverrides = jsonutil.CloneJSONInValue(
			*plan.ModelOverrides,
		)
	}
	if plan.Agent != nil {
		agentSelection := plan.Agent.Artifact.Selection()
		defaults.Agent = &agentSelection
	}
	if plan.Workspace != nil {
		workspaceSelection := plan.Workspace.Artifact.Selection()
		defaults.Workspace = &workspaceSelection
	}
	for _, skill := range plan.Skills {
		selection := skill.Artifact.Selection()
		defaults.Skills = append(defaults.Skills, selection)
		if skill.Active {
			defaults.ActiveSkills = append(
				defaults.ActiveSkills,
				selection,
			)
		}
	}
	for _, value := range plan.MCP {
		selection := sessionv3Runtime.MCPServerSelection{
			Server:                    value.Server.Selection(),
			IncludeServerInstructions: value.IncludeServerInstructions,
			ToolExposure:              value.ToolExposure,
		}
		for _, tool := range value.Tools {
			selection.SelectedTools = append(
				selection.SelectedTools,
				sessionv3Runtime.MCPToolSelection{
					ToolName:       tool.Name,
					ExpectedDigest: tool.ExpectedDigest,
					ApprovalRule:   cloneString(tool.ApprovalRule),
					ExecutionMode:  cloneString(tool.ExecutionMode),
				},
			)
		}
		defaults.MCP = append(defaults.MCP, selection)
	}
	if err := sessionv3Runtime.ValidateSessionDefaults(defaults); err != nil {
		return sessionv3Runtime.SessionDefaults{}, err
	}
	return defaults, nil
}

func recoveredToolChoices(
	plan sessionv3Runtime.ModelDispatchPlan,
) (
	[]inferenceSpec.ToolChoice,
	map[string]sessionv3Runtime.ToolCall,
	error,
) {
	prepared, err := preparedChoicesFromDispatchPlans(plan.Tools)
	if err != nil {
		return nil, nil, err
	}

	choices := make([]inferenceSpec.ToolChoice, 0, len(prepared))
	bindings := make(map[string]sessionv3Runtime.ToolCall, len(prepared)*2)
	for _, value := range prepared {
		if _, duplicate := bindings[value.Choice.ID]; duplicate {
			return nil, nil, errors.New(
				"recovered provider tool choice identity is duplicated",
			)
		}
		if _, duplicate := bindings[value.Choice.Name]; duplicate {
			return nil, nil, errors.New(
				"recovered provider tool name is duplicated",
			)
		}
		choices = append(choices, value.Choice)
		bindings[value.Choice.ID] = value.Call
		bindings[value.Choice.Name] = value.Call
	}
	return choices, bindings, nil
}

func cloneRecoveredRuntimeConfiguration(
	value inferenceRuntime.RuntimeConfiguration,
) inferenceRuntime.RuntimeConfiguration {
	output := value
	output.ProviderParam.DefaultHeaders = jsonutil.CloneJSONInValue(
		value.ProviderParam.DefaultHeaders,
	)
	output.CapabilityOverrides = append(
		[]*capabilityoverride.ModelCapabilitiesOverride(nil),
		value.CapabilityOverrides...,
	)
	return output
}
