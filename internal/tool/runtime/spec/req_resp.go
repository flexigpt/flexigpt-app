package spec

import (
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

// InvokeGoOptions contains options specific to Go tool invocations.
type InvokeGoOptions struct {
	// Overrides the tool invocation timeout (in milliseconds). Optional.
	TimeoutMS int `json:"timeoutMS,omitempty"`
}

// InvokeToolRequestBody is the body for invoking a tool.
//
// Args is JSON source. Wails transports it as a quoted string and the runtime
// unwraps it through jsonutil.DecodeJSONStringRaw before invoking the tool.
type InvokeToolRequestBody struct {
	// Arguments passed to the built-in Go tool.
	Args jsonutil.JSONRawString `json:"args" required:"true"`

	GoOptions *InvokeGoOptions `json:"goOptions,omitempty"`
}

type InvokeToolRequest struct {
	BundleID bundleitemutils.BundleID    `path:"bundleID" required:"true"`
	ToolSlug bundleitemutils.ItemSlug    `path:"toolSlug" required:"true"`
	Version  bundleitemutils.ItemVersion `path:"version"  required:"true"`
	Body     *InvokeToolRequestBody
}

// InvokeToolResponseBody is the result of a tool invocation.
type InvokeToolResponseBody struct {
	// Output is the JSON-serializable result produced by the tool. Its shape depends on
	// the tool providerapi.
	Outputs []llmtoolsSpec.ToolOutputUnion `json:"outputs,omitempty"`

	// Meta contains implementation-specific metadata (e.g., HTTP status, duration, etc.).
	Meta map[string]any `json:"meta,omitempty"`

	// True if the tool was served from the built-in data overlay.
	IsBuiltIn bool `json:"isBuiltIn"`

	// True if the tool itself reported an error during execution.
	// When true, Output may be empty or contain a tool-specific error payload.
	IsError bool `json:"isError"`

	// ErrorMessage contains the error message returned by the tool, if any.
	// This is set when IsError is true.
	ErrorMessage string `json:"errorMessage"`
}

type InvokeToolResponse struct {
	Body *InvokeToolResponseBody
}
