package spec

import inferenceSpec "github.com/flexigpt/inference-go/spec"

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

// CompletionRequest is the provider-only execution contract.
//
// All source resolution, history selection, attachment materialization,
// instruction composition, Tool binding preparation, Skill preparation, and
// MCP preparation must finish before this value reaches inference.
type CompletionRequest struct {
	Configuration    RuntimeConfiguration       `json:"-"`
	Inputs           []inferenceSpec.InputUnion `json:"-"`
	ToolChoices      []inferenceSpec.ToolChoice `json:"-"`
	OnStreamText     func(string) error         `json:"-"`
	OnStreamThinking func(string) error         `json:"-"`
}

type CompletionResponseBody struct {
	InferenceResponse *inferenceSpec.FetchCompletionResponse `json:"inferenceResponse,omitempty"`
}

type CompletionResponse struct {
	Body *CompletionResponseBody
}
