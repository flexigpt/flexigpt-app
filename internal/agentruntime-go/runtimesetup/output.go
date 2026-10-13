package runtimesetup

import (
	"encoding/json"
	"strings"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
)

type completionOutputWire struct {
	Kind              string         `json:"kind"`
	OutputMessage     *messageWire   `json:"outputMessage"`
	ReasoningMessage  *reasoningWire `json:"reasoningMessage"`
	FunctionToolCall  *toolCallWire  `json:"functionToolCall"`
	CustomToolCall    *toolCallWire  `json:"customToolCall"`
	WebSearchToolCall *toolCallWire  `json:"webSearchToolCall"`
}

type messageWire struct {
	Contents []contentWire `json:"contents"`
}

type contentWire struct {
	Kind     string `json:"kind"`
	TextItem *struct {
		Text string `json:"text"`
	} `json:"textItem"`
}

type reasoningWire struct {
	Thinking []string `json:"thinking"`
}

type toolCallWire struct {
	ID        string          `json:"id"`
	CallID    string          `json:"callID"`
	Name      string          `json:"name"`
	ChoiceID  string          `json:"choiceID"`
	Type      string          `json:"type"`
	Arguments json.RawMessage `json:"arguments"`
}

func decodeCompletionOutputs(
	outputs []json.RawMessage,
	bindings preparedBindings,
) (
	outputText string,
	outputThinking string,
	outputCalls []sessionRuntime.ToolCallProposal,
) {
	var text strings.Builder
	var thinking strings.Builder
	calls := make([]sessionRuntime.ToolCallProposal, 0)

	for _, raw := range outputs {
		var output completionOutputWire
		if err := json.Unmarshal(raw, &output); err != nil {
			continue
		}

		if output.OutputMessage != nil {
			for _, content := range output.OutputMessage.Contents {
				if content.Kind == "text" && content.TextItem != nil {
					text.WriteString(content.TextItem.Text)
				}
			}
		}
		if output.ReasoningMessage != nil {
			for _, value := range output.ReasoningMessage.Thinking {
				thinking.WriteString(value)
			}
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

		proposal, found := bindings.byChoice[call.ChoiceID]
		if !found {
			proposal, found = bindings.byName[call.Name]
		}
		if !found {
			calls = append(calls, sessionRuntime.ToolCallProposal{
				ID:             sessionRuntime.ToolCallID(firstNonEmpty(call.ID, call.CallID)),
				ProviderCallID: firstNonEmpty(call.CallID, call.ID),
				Name:           call.Name,
				ChoiceID:       call.ChoiceID,
				Type:           call.Type,
				Arguments:      normalizedArguments(call.Arguments),
				Required:       true,
				Binding: sessionRuntime.ToolBinding{
					Kind: sessionRuntime.ToolBindingKindUnresolved,
				},
				Error: "provider Tool call cannot be matched to the immutable prepared bindings",
			})
			continue
		}

		proposal = cloneToolProposal(proposal)
		proposal.ID = sessionRuntime.ToolCallID(firstNonEmpty(call.ID, call.CallID))
		proposal.ProviderCallID = firstNonEmpty(call.CallID, call.ID)
		proposal.Name = firstNonEmpty(call.Name, proposal.Name)
		proposal.ChoiceID = firstNonEmpty(call.ChoiceID, proposal.ChoiceID)
		if call.Type != "" {
			proposal.Type = call.Type
		}
		proposal.Arguments = normalizedArguments(call.Arguments)
		calls = append(calls, proposal)
	}

	return text.String(), thinking.String(), calls
}

func normalizedArguments(raw json.RawMessage) json.RawMessage {
	if strings.TrimSpace(string(raw)) == "" {
		return json.RawMessage(`{}`)
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err == nil {
		raw = json.RawMessage(encoded)
	}
	if !json.Valid(raw) {
		return json.RawMessage(`{}`)
	}
	return append(json.RawMessage(nil), raw...)
}

func hasUsableProtocolOutput(outputs []json.RawMessage) bool {
	for _, raw := range outputs {
		var output completionOutputWire
		if err := json.Unmarshal(raw, &output); err != nil {
			continue
		}
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
