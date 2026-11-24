package sessionv3runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	inferenceRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/contributionruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/skillruntime"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

const (
	v3LLMSemanticBindingVersion = "v3-llm/v1"
	inferenceReceiptSystem      = "inference"
	emptyCompletionCode         = "empty_completion"
	completionFailedCode        = "completion_failed"
	providerOutputInvalidCode   = "provider_output_invalid"
	providerDebugInvalidCode    = "provider_debug_output_invalid"
	providerCallIDFormat        = "provider-call-%d"
	promptJoinSeparator         = "\n\n"
)

type privateLLM struct {
	Configuration inferenceRuntime.RuntimeConfiguration
	Inputs        []inferenceSpec.InputUnion
	Choices       []inferenceSpec.ToolChoice
	Bindings      map[string]sessionv3Runtime.ToolCall
	Fingerprint   string
}

func (b *Binder) bindLLM(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
) (sessionv3Runtime.BindResult, error) {
	payload := req.WorkItem.Input.LLM
	if payload == nil {
		return sessionv3Runtime.BindResult{}, errors.New("llm payload is missing")
	}
	if req.WorkItem.Binding != nil &&
		req.WorkItem.Binding.Plan != nil {
		return b.bindRecoveredLLM(
			ctx,
			req,
			*req.WorkItem.Binding,
		)
	}

	plan, err := b.prepareLLMPlan(ctx, req)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	modelSel := plan.model

	ref := contributionruntime.ArtifactRefForSelection(*modelSel)
	if err := ref.Validate(); err != nil {
		return sessionv3Runtime.BindResult{}, err
	}

	requestDefaults := sessionv3Runtime.EffectiveModelDefaultsOverride(
		plan.defaults.ModelOverrides,
		payload.Temperature,
	)
	modelDefaults, explicitSystemPrompt, clearModelSystemPrompt := splitV3ModelSystemPromptOverride(requestDefaults)
	if plan.agentModelIncludeSystemPrompt != nil &&
		!*plan.agentModelIncludeSystemPrompt {
		clearModelSystemPrompt = true
	}
	if clearModelSystemPrompt {
		modelDefaults.Clear = appendV3ModelDefaultClear(
			modelDefaults.Clear,
			sessionv3Runtime.ModelDefaultFieldSystemPrompt,
		)
	}

	requestPatch, patchErr := v3ModelRequestPatch(modelDefaults)
	if patchErr != nil {
		return sessionv3Runtime.BindResult{}, patchErr
	}
	resolved, modelUse, err := b.deps.Models.ResolveRuntimeModelWithRequestPatch(
		ctx,
		ref,
		requestPatch,
	)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	if modelUse == nil {
		return sessionv3Runtime.BindResult{}, errors.New("runtime model resolution returned no artifact use")
	}
	if modelUse.RootID == "" ||
		modelUse.ArtifactID == "" ||
		modelUse.Revision == 0 ||
		strings.TrimSpace(modelUse.Digest) == "" {
		return sessionv3Runtime.BindResult{}, errors.New(
			"runtime model resolution returned an incomplete artifact pin",
		)
	}
	if modelSel.ExpectedRevision != 0 &&
		modelSel.ExpectedRevision != modelUse.Revision {
		return sessionv3Runtime.BindResult{}, errors.New("selected model revision changed")
	}
	if modelSel.ExpectedDigest != "" &&
		modelSel.ExpectedDigest != modelUse.Digest {
		return sessionv3Runtime.BindResult{}, errors.New("selected model definition changed")
	}
	if modelUse.PreparedRuntimeConfiguration == nil {
		return sessionv3Runtime.BindResult{}, errors.New(
			"runtime model resolution returned no durable prepared configuration",
		)
	}

	if prompt := plan.systemPrompt(); strings.TrimSpace(prompt) != "" {
		resolved.ModelParam.SystemPrompt = appendPrompt(resolved.ModelParam.SystemPrompt, prompt)
	}

	inputs, err := b.buildInputs(
		req,
		plan.defaults,
		plan.context,
		plan.activities,
	)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}

	skillHydration := skillruntime.V3LLMSkillHydration{}
	if len(plan.defaults.Skills) != 0 {
		if b.skillTools == nil {
			return sessionv3Runtime.BindResult{}, errors.New(
				"session v3 skill runtime is unavailable",
			)
		}
		key := skillruntime.V3SkillSessionKey{
			ExecutionID: req.Execution.ID,
			FrameID:     req.WorkItem.FrameID,
		}
		skillHydration, err = b.skillTools.HydrateLLM(
			ctx,
			key.ExecutionID,
			key.FrameID,
			plan.defaults.Skills,
			plan.defaults.ActiveSkills,
		)
		if err != nil {
			return sessionv3Runtime.BindResult{}, err
		}
		if strings.TrimSpace(skillHydration.Prompt) != "" {
			resolved.ModelParam.SystemPrompt = appendPrompt(
				resolved.ModelParam.SystemPrompt,
				skillHydration.Prompt,
			)
		}
	}
	if explicitSystemPrompt != nil && strings.TrimSpace(*explicitSystemPrompt) != "" {
		resolved.ModelParam.SystemPrompt = appendPrompt(
			resolved.ModelParam.SystemPrompt,
			*explicitSystemPrompt,
		)
	}

	configuration, finalRuntimeConfiguration, err := b.deps.Models.
		FinalizePreparedRuntimeConfiguration(
			modelUse.PreparedRuntimeConfiguration,
			resolved,
		)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}

	choices, bindings, preparedChoices, err := b.toolChoices(
		ctx,
		req,
		plan,
		skillHydration.Choices,
	)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}

	artifacts := append(
		[]sessionv3Runtime.ArtifactPin(nil),
		plan.artifacts...,
	)
	modelPin := sessionv3Runtime.ArtifactPin{
		RootID:     modelUse.RootID,
		ArtifactID: modelUse.ArtifactID,
		Revision:   modelUse.Revision,
		Digest:     modelUse.Digest,
	}
	seen := make(map[string]sessionv3Runtime.ArtifactPin, len(artifacts)+1)
	for _, artifact := range artifacts {
		key := selectionKey(artifact.Selection())
		seen[key] = artifact
	}
	if existing, found := seen[selectionKey(modelPin.Selection())]; found {
		if existing != modelPin {
			return sessionv3Runtime.BindResult{}, errors.New(
				"model execution plan contains a conflicting model pin",
			)
		}
	} else {
		artifacts = append(artifacts, modelPin)
	}

	configuration.ProviderParam.DefaultHeaders = maps.Clone(
		configuration.ProviderParam.DefaultHeaders,
	)
	configuration.CapabilityOverrides = slices.Clone(configuration.CapabilityOverrides)
	semantic := sessionv3Runtime.SemanticBinding{
		Version:           v3LLMSemanticBindingVersion,
		Artifacts:         artifacts,
		TargetFingerprint: resolved.Fingerprint,
		BoundAt:           time.Now().UTC(),
	}
	dispatchPlan, err := b.durableLLMDispatchPlan(
		req,
		plan,
		modelPin,
		preparedChoices,
		inputs,
		configuration.ModelParam.SystemPrompt,
		finalRuntimeConfiguration,
	)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	semantic.Plan = &dispatchPlan
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

