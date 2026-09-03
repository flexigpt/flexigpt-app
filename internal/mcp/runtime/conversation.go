package runtime

import (
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
)

type MCPToolSelection struct {
	Server           mcpSpec.ServerID `json:"server"`
	ToolName         string           `json:"toolName"`
	ProviderToolName string           `json:"providerToolName,omitempty"`
	ChoiceID         string           `json:"choiceID,omitempty"`
	Digest           string           `json:"digest,omitempty"`

	ApprovalRule  *mcpSpec.MCPApprovalRule  `json:"approvalRule,omitempty"`
	ExecutionMode *mcpSpec.MCPExecutionMode `json:"executionMode,omitempty"`

	AppResourceURI string   `json:"appResourceUri,omitempty"`
	Visibility     []string `json:"visibility,omitempty"`
}

type MCPProviderToolMapping struct {
	Server mcpSpec.ServerID `json:"server"`

	ProviderToolName string `json:"providerToolName"`
	ChoiceID         string `json:"choiceID"`

	ToolName   string `json:"toolName"`
	ToolDigest string `json:"toolDigest"`

	ApprovalRule   mcpSpec.MCPApprovalRule  `json:"approvalRule,omitempty"`
	ExecutionMode  mcpSpec.MCPExecutionMode `json:"executionMode,omitempty"`
	AppResourceURI string                   `json:"appResourceUri,omitempty"`
	Visibility     []string                 `json:"visibility,omitempty"`
}

type MCPToolExposure string

const (
	MCPToolExposureNone     MCPToolExposure = "none"
	MCPToolExposureAll      MCPToolExposure = "all"
	MCPToolExposureSelected MCPToolExposure = "selected"
)

type MCPServerSelection struct {
	Server mcpSpec.ServerID `json:"server"`

	SnapshotDigest string `json:"snapshotDigest,omitempty"`

	ToolExposure  MCPToolExposure    `json:"toolExposure"` // none | all | selected
	SelectedTools []MCPToolSelection `json:"selectedTools,omitempty"`

	IncludeServerInstructions bool `json:"includeServerInstructions,omitempty"`
}

type MCPResourceTemplateSelection struct {
	MCPResourceTemplateRef

	ArgumentValues map[string]string `json:"argumentValues,omitempty"`
}

type MCPPromptSelection struct {
	MCPPromptRef

	ArgumentValues map[string]string `json:"argumentValues,omitempty"`
}

type MCPConversationContext struct {
	Servers           []MCPServerSelection           `json:"servers"`
	Resources         []MCPResourceRef               `json:"resources,omitempty"`
	ResourceTemplates []MCPResourceTemplateSelection `json:"resourceTemplates,omitempty"`
	Prompts           []MCPPromptSelection           `json:"prompts,omitempty"`
}
type MCPToolCallProvenance struct {
	Server     mcpSpec.ServerID  `json:"server"`
	Collection mcpSpec.CatalogID `json:"collection"`

	ServerDisplayName string `json:"serverDisplayName,omitempty"`

	ToolName         string `json:"toolName"`
	ProviderToolName string `json:"providerToolName"`
	ToolDigest       string `json:"toolDigest,omitempty"`
	ChoiceID         string `json:"choiceID,omitempty"`

	ToolUseID  string `json:"toolUseID,omitempty"`
	ApprovalID string `json:"approvalID,omitempty"`

	AppResourceURI string `json:"appResourceUri,omitempty"`
	AppInstanceID  string `json:"appInstanceID,omitempty"`
}
