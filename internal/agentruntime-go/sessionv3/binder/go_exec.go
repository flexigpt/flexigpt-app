package binder

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

func (b *Binder) executeGoTool(
	ctx context.Context,
	payload *sessionv3.ToolWork,
) (sessionv3.WorkExecutionResult, error) {
	binding := payload.Call.Binding.Go
	if binding == nil {
		return sessionv3.WorkExecutionResult{
			Error: &sessionv3.SanitizedError{Code: "invalid_go_binding", Message: "go binding is missing"},
		}, nil
	}
	args := payload.Call.Arguments
	if strings.TrimSpace(string(args)) == "" {
		args = sessionv3.CanonicalValue(`{}`)
	}
	if !json.Valid([]byte(args)) {
		return sessionv3.WorkExecutionResult{
			Error: &sessionv3.SanitizedError{Code: "invalid_arguments", Message: "tool arguments are not JSON"},
		}, nil
	}
	outputs, err := b.deps.ToolRuntime.CallGoTool(ctx, binding.Function, json.RawMessage(args), 0)
	if err != nil {
		text := string(mustMarshal(outputs))
		//nolint:nilerr // Explicit.
		return sessionv3.WorkExecutionResult{
			Output: &sessionv3.WorkItemOutput{
				Tool: &sessionv3.ToolOutput{
					Kind:       sessionv3.ToolOutputKindError,
					Text:       text,
					Structured: sessionv3.CanonicalValue(mustMarshal(outputs)),
				},
				Receipt: &sessionv3.ExternalReceipt{
					System:      "go-tool",
					Reference:   binding.Function,
					ConfirmedAt: time.Now().UTC(),
				},
			},
			Confirmed: true,
		}, nil
	}
	text, structured := toolOutputsToV3(outputs)
	return sessionv3.WorkExecutionResult{
		Output: &sessionv3.WorkItemOutput{
			Tool: &sessionv3.ToolOutput{Kind: sessionv3.ToolOutputKindSuccess, Text: text, Structured: structured},
			Receipt: &sessionv3.ExternalReceipt{
				System:      "go-tool",
				Reference:   binding.Function,
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, nil
}

func toolOutputsToV3(outputs []llmtoolsSpec.ToolOutputUnion) (string, sessionv3.CanonicalValue) {
	parts := []string{}
	for _, o := range outputs {
		switch o.Kind {
		case llmtoolsSpec.ToolOutputKindText:
			if o.TextItem != nil {
				parts = append(parts, o.TextItem.Text)
			}
		case llmtoolsSpec.ToolOutputKindImage:
			parts = append(parts, "[tool image omitted]")
		case llmtoolsSpec.ToolOutputKindFile:
			parts = append(parts, "[tool file omitted]")
		default:
			parts = append(parts, "[tool output omitted]")
		}
	}
	return strings.Join(parts, "\n\n"), sessionv3.MustCanonicalValue(outputs)
}

func mustMarshal(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return raw
}
