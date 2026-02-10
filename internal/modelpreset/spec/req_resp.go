package spec

import inferencegoSpec "github.com/flexigpt/inference-go/spec"

type PatchDefaultProviderRequestBody struct {
	DefaultProvider inferencegoSpec.ProviderName `json:"defaultProvider" required:"true"`
}

type PatchDefaultProviderRequest struct {
	Body *PatchDefaultProviderRequestBody
}

type PatchDefaultProviderResponse struct{}

type GetDefaultProviderRequest struct{}

type GetDefaultProviderResponseBody struct {
	DefaultProvider inferencegoSpec.ProviderName
}
type GetDefaultProviderResponse struct {
	Body *GetDefaultProviderResponseBody
}

type PutProviderPresetRequestBody struct {
	DisplayName              ProviderDisplayName             `json:"displayName"               required:"true"`
	SDKType                  inferencegoSpec.ProviderSDKType `json:"sdkType"                   required:"true"`
	IsEnabled                bool                            `json:"isEnabled"                 required:"true"`
	Origin                   string                          `json:"origin"                    required:"true"`
	ChatCompletionPathPrefix string                          `json:"chatCompletionPathPrefix"  required:"true"`
	APIKeyHeaderKey          string                          `json:"apiKeyHeaderKey,omitempty"`
	DefaultHeaders           map[string]string               `json:"defaultHeaders,omitempty"`
}
type PutProviderPresetRequest struct {
	ProviderName inferencegoSpec.ProviderName `path:"providerName" required:"true"`
	Body         *PutProviderPresetRequestBody
}

type PutProviderPresetResponse struct{}

type PatchProviderPresetRequestBody struct {
	IsEnabled            *bool          `json:"isEnabled,omitempty"`
	DefaultModelPresetID *ModelPresetID `json:"defaultModelPresetID,omitempty"`
}

type PatchProviderPresetRequest struct {
	ProviderName inferencegoSpec.ProviderName `path:"providerName" required:"true"`
	Body         *PatchProviderPresetRequestBody
}

type PatchProviderPresetResponse struct{}

type DeleteProviderPresetRequest struct {
	ProviderName inferencegoSpec.ProviderName `path:"providerName" required:"true"`
}
type DeleteProviderPresetResponse struct{}

type PutModelPresetRequestBody struct {
	Name        ModelName        `json:"name"        required:"true"`
	Slug        ModelSlug        `json:"slug"        required:"true"`
	DisplayName ModelDisplayName `json:"displayName" required:"true"`
	IsEnabled   bool             `json:"isEnabled"   required:"true"`

	Stream                      *bool                           `json:"stream,omitempty"`
	MaxPromptLength             *int                            `json:"maxPromptLength,omitempty"`
	MaxOutputLength             *int                            `json:"maxOutputLength,omitempty"`
	Temperature                 *float64                        `json:"temperature,omitempty"`
	Reasoning                   *inferencegoSpec.ReasoningParam `json:"reasoning,omitempty"`
	SystemPrompt                *string                         `json:"systemPrompt,omitempty"`
	Timeout                     *int                            `json:"timeout,omitempty"`
	AdditionalParametersRawJSON *string                         `json:"additionalParametersRawJSON,omitempty"`
}

type PutModelPresetRequest struct {
	ProviderName  inferencegoSpec.ProviderName `path:"providerName"  required:"true"`
	ModelPresetID ModelPresetID                `path:"modelPresetID" required:"true"`
	Body          *PutModelPresetRequestBody
}
type PutModelPresetResponse struct{}

type PatchModelPresetRequestBody struct {
	IsEnabled bool `json:"isEnabled" required:"true"`
}

type PatchModelPresetRequest struct {
	ProviderName  inferencegoSpec.ProviderName `path:"providerName"  required:"true"`
	ModelPresetID ModelPresetID                `path:"modelPresetID" required:"true"`
	Body          *PatchModelPresetRequestBody
}
type PatchModelPresetResponse struct{}

type DeleteModelPresetRequest struct {
	ProviderName  inferencegoSpec.ProviderName `path:"providerName"  required:"true"`
	ModelPresetID ModelPresetID                `path:"modelPresetID" required:"true"`
}
type DeleteModelPresetResponse struct{}

type ProviderPageToken struct {
	Names           []inferencegoSpec.ProviderName `json:"n,omitempty"` //nolint:tagliatelle // PageToken Specific.
	IncludeDisabled bool                           `json:"d,omitempty"` //nolint:tagliatelle // PageToken Specific.
	PageSize        int                            `json:"s,omitempty"` //nolint:tagliatelle // PageToken Specific.
	CursorSlug      inferencegoSpec.ProviderName   `json:"c,omitempty"` //nolint:tagliatelle // PageToken Specific.
}

type ListProviderPresetsRequest struct {
	Names           []inferencegoSpec.ProviderName `query:"names"`
	IncludeDisabled bool                           `query:"includeDisabled"`
	PageSize        int                            `query:"pageSize"`
	PageToken       string                         `query:"pageToken"`
}
type ListProviderPresetsResponseBody struct {
	Providers     []ProviderPreset `json:"providers"`
	NextPageToken *string          `json:"nextPageToken,omitempty"`
}
type ListProviderPresetsResponse struct {
	Body *ListProviderPresetsResponseBody
}
