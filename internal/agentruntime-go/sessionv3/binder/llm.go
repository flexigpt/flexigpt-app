package binder

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	inferenceRuntimeSpec "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference/spec"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

type privateLLM struct {
	Configuration inferenceRuntimeSpec.RuntimeConfiguration
	Inputs        []inferenceSpec.InputUnion
	Choices       []inferenceSpec.ToolChoice
	Bindings      map[string]sessionv3.ToolCall
	Fingerprint   string
}

func artifactRefForSelection(s sessionv3.ArtifactSelection) artifactModel.ArtifactRef {
	return artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(s.RootID),
		ArtifactID: artifactModel.ArtifactID(s.ArtifactID),
	}
}

func (b *Binder) bindLLM(ctx context.Context, req sessionv3.WorkDispatchRequest) (sessionv3.BindResult, error) {
	payload := req.WorkItem.Input.LLM
	if payload == nil {
		return sessionv3.BindResult{}, errors.New("LLM payload is missing")
	}
	modelSel := payload.Model
	if modelSel == nil {
		modelSel = req.Session.Defaults.Model
	}
	if modelSel == nil {
		return sessionv3.BindResult{}, errors.New("no effective model selection")
	}

	ref := artifactRefForSelection(*modelSel)
	if err := ref.Validate(); err != nil {
		return sessionv3.BindResult{}, err
	}
	resolved, modelUse, err := b.deps.Models.ResolveRuntimeModelWithRequestPatch(
		ctx,
		ref,
		nil,
	)
	if err != nil {
		return sessionv3.BindResult{}, err
	}
	if modelUse == nil {
		return sessionv3.BindResult{}, errors.New("runtime model resolution returned no Artifact use")
	}
	if modelSel.ExpectedRevision != 0 &&
		modelSel.ExpectedRevision != modelUse.Revision {
		return sessionv3.BindResult{}, errors.New("selected Model revision changed")
	}
	if modelSel.ExpectedDigest != "" &&
		modelSel.ExpectedDigest != modelUse.Digest {
		return sessionv3.BindResult{}, errors.New("selected Model definition changed")
	}

	if prompt := b.systemPromptText(ctx, req); strings.TrimSpace(prompt) != "" {
		resolved.ModelParam.SystemPrompt = appendPrompt(resolved.ModelParam.SystemPrompt, prompt)
	}
	inputs, err := b.buildInputs(req)
	if err != nil {
		return sessionv3.BindResult{}, err
	}
	choices, bindings, err := b.toolChoices(ctx, req)
	if err != nil {
		return sessionv3.BindResult{}, err
	}

	configuration := resolved
	configuration.ProviderParam.DefaultHeaders = maps.Clone(
		resolved.ProviderParam.DefaultHeaders,
	)
	configuration.CapabilityOverrides = slices.Clone(
		resolved.CapabilityOverrides,
	)
	semantic := sessionv3.SemanticBinding{
		Version:           "v3-llm/v1",
		TargetFingerprint: resolved.Fingerprint,
		BoundAt:           time.Now().UTC(),
	}
	return sessionv3.BindResult{
		SemanticBinding: semantic,
		Private: privateLLM{
			Configuration: configuration,
			Inputs:        inputs,
			Choices:       choices,
			Bindings:      bindings,
		},
	}, nil
}

