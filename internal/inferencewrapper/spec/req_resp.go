package spec

import (
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	mcpConversation "github.com/flexigpt/flexigpt-app/internal/mcp/conversation"
	workspaceConversation "github.com/flexigpt/flexigpt-app/internal/workspace/conversation"
)

type AddProviderRequestBody struct {
	SDKType                  inferenceSpec.ProviderSDKType `json:"sdkType"`
	Origin                   string                        `json:"origin"`
	ChatCompletionPathPrefix string                        `json:"chatCompletionPathPrefix"`
	APIKeyHeaderKey          string                        `json:"apiKeyHeaderKey"`
	DefaultHeaders           map[string]string             `json:"defaultHeaders"`
}

type AddProviderRequest struct {
	Provider inferenceSpec.ProviderName `path:"provider" required:"true"`
	Body     *AddProviderRequestBody
}

type AddProviderResponse struct{}

type DeleteProviderRequest struct {
	Provider inferenceSpec.ProviderName `path:"provider" required:"true"`
}

type DeleteProviderResponse struct{}

type SetProviderAPIKeyRequestBody struct {
	APIKey string `json:"apiKey" required:"true"`
}

type SetProviderAPIKeyRequest struct {
	Provider inferenceSpec.ProviderName `path:"provider" required:"true"`
	Body     *SetProviderAPIKeyRequestBody
}

type SetProviderAPIKeyResponse struct{}

type CompletionRequest struct {
	Runtime *RuntimeModel `json:"-"`

	History        []conversationSpec.ConversationMessage  `json:"-"`
	Current        conversationSpec.ConversationMessage    `json:"-"`
	ToolSelections []conversationSpec.ToolSelection        `json:"-"`
	MCPContext     *mcpConversation.MCPConversationContext `json:"-"`
	SkillSessionID string                                  `json:"-"`

	OnStreamText     func(text string) error     `json:"-"`
	OnStreamThinking func(thinking string) error `json:"-"`
}

type CompletionResponseBody struct {
	InferenceResponse     *inferenceSpec.FetchCompletionResponse `json:"inferenceResponse,omitempty"`
	HydratedCurrentInputs []inferenceSpec.InputUnion             `json:"hydratedCurrentInputs,omitempty"`

	// MCPToolMappings must be persisted on the corresponding conversation
	// user turn before a later provider-tool call is routed through
	// InvokeMappedMCPTool. Conversation storage validates the mappings against
	// that turn's MCPContext.
	MCPToolMappings []mcpConversation.MCPProviderToolMapping `json:"mcpToolMappings,omitempty"`
	WorkspaceUsage  *workspaceConversation.ConversationUsage `json:"workspaceUsage,omitempty"`
}

type CompletionResponse struct {
	Body *CompletionResponseBody
}
