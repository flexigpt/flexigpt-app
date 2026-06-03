package spec

import "github.com/flexigpt/flexigpt-app/internal/bundleitemutils"

type MCPToolSelection struct {
	BundleID         bundleitemutils.BundleID `json:"bundleID"`
	ServerID         MCPServerID              `json:"serverID"`
	ToolName         string                   `json:"toolName"`
	ProviderToolName string                   `json:"providerToolName,omitempty"`
	ChoiceID         string                   `json:"choiceID,omitempty"`
	Digest           string                   `json:"digest,omitempty"`

	ApprovalRule  *MCPApprovalRule  `json:"approvalRule,omitempty"`
	ExecutionMode *MCPExecutionMode `json:"executionMode,omitempty"`

	AppResourceURI string   `json:"appResourceUri,omitempty"`
	Visibility     []string `json:"visibility,omitempty"`
}

type MCPProviderToolMapping struct {
	BundleID bundleitemutils.BundleID `json:"bundleID"`
	ServerID MCPServerID              `json:"serverID"`

	ProviderToolName string `json:"providerToolName"`
	ChoiceID         string `json:"choiceID"`

	ToolName   string `json:"toolName"`
	ToolDigest string `json:"toolDigest"`

	ApprovalRule   MCPApprovalRule  `json:"approvalRule,omitempty"`
	ExecutionMode  MCPExecutionMode `json:"executionMode,omitempty"`
	AppResourceURI string           `json:"appResourceUri,omitempty"`
	Visibility     []string         `json:"visibility,omitempty"`
}

type MCPServerSelection struct {
	BundleID bundleitemutils.BundleID `json:"bundleID"`
	ServerID MCPServerID              `json:"serverID"`

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
	BundleID bundleitemutils.BundleID `json:"bundleID"`
	ServerID MCPServerID              `json:"serverID"`

	ServerDisplayName string `json:"serverDisplayName,omitempty"`

	ToolName         string `json:"toolName"`
	ProviderToolName string `json:"providerToolName"`
	ToolDigest       string `json:"toolDigest,omitempty"`

	ToolUseID  string `json:"toolUseID,omitempty"`
	ApprovalID string `json:"approvalID,omitempty"`

	AppResourceURI string `json:"appResourceUri,omitempty"`
	AppInstanceID  string `json:"appInstanceID,omitempty"`
}