func (b *Binder) executeLLM(
	ctx context.Context,
	req sessionv3.WorkDispatchRequest,
	private privateLLM,
) (sessionv3.WorkExecutionResult, error) {
	configuration := private.Configuration
	configuration.ProviderParam.DefaultHeaders = maps.Clone(
		private.Configuration.ProviderParam.DefaultHeaders,
	)
	configuration.CapabilityOverrides = slices.Clone(
		private.Configuration.CapabilityOverrides,
	)
	resp, err := b.deps.Providers.FetchCompletion(
		ctx,
		&inferenceRuntimeSpec.CompletionRequest{
			Configuration: configuration,
			Inputs:        private.Inputs,
			ToolChoices:   private.Choices,
		},
	)
	if err != nil {
		//nolint:nilerr // Explicit return.
		return sessionv3.WorkExecutionResult{
			Error: &sessionv3.SanitizedError{Code: "completion_failed", Message: err.Error()},
		}, nil
	}
	if resp == nil || resp.Body == nil || resp.Body.InferenceResponse == nil {
		return sessionv3.WorkExecutionResult{
			Error: &sessionv3.SanitizedError{Code: "empty_completion", Message: "empty provider response"},
		}, nil
	}
	out := resp.Body.InferenceResponse
	if out.Error != nil {
		return sessionv3.WorkExecutionResult{
			Output:    &sessionv3.WorkItemOutput{LLM: &sessionv3.LLMOutput{Text: decodedProviderText(out.Outputs)}},
			Error:     &sessionv3.SanitizedError{Code: out.Error.Code, Message: out.Error.Message},
			Confirmed: true,
		}, nil
	}
	text, thinking, calls := b.decodeProviderOutputs(out.Outputs, private.Bindings)
	confirmed := strings.TrimSpace(text) != "" || len(calls) != 0
	return sessionv3.WorkExecutionResult{
		Output: &sessionv3.WorkItemOutput{
			LLM: &sessionv3.LLMOutput{Text: text, Thinking: thinking, ToolCalls: calls},
			Receipt: &sessionv3.ExternalReceipt{
				System:      "inference",
				Reference:   private.Fingerprint,
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: confirmed || out != nil,
	}, nil
}

func decodedProviderText(outputs []inferenceSpec.OutputUnion) string {
	var sb strings.Builder
	for _, o := range outputs {
		if o.OutputMessage != nil {
			for _, c := range o.OutputMessage.Contents {
				if c.TextItem != nil {
					sb.WriteString(c.TextItem.Text)
				}
			}
		}
	}
	return sb.String()
}

func (b *Binder) decodeProviderOutputs(
	outputs []inferenceSpec.OutputUnion,
	bindings map[string]sessionv3.ToolCall,
) (t, th string, c []sessionv3.ToolCall) {
	var text strings.Builder
	var thinking strings.Builder
	calls := []sessionv3.ToolCall{}
	for _, o := range outputs {
		if o.OutputMessage != nil {
			for _, c := range o.OutputMessage.Contents {
				if c.Kind == inferenceSpec.ContentItemKindText && c.TextItem != nil {
					text.WriteString(c.TextItem.Text)
				}
			}
		}
		if o.ReasoningMessage != nil {
			thinking.WriteString(strings.Join(o.ReasoningMessage.Thinking, ""))
		}
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
		raw := json.RawMessage(`{}`)
		if strings.TrimSpace(call.Arguments) != "" && json.Valid([]byte(call.Arguments)) {
			raw = json.RawMessage(append([]byte(nil), call.Arguments...))
		}
		if bound, ok := bindings[call.ChoiceID]; ok {
			next := bound
			next.ID = firstText(call.ID, call.CallID)
			next.ProviderCallID = firstText(call.CallID, call.ID)
			next.Arguments = sessionv3.CanonicalValue(raw)
			calls = append(calls, next)
			continue
		}
		if bound, ok := bindings[call.Name]; ok {
			next := bound
			next.ID = firstText(call.ID, call.CallID)
			next.ProviderCallID = firstText(call.CallID, call.ID)
			next.Arguments = sessionv3.CanonicalValue(raw)
			calls = append(calls, next)
			continue
		}
		calls = append(
			calls,
			sessionv3.ToolCall{
				ID:             firstText(call.ID, call.CallID),
				ProviderCallID: firstText(call.CallID, call.ID),
				Name:           call.Name,
				ChoiceID:       call.ChoiceID,
				Arguments:      sessionv3.CanonicalValue(raw),
				Binding: sessionv3.ToolBinding{
					Kind:       sessionv3.ToolBindingKindUnresolved,
					Unresolved: &sessionv3.UnresolvedToolBinding{Reason: "unresolved provider call"},
				},
			},
		)
	}
	return text.String(), thinking.String(), calls
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
	return base + "\n\n" + extra
}
