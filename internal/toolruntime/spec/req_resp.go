package spec

import (
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	llmtoolsgoSpec "github.com/flexigpt/llmtools-go/spec"
)

type JSONRawString = toolSpec.JSONRawString

// InvokeHTTPOptions contains options specific to HTTP tool invocations.
// These are part of the HTTP request body.
type InvokeHTTPOptions struct {
	// Overrides the tool-level HTTP timeout (in milliseconds). Optional.
	TimeoutMS int `json:"timeoutMS,omitempty"`
	// ExtraHeaders will be merged into the outgoing request headers (taking precedence).
	ExtraHeaders map[string]string `json:"extraHeaders,omitempty"`
	// Secrets are key->value mappings used for template substitution in HTTP request
	// components (e.g. ${SECRET}). Optional.
	Secrets map[string]string `json:"secrets,omitempty"`
}

// InvokeGoOptions contains options specific to Go tool invocations.
type InvokeGoOptions struct {
	// Overrides the tool invocation timeout (in milliseconds). Optional.
	TimeoutMS int `json:"timeoutMS,omitempty"`
}

// InvokeToolRequestBody is the body for invoking a tool.
type InvokeToolRequestBody struct {
	// Arguments passed to the tool. Must be JSON-serializable.
	Args JSONRawString `json:"args" required:"true"`

	// Tool-type-specific options (only one of these is used depending on the tool type).
	HTTPOptions *InvokeHTTPOptions `json:"httpOptions,omitempty"`
	GoOptions   *InvokeGoOptions   `json:"goOptions,omitempty"`
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
	// the tool definition.
	Outputs []llmtoolsgoSpec.ToolOutputUnion `json:"outputs,omitempty"`

	// Meta contains implementation-specific metadata (e.g., HTTP status, duration, etc.).
	Meta map[string]any `json:"meta,omitempty"`

	// True if the tool was served from the built-in data overlay.
	IsBuiltIn bool `json:"isBuiltIn"`

	// True if the tool itself reported an error during execution.
	// When true, Output may be empty or contain a tool-specific error payload.
	IsError bool `json:"isError,omitzero"`

	// ErrorMessage contains the error message returned by the tool, if any.
	// This is set when IsError is true.
	ErrorMessage string `json:"errorMessage,omitzero"`
}

type InvokeToolResponse struct {
	Body *InvokeToolResponseBody
}
