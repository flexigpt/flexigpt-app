package spec

import "github.com/flexigpt/flexigpt-app/internal/bundleitemutils"

type MCPToolExposure string

const (
	MCPToolExposureNone     MCPToolExposure = "none"
	MCPToolExposureAll      MCPToolExposure = "all"
	MCPToolExposureSelected MCPToolExposure = "selected"
)

type InvokeMCPToolRequestBody struct {
	Source           MCPInvocationSource `json:"source"                     required:"true"`
	ToolName         string              `json:"toolName"                   required:"true"`
	ProviderToolName string              `json:"providerToolName,omitempty"`
	ToolDigest       string              `json:"toolDigest,omitempty"`

	Arguments map[string]any `json:"arguments,omitempty"`

	ApprovalID    string `json:"approvalID,omitempty"`
	ApprovalToken string `json:"approvalToken,omitempty"`

	ConversationID string `json:"conversationID,omitempty"`
	MessageID      string `json:"messageID,omitempty"`
	ToolUseID      string `json:"toolUseID,omitempty"`

	AppInstanceID string `json:"appInstanceID,omitempty"`
}

type InvokeMCPToolRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

	Body *InvokeMCPToolRequestBody
}

type MCPToolAppRenderInfo struct {
	ResourceURI       string       `json:"resourceUri,omitempty"`
	MimeType          string       `json:"mimeType,omitempty"`
	Content           []MCPContent `json:"content,omitempty"`
	StructuredContent any          `json:"structuredContent,omitempty"`
	IsError           bool         `json:"isError,omitempty"`
}

type InvokeMCPToolResponseBody struct {
	BundleID bundleitemutils.BundleID `json:"bundleID"`
	ServerID MCPServerID              `json:"serverID"`

	ToolName         string `json:"toolName"`
	ProviderToolName string `json:"providerToolName,omitempty"`

	Content           []MCPContent `json:"content,omitempty"`
	StructuredContent any          `json:"structuredContent,omitempty"`
	IsError           bool         `json:"isError,omitempty"`

	Provenance MCPToolCallProvenance `json:"provenance"`
	App        *MCPToolAppRenderInfo `json:"app,omitempty"`
}

type InvokeMCPToolResponse struct {
	Body *InvokeMCPToolResponseBody
}
