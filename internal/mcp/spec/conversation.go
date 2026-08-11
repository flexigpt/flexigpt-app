package spec

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
)

type MCPToolSelection struct {
	Server           artifact.ArtifactRef `json:"server"`
	ToolName         string               `json:"toolName"`
	ProviderToolName string               `json:"providerToolName,omitempty"`
	ChoiceID         string               `json:"choiceID,omitempty"`
	Digest           string               `json:"digest,omitempty"`

	ApprovalRule  *MCPApprovalRule  `json:"approvalRule,omitempty"`
	ExecutionMode *MCPExecutionMode `json:"executionMode,omitempty"`

	AppResourceURI string   `json:"appResourceUri,omitempty"`
	Visibility     []string `json:"visibility,omitempty"`
}

type MCPProviderToolMapping struct {
	Server artifact.ArtifactRef `json:"server"`

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
	Server artifact.ArtifactRef `json:"server"`

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
	Server     artifact.ArtifactRef     `json:"server"`
	Collection collection.CollectionRef `json:"collection"`

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
