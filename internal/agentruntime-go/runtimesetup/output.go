package runtimesetup

import (
	"encoding/json"
	"errors"
	"strings"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
)

func decodeCompletionOutputs(
	outputs []inferenceSpec.OutputUnion,
	bindings preparedBindings,
) (
	outputText string,
	outputThinking string,
	outputCalls []sessionRuntime.ToolCallProposal,
) {
	var text strings.Builder
	var thinking strings.Builder
	calls := make([]sessionRuntime.ToolCallProposal, 0)

	for _, output := range outputs {
		if output.OutputMessage != nil {
			for _, content := range output.OutputMessage.Contents {
				if content.Kind == inferenceSpec.ContentItemKindText &&
					content.TextItem != nil {
					text.WriteString(content.TextItem.Text)
				}
			}
		}
		if output.ReasoningMessage != nil {
			thinking.WriteString(
				strings.Join(output.ReasoningMessage.Thinking, ""),
			)
		}

		call := output.FunctionToolCall
		if call == nil {
			call = output.CustomToolCall
		}
		if call == nil {
			call = output.WebSearchToolCall
		}
		if call == nil {
			continue
		}

		arguments, argumentsErr := normalizedArguments(call.Arguments)
		if argumentsErr != nil {
			calls = append(calls, unresolvedCompletionToolCall(
				call,
				json.RawMessage(`{}`),
				"provider Tool call arguments are invalid",
			))
			continue
		}

		proposal, found := bindings.byChoice[call.ChoiceID]
		if !found && call.ChoiceID == "" {
			proposal, found = bindings.byName[call.Name]
		}
		if !found {
			calls = append(calls, unresolvedCompletionToolCall(
				call,
				arguments,
				"provider Tool call cannot be matched to the immutable prepared bindings",
			))
			continue
		}
		if call.Name != "" && call.Name != proposal.Name {
			calls = append(calls, unresolvedCompletionToolCall(
				call,
				arguments,
				"provider Tool call name does not match the immutable prepared binding",
			))
			continue
		}

		proposal = cloneToolProposal(proposal)
		proposal.ID = sessionRuntime.ToolCallID(firstNonEmpty(call.ID, call.CallID))
		proposal.ProviderCallID = firstNonEmpty(call.CallID, call.ID)
		proposal.Arguments = arguments
		calls = append(calls, proposal)
	}

	return text.String(), thinking.String(), calls
}

func unresolvedCompletionToolCall(
	call *inferenceSpec.ToolCall,
	arguments json.RawMessage,
	message string,
) sessionRuntime.ToolCallProposal {
	if len(arguments) == 0 {
		arguments = json.RawMessage(`{}`)
	}

	return sessionRuntime.ToolCallProposal{
		ID:             sessionRuntime.ToolCallID(firstNonEmpty(call.ID, call.CallID)),
		ProviderCallID: firstNonEmpty(call.CallID, call.ID),
		Name:           call.Name,
		ChoiceID:       call.ChoiceID,
		Type:           call.Type,
		Arguments:      append(json.RawMessage(nil), arguments...),
		Required:       true,
		Binding: sessionRuntime.ToolBinding{
			Kind: sessionRuntime.ToolBindingKindUnresolved,
		},
		Error: message,
	}
}

func normalizedArguments(value string) (json.RawMessage, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return json.RawMessage(`{}`), nil
	}
	raw := json.RawMessage(value)
	if !json.Valid(raw) {
		return nil, errors.New("provider Tool arguments are not valid JSON")
	}
	return append(json.RawMessage(nil), raw...), nil
}

func hasUsableProtocolOutput(outputs []inferenceSpec.OutputUnion) bool {
	for _, output := range outputs {
		if output.OutputMessage != nil ||
			output.ReasoningMessage != nil ||
			output.FunctionToolCall != nil ||
			output.CustomToolCall != nil ||
			output.WebSearchToolCall != nil {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
