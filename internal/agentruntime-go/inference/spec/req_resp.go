package spec

import (
	"encoding/json"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
)

type PreparedToolBinding struct {
	ChoiceID string `json:"choiceID"`
	Name     string `json:"name"`
	Type     string `json:"type"`

	BindingKind string          `json:"bindingKind"`
	BindingData json.RawMessage `json:"bindingData,omitempty"`

	Required        bool `json:"required"`
	ProviderManaged bool `json:"providerManaged,omitempty"`
	Automatic       bool `json:"automatic,omitempty"`
}

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
	Model        RuntimeModelRequest `json:"-"`
	SystemPrompt *string             `json:"-"`

	History        []conversationSpec.ConversationMessage   `json:"-"`
	Current        conversationSpec.ConversationMessage     `json:"-"`
	ToolSelections []conversationSpec.ToolSelection         `json:"-"`
	MCPContext     *conversationSpec.MCPConversationContext `json:"-"`
	SkillSessionID string                                   `json:"-"`

	OnStreamText     func(text string) error     `json:"-"`
	OnStreamThinking func(thinking string) error `json:"-"`
}

type CompletionResponseBody struct {
	InferenceResponse     *inferenceSpec.FetchCompletionResponse `json:"inferenceResponse,omitempty"`
	HydratedCurrentInputs []inferenceSpec.InputUnion             `json:"hydratedCurrentInputs,omitempty"`

	// MCPToolMappings must be persisted on the corresponding conversation
	// user turn before a later provider-tool call is routed through
	// InvokeMappedMCPTool. Conversation storage preserves this metadata;
	// mapping/context validation and authorization belong at runtime.
	MCPToolMappings      []conversationSpec.MCPProviderToolMapping    `json:"mcpToolMappings,omitempty"`
	WorkspaceUsage       *conversationSpec.WorkspaceConversationUsage `json:"workspaceUsage,omitempty"`
	PreparedToolBindings []PreparedToolBinding                        `json:"preparedToolBindings,omitempty"`
}

type CompletionResponse struct {
	Body *CompletionResponseBody
}