func (b *Binder) executeLLM(
	ctx context.Context,
	req sessionv3Runtime.WorkDispatchRequest,
	private privateLLM,
) (sessionv3Runtime.WorkExecutionResult, error) {
	configuration := private.Configuration
	configuration.ProviderParam.DefaultHeaders = maps.Clone(
		private.Configuration.ProviderParam.DefaultHeaders,
	)
	configuration.CapabilityOverrides = slices.Clone(
		private.Configuration.CapabilityOverrides,
	)
	resp, err := b.deps.Providers.FetchCompletion(
		ctx,
		&inferenceRuntime.CompletionRequest{
			Configuration:    configuration,
			Inputs:           private.Inputs,
			ToolChoices:      private.Choices,
			OnStreamText:     req.Stream.OnText,
			OnStreamThinking: req.Stream.OnThinking,
		},
	)
	if resp == nil || resp.Body == nil ||
		resp.Body.InferenceResponse == nil {
		message := "empty provider response"
		if err != nil {
			message = strings.ToLower(err.Error())
		}
		var typed *sessionv3Runtime.SanitizedError
		if errors.As(err, &typed) && typed != nil {
			value := *typed
			value.Details = typed.Details.Clone()
			value.Causes = append(
				[]sessionv3Runtime.SanitizedError(nil),
				typed.Causes...,
			)
			return sessionv3Runtime.WorkExecutionResult{
				Error: &value,
			}, nil
		}
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    emptyCompletionCode,
				Message: message,
			},
		}, nil
	}
	out := resp.Body.InferenceResponse
	facts, factsErr := modelruntime.InterpretCompletion(out)
	if factsErr != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    providerOutputInvalidCode,
				Message: strings.ToLower(factsErr.Error()),
			},
		}, nil
	}
	calls := b.decodeProviderToolCalls(out.Outputs, private.Bindings)
	llmOutput := &sessionv3Runtime.LLMOutput{
		Text:            facts.Text,
		Thinking:        facts.Thinking,
		ToolCalls:       calls,
		Usage:           facts.Usage,
		ProtocolOutputs: facts.ProtocolOutputs,
		DebugDetails:    facts.DebugDetails,
	}
	value := jsonutil.MustCanonicalValue(llmOutput)
	output := &sessionv3Runtime.WorkItemOutput{
		Value: value,
		LLM:   llmOutput,
		Receipt: &sessionv3Runtime.ExternalReceipt{
			System:      inferenceReceiptSystem,
			Reference:   private.Fingerprint,
			ConfirmedAt: time.Now().UTC(),
		},
	}
	if out.Error != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Output: output,
			Error: &sessionv3Runtime.SanitizedError{
				Code:    out.Error.Code,
				Message: strings.ToLower(out.Error.Message),
			},
			Confirmed: true,
		}, nil
	}
	confirmed := strings.TrimSpace(facts.Text) != "" ||
		len(calls) != 0 ||
		len(facts.ProtocolOutputs) != 0
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Output: output,
			Error: sessionv3Runtime.ClassifyDomainError(
				err, completionFailedCode, sessionv3Runtime.ErrorCategoryExecution,
			),
			Confirmed: confirmed,
		}, nil
	}
	if !confirmed {
		return sessionv3Runtime.WorkExecutionResult{
			Output: output,
			Error: &sessionv3Runtime.SanitizedError{
				Code:    emptyCompletionCode,
				Message: "provider returned no text or executable tool call",
			},
		}, nil
	}
	return sessionv3Runtime.WorkExecutionResult{
		Output:    output,
		Confirmed: true,
	}, nil
}

