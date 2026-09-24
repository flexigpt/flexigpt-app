package runtime

import (
	"encoding/json"

	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

type InvokeRequest struct {
	Function  string          `json:"function"`
	Args      json.RawMessage `json:"args"`
	TimeoutMS int             `json:"timeoutMS,omitempty"`
}

type InvokeResponse struct {
	Outputs      []llmtoolsSpec.ToolOutputUnion `json:"outputs,omitempty"`
	Meta         map[string]any                 `json:"meta,omitempty"`
	IsError      bool                           `json:"isError"`
	ErrorMessage string                         `json:"errorMessage,omitempty"`
}
