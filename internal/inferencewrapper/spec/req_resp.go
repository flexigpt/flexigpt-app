package spec

import (
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
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

type CompletionRequestBody struct {
	// Model configuration for this *call*. If nil, the aggregator can fall
	// back to the last non-nil ModelParam from History.
	ModelParam *inferenceSpec.ModelParam `json:"modelParam,omitempty"`

	// Past turns of the conversation, already persisted.
	History []conversationSpec.ConversationMessage `json:"history"`

	// New user turn to complete. Must have Role=user. Typically will have:
	//   - Attachments (ref attachments),
	//   - either:
	//       * pre-built InputUnion(s) in Inputs, or
	//       * just Messages + Attachments and let the aggregator build
	//         the InputUnion for this turn.
	Current conversationSpec.ConversationMessage `json:"current"`

	// ToolStoreChoices is the set of tool-store handles that should be enabled
	// for *this call*.
	//
	// The aggregator always hydrates ToolChoices from tool-store based on this
	// slice; it does NOT infer tools from History[i].ToolChoices or Current.ToolChoices.
	// (Those are persisted for UI/analytics only.)
	ToolStoreChoices []toolSpec.ToolStoreChoice `json:"toolStoreChoices,omitempty"`
	SkillSessionID   string                     `json:"skillSessionID,omitempty"`
}

type CompletionRequest struct {
	Provider      inferenceSpec.ProviderName    `path:"provider"      required:"true"`
	ModelPresetID modelpresetSpec.ModelPresetID `path:"modelPresetID" required:"true"`

	Body *CompletionRequestBody

	OnStreamText     func(text string) error     `json:"-"`
	OnStreamThinking func(thinking string) error `json:"-"`
}

type CompletionResponseBody struct {
	InferenceResponse     *inferenceSpec.FetchCompletionResponse `json:"inferenceResponse,omitempty"`
	HydratedCurrentInputs []inferenceSpec.InputUnion             `json:"hydratedCurrentInputs,omitempty"`
}

type CompletionResponse struct {
	Body *CompletionResponseBody
}