func (b *Binder) decodeProviderToolCalls(
	outputs []inferenceSpec.OutputUnion,
	bindings map[string]sessionv3Runtime.ToolCall,
) []sessionv3Runtime.ToolCall {
	calls := make([]sessionv3Runtime.ToolCall, 0)
	for _, o := range outputs {
		call := o.FunctionToolCall
		if call == nil {
			call = o.CustomToolCall
		}
		if call == nil {
			call = o.WebSearchToolCall
		}
		if call == nil {
			continue
		}

		callID := firstText(call.ID, call.CallID)
		if callID == "" {
			callID = fmt.Sprintf(providerCallIDFormat, len(calls)+1)
		}

		arguments := jsonutil.MustCanonicalValue(map[string]any{})
		if raw := strings.TrimSpace(call.Arguments); raw != "" {
			value, err := jsonutil.CanonicalValueFrom([]byte(raw))
			if err != nil {
				calls = append(calls, unresolvedProviderToolCall(
					call,
					callID,
					arguments,
					"provider tool call arguments are invalid json",
				))
				continue
			}
			arguments = value
		}

		bound, found := bindings[call.ChoiceID]
		if !found && call.ChoiceID == "" {
			bound, found = bindings[call.Name]
		}
		if !found || (call.Name != "" && call.Name != bound.Name) {
			calls = append(calls, unresolvedProviderToolCall(
				call,
				callID,
				arguments,
				"provider tool call does not match a prepared binding",
			))
			continue
		}

		next := bound
		next.ID = callID
		next.Type = call.Type
		next.Source = sessionv3Runtime.ToolCallSourceModel
		next.ProviderCallID = firstText(call.CallID, call.ID)
		next.Arguments = arguments
		calls = append(calls, next)
	}
	return calls
}

func unresolvedProviderToolCall(
	call *inferenceSpec.ToolCall,
	callID string,
	arguments jsonutil.CanonicalValue,
	reason string,
) sessionv3Runtime.ToolCall {
	return sessionv3Runtime.ToolCall{
		ID:             callID,
		ProviderCallID: firstText(call.CallID, call.ID),
		Name:           call.Name,
		ChoiceID:       call.ChoiceID,
		Type:           call.Type,
		Arguments:      arguments,
		Binding: sessionv3Runtime.ToolBinding{
			Kind:       sessionv3Runtime.ToolBindingKindUnresolved,
			Unresolved: &sessionv3Runtime.UnresolvedToolBinding{Reason: reason},
		},
		Source: sessionv3Runtime.ToolCallSourceModel,
	}
}

func firstText(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func appendPrompt(base, extra string) string {
	if strings.TrimSpace(base) == "" {
		return extra
	}
	if strings.TrimSpace(extra) == "" {
		return base
	}
	return base + promptJoinSeparator + extra
}

func splitV3ModelSystemPromptOverride(
	input sessionv3Runtime.ModelDefaultsOverride,
) (
	modelOverrides sessionv3Runtime.ModelDefaultsOverride,
	explicitSystemPrompt *string,
	clearModelSystemPrompt bool,
) {
	modelOverrides = jsonutil.CloneJSONInValue(input)
	modelOverrides.SystemPrompt = nil
	modelOverrides.Clear = make(
		[]sessionv3Runtime.ModelDefaultField,
		0,
		len(input.Clear),
	)
	for _, field := range input.Clear {
		if field == sessionv3Runtime.ModelDefaultFieldSystemPrompt {
			clearModelSystemPrompt = true
			continue
		}
		modelOverrides.Clear = append(modelOverrides.Clear, field)
	}
	if len(modelOverrides.Clear) == 0 {
		modelOverrides.Clear = nil
	}
	if input.SystemPrompt != nil {
		value := *input.SystemPrompt
		explicitSystemPrompt = &value
	}
	return modelOverrides, explicitSystemPrompt, clearModelSystemPrompt
}

func appendV3ModelDefaultClear(
	values []sessionv3Runtime.ModelDefaultField,
	field sessionv3Runtime.ModelDefaultField,
) []sessionv3Runtime.ModelDefaultField {
	if slices.Contains(values, field) {
		return values
	}
	return append(values, field)
}

func v3ModelRequestPatch(
	overrides sessionv3Runtime.ModelDefaultsOverride,
) (*modelruntime.RuntimeRequestPatch, error) {
	raw, err := json.Marshal(overrides)
	if err != nil {
		return nil, err
	}
	return modelruntime.RuntimeRequestPatchFromJSON(raw)
}
