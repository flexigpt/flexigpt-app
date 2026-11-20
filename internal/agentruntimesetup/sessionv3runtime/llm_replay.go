package sessionv3runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

	requestDefaults := sessionv3Runtime.EffectiveModelDefaultsOverride(
		defaults.ModelOverrides,
		request.WorkItem.Input.LLM.Temperature,
	)
	modelDefaults, explicitPrompt, clearModelPrompt := splitV3ModelSystemPromptOverride(
		requestDefaults,
	)
	if plan.Agent != nil &&
		plan.Agent.IncludeModelSystemPrompt != nil &&
		!*plan.Agent.IncludeModelSystemPrompt {
		clearModelPrompt = true
	}
	if clearModelPrompt {
		modelDefaults.Clear = appendV3ModelDefaultClear(
			modelDefaults.Clear,
			sessionv3Runtime.ModelDefaultFieldSystemPrompt,
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

	instructions := make([]string, 0, len(plan.Instructions))
	for _, instruction := range plan.Instructions {
		if strings.TrimSpace(instruction.Content) != "" {
			instructions = append(instructions, instruction.Content)
		}
	}
	if len(instructions) != 0 {
		resolved.ModelParam.SystemPrompt = appendPrompt(
			resolved.ModelParam.SystemPrompt,
			strings.Join(instructions, "\n\n"),
		)
	}

	contextValues := make([]string, 0, len(plan.Context))
	for _, value := range plan.Context {
		text, decodeErr := jsonutil.DecodeCanonicalValue[string](value.Value)
		if decodeErr != nil {
			return sessionv3Runtime.BindResult{}, fmt.Errorf(
				"decode recovered context %q: %w",
				value.Name,
				decodeErr,
			)
		}
		if strings.TrimSpace(text) != "" {
			contextValues = append(contextValues, text)
		}
	}

	activities, err := recoveredDispatchActivities(plan)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	inputs, err := b.buildInputs(
		request,
		defaults,
		contextValues,
		activities,
	)
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
		if strings.TrimSpace(hydration.Prompt) != "" {
			resolved.ModelParam.SystemPrompt = appendPrompt(
				resolved.ModelParam.SystemPrompt,
				hydration.Prompt,
			)
		}
	}
	if explicitPrompt != nil && strings.TrimSpace(*explicitPrompt) != "" {
		resolved.ModelParam.SystemPrompt = appendPrompt(
			resolved.ModelParam.SystemPrompt,
			*explicitPrompt,
		)
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

func recoveredDispatchActivities(
	plan sessionv3Runtime.ModelDispatchPlan,
) ([]sessionv3Runtime.Activity, error) {
	if len(plan.Messages) == 0 {
		return nil, errors.New(
			"recovered llm dispatch plan has no captured message sequence",
		)
	}

	output := make(
		[]sessionv3Runtime.Activity,
		0,
		len(plan.Messages),
	)
	for _, message := range plan.Messages {
		if message.ID == "" {
			return nil, errors.New(
				"recovered llm dispatch plan has a message without identity",
			)
		}
		output = append(output, sessionv3Runtime.Activity{
			ID:         sessionv3Runtime.ActivityID(message.ID),
			Kind:       sessionv3Runtime.ActivityKindMessage,
			Visibility: sessionv3Runtime.ActivityVisibilityBoth,
			Message: &sessionv3Runtime.MessageActivity{
				Role:        message.Role,
				Text:        message.Text,
				Attachments: jsonutil.CloneJSONInValue(message.Attachments),
				ToolResults: jsonutil.CloneJSONInValue(message.ToolResults),
				AppContext:  jsonutil.CloneJSONInValue(message.AppContext),
				Metadata:    message.Metadata.Clone(),
			},
		})
	}
	return output, nil
}

func recoveredSessionDefaults(
	plan sessionv3Runtime.ModelDispatchPlan,
) (sessionv3Runtime.SessionDefaults, error) {
	defaults := sessionv3Runtime.SessionDefaults{
		History: plan.History,
		Model: &sessionv3Runtime.ArtifactSelection{
			RootID:           plan.Model.Artifact.RootID,
			ArtifactID:       plan.Model.Artifact.ArtifactID,
			ExpectedRevision: plan.Model.Artifact.Revision,
			ExpectedDigest:   plan.Model.Artifact.Digest,
		},
	}
	if plan.ModelOverrides != nil {
		defaults.ModelOverrides = jsonutil.CloneJSONInValue(
			*plan.ModelOverrides,
		)
	}
	if plan.Agent != nil {
		defaults.Agent = &sessionv3Runtime.ArtifactSelection{
			RootID:           plan.Agent.Artifact.RootID,
			ArtifactID:       plan.Agent.Artifact.ArtifactID,
			ExpectedRevision: plan.Agent.Artifact.Revision,
			ExpectedDigest:   plan.Agent.Artifact.Digest,
		}
	}
	if plan.Workspace != nil {
		defaults.Workspace = &sessionv3Runtime.ArtifactSelection{
			RootID:           plan.Workspace.Artifact.RootID,
			ArtifactID:       plan.Workspace.Artifact.ArtifactID,
			ExpectedRevision: plan.Workspace.Artifact.Revision,
			ExpectedDigest:   plan.Workspace.Artifact.Digest,
		}
	}
	for _, skill := range plan.Skills {
		selection := sessionv3Runtime.ArtifactSelection{
			RootID:           skill.Artifact.RootID,
			ArtifactID:       skill.Artifact.ArtifactID,
			ExpectedRevision: skill.Artifact.Revision,
			ExpectedDigest:   skill.Artifact.Digest,
		}
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
			Server: sessionv3Runtime.ArtifactSelection{
				RootID:           value.Server.RootID,
				ArtifactID:       value.Server.ArtifactID,
				ExpectedRevision: value.Server.Revision,
				ExpectedDigest:   value.Server.Digest,
			},
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
	choices := make([]inferenceSpec.ToolChoice, 0, len(plan.Tools))
	bindings := make(map[string]sessionv3Runtime.ToolCall, len(plan.Tools)*2)

	for _, value := range plan.Tools {
		if len(value.Choice) == 0 {
			return nil, nil, errors.New(
				"recovered tool dispatch plan has no provider choice",
			)
		}
		choice, err := jsonutil.DecodeCanonicalValue[inferenceSpec.ToolChoice](
			value.Choice,
		)
		if err != nil {
			return nil, nil, err
		}
		if choice.ID == "" ||
			choice.Name == "" ||
			choice.ID != value.Call.ChoiceID ||
			choice.Name != value.Call.Name {
			return nil, nil, errors.New(
				"recovered provider choice differs from its durable tool call",
			)
		}
		if _, duplicate := bindings[choice.ID]; duplicate {
			return nil, nil, errors.New(
				"recovered provider tool choice identity is duplicated",
			)
		}
		if _, duplicate := bindings[choice.Name]; duplicate {
			return nil, nil, errors.New(
				"recovered provider tool name is duplicated",
			)
		}
		choices = append(choices, choice)
		bindings[choice.ID] = jsonutil.CloneJSONInValue(value.Call)
		bindings[choice.Name] = jsonutil.CloneJSONInValue(value.Call)
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
