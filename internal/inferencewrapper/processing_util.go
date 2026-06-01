package inferencewrapper

import (
	"maps"
	"strings"

	"github.com/flexigpt/inference-go/debugclient"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

var defaultDebugConfig = debugclient.DebugConfig{
	Disable:                 false,
	DisableRequestBody:      false,
	DisableResponseBody:     false,
	DisableContentStripping: false,
	LogToSlog:               false,
}

func DefaultDebugConfig() debugclient.DebugConfig {
	return defaultDebugConfig
}

func disabledDebugConfig() debugclient.DebugConfig {
	cfg := defaultDebugConfig
	cfg.Disable = true
	return cfg
}

func prependCurrentInput(
	all []inferenceSpec.InputUnion,
	current []inferenceSpec.InputUnion,
	extra inferenceSpec.InputUnion,
) (nextAll, nextCurrent []inferenceSpec.InputUnion) {
	nextCurrent = make([]inferenceSpec.InputUnion, 0, len(current)+1)
	nextCurrent = append(nextCurrent, extra)
	nextCurrent = append(nextCurrent, current...)

	historyLen := len(all) - len(current)
	if historyLen < 0 || historyLen > len(all) {
		nextAll := make([]inferenceSpec.InputUnion, 0, len(all)+1)
		nextAll = append(nextAll, extra)
		nextAll = append(nextAll, all...)
		return nextAll, nextCurrent
	}

	nextAll = make([]inferenceSpec.InputUnion, 0, len(all)+1)
	nextAll = append(nextAll, all[:historyLen]...)
	nextAll = append(nextAll, extra)
	nextAll = append(nextAll, all[historyLen:]...)

	return nextAll, nextCurrent
}

// outputToInput converts an OutputUnion from a previous completion into an
// InputUnion so it can be replayed as prior context in the next call.
func outputToInput(o inferenceSpec.OutputUnion) *inferenceSpec.InputUnion {
	switch o.Kind {
	case inferenceSpec.OutputKindOutputMessage:
		return &inferenceSpec.InputUnion{
			Kind:          inferenceSpec.InputKindOutputMessage,
			OutputMessage: o.OutputMessage,
		}
	case inferenceSpec.OutputKindReasoningMessage:
		return &inferenceSpec.InputUnion{
			Kind:             inferenceSpec.InputKindReasoningMessage,
			ReasoningMessage: o.ReasoningMessage,
		}
	case inferenceSpec.OutputKindFunctionToolCall:
		return &inferenceSpec.InputUnion{
			Kind:             inferenceSpec.InputKindFunctionToolCall,
			FunctionToolCall: o.FunctionToolCall,
		}
	case inferenceSpec.OutputKindCustomToolCall:
		return &inferenceSpec.InputUnion{
			Kind:           inferenceSpec.InputKindCustomToolCall,
			CustomToolCall: o.CustomToolCall,
		}
	case inferenceSpec.OutputKindWebSearchToolCall:
		return &inferenceSpec.InputUnion{
			Kind:              inferenceSpec.InputKindWebSearchToolCall,
			WebSearchToolCall: o.WebSearchToolCall,
		}
	case inferenceSpec.OutputKindWebSearchToolOutput:
		return &inferenceSpec.InputUnion{
			Kind:                inferenceSpec.InputKindWebSearchToolOutput,
			WebSearchToolOutput: o.WebSearchToolOutput,
		}
	default:
		// Unknown kinds are dropped.
		return nil
	}
}

func mergeCompletionDebugDetails(existing any, key string, value any) any {
	if value == nil {
		return existing
	}
	if strings.TrimSpace(key) == "" {
		key = "extra"
	}
	if existing == nil {
		return map[string]any{key: value}
	}
	if m, ok := existing.(map[string]any); ok {
		out := maps.Clone(m)
		out[key] = value
		return out
	}
	return map[string]any{
		"provider": existing,
		key:        value,
	}
}

func getEmptySchema() map[string]any {
	return map[string]any{"type": "object"}
}

func getNonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}
