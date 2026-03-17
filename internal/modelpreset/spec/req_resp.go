package spec

import inferenceSpec "github.com/flexigpt/inference-go/spec"

type PatchDefaultProviderRequestBody struct {
	DefaultProvider inferenceSpec.ProviderName `json:"defaultProvider" required:"true"`
}

type PatchDefaultProviderRequest struct {
	Body *PatchDefaultProviderRequestBody
}

type PatchDefaultProviderResponse struct{}

type GetDefaultProviderRequest struct{}

type GetDefaultProviderResponseBody struct {
	DefaultProvider inferenceSpec.ProviderName
}
type GetDefaultProviderResponse struct {
	Body *GetDefaultProviderResponseBody
}

type PostProviderPresetRequestBody struct {
	DisplayName              ProviderDisplayName           `json:"displayName"              required:"true"`
	SDKType                  inferenceSpec.ProviderSDKType `json:"sdkType"                  required:"true"`
	IsEnabled                bool                          `json:"isEnabled"                required:"true"`
	Origin                   string                        `json:"origin"                   required:"true"`
	ChatCompletionPathPrefix string                        `json:"chatCompletionPathPrefix" required:"true"`

	APIKeyHeaderKey      string                     `json:"apiKeyHeaderKey,omitempty"`
	DefaultHeaders       map[string]string          `json:"defaultHeaders,omitempty"`
	CapabilitiesOverride *ModelCapabilitiesOverride `json:"capabilitiesOverride,omitempty"`
}
type PostProviderPresetRequest struct {
	ProviderName inferenceSpec.ProviderName `path:"providerName" required:"true"`
	Body         *PostProviderPresetRequestBody
}

type PostProviderPresetResponse struct{}

// PatchProviderPresetRequestBody patches an existing provider preset.
//
// Semantics:
//   - nil pointer fields => not provided
//   - DefaultHeaders nil => not provided
//   - DefaultHeaders {} => replace with empty map
//   - only user providers can patch provider metadata/capabilities
//   - built-ins only support isEnabled and defaultModelPresetID
type PatchProviderPresetRequestBody struct {
	DisplayName              *ProviderDisplayName           `json:"displayName,omitempty"`
	SDKType                  *inferenceSpec.ProviderSDKType `json:"sdkType,omitempty"`
	IsEnabled                *bool                          `json:"isEnabled,omitempty"`
	Origin                   *string                        `json:"origin,omitempty"`
	ChatCompletionPathPrefix *string                        `json:"chatCompletionPathPrefix,omitempty"`
	APIKeyHeaderKey          *string                        `json:"apiKeyHeaderKey,omitempty"`
	DefaultHeaders           map[string]string              `json:"defaultHeaders,omitempty"`
	DefaultModelPresetID     *ModelPresetID                 `json:"defaultModelPresetID,omitempty"`

	CapabilitiesOverride      *ModelCapabilitiesOverride `json:"capabilitiesOverride,omitempty"`
	ClearCapabilitiesOverride bool                       `json:"clearCapabilitiesOverride,omitempty"`
}

type PatchProviderPresetRequest struct {
	ProviderName inferenceSpec.ProviderName `path:"providerName" required:"true"`
	Body         *PatchProviderPresetRequestBody
}

type PatchProviderPresetResponse struct{}

type DeleteProviderPresetRequest struct {
	ProviderName inferenceSpec.ProviderName `path:"providerName" required:"true"`
}
type DeleteProviderPresetResponse struct{}

type PostModelPresetRequestBody struct {
	ModelPresetPatch

	Name        ModelName        `json:"name"        required:"true"`
	Slug        ModelSlug        `json:"slug"        required:"true"`
	DisplayName ModelDisplayName `json:"displayName" required:"true"`
	IsEnabled   bool             `json:"isEnabled"   required:"true"`

	CapabilitiesOverride *ModelCapabilitiesOverride `json:"capabilitiesOverride,omitempty"`
}

type PostModelPresetRequest struct {
	ProviderName  inferenceSpec.ProviderName `path:"providerName"  required:"true"`
	ModelPresetID ModelPresetID              `path:"modelPresetID" required:"true"`
	Body          *PostModelPresetRequestBody
}
type PostModelPresetResponse struct{}

// PatchModelPresetRequestBody patches a stored model preset.
//
// Semantics:
//   - nil pointer/object fields => not provided
//   - StopSequences nil/empty => not provided
//   - use Clear* flags to explicitly remove stored optional values
//   - at least one field/clear-flag/override field must be supplied
type PatchModelPresetRequestBody struct {
	ModelPresetPatch

	Name        *ModelName        `json:"name,omitempty"`
	Slug        *ModelSlug        `json:"slug,omitempty"`
	DisplayName *ModelDisplayName `json:"displayName,omitempty"`
	IsEnabled   *bool             `json:"isEnabled,omitempty"`

	// CapabilitiesOverride can only be patched for USER provider presets (not built-ins).
	// To clear the override entirely, set ClearCapabilitiesOverride=true.
	CapabilitiesOverride *ModelCapabilitiesOverride `json:"capabilitiesOverride,omitempty"`

	ClearStopSequences bool `json:"clearStopSequences,omitempty"`
}

type PatchModelPresetRequest struct {
	ProviderName  inferenceSpec.ProviderName `path:"providerName"  required:"true"`
	ModelPresetID ModelPresetID              `path:"modelPresetID" required:"true"`
	Body          *PatchModelPresetRequestBody
}
type PatchModelPresetResponse struct{}

type DeleteModelPresetRequest struct {
	ProviderName  inferenceSpec.ProviderName `path:"providerName"  required:"true"`
	ModelPresetID ModelPresetID              `path:"modelPresetID" required:"true"`
}
type DeleteModelPresetResponse struct{}

type GetModelPresetRequest struct {
	ProviderName  inferenceSpec.ProviderName `path:"providerName"  required:"true"`
	ModelPresetID ModelPresetID              `path:"modelPresetID" required:"true"`

	// If false, disabled provider/model return an error.
	IncludeDisabled bool `query:"includeDisabled"`
}

type GetModelPresetResponseBody struct {
	// Provider includes CapabilitiesOverride (provider-wide).
	// ModelPresets map is intentionally omitted/empty in store response for safety/perf.
	Provider ProviderPreset `json:"provider"`

	// Model includes Name + CapabilitiesOverride (model-specific).
	Model ModelPreset `json:"model"`
}

type GetModelPresetResponse struct {
	Body *GetModelPresetResponseBody
}

type ProviderPageToken struct {
	Names           []inferenceSpec.ProviderName `json:"n,omitempty"` //nolint:tagliatelle // PageToken Specific.
	IncludeDisabled bool                         `json:"d,omitempty"` //nolint:tagliatelle // PageToken Specific.
	PageSize        int                          `json:"s,omitempty"` //nolint:tagliatelle // PageToken Specific.
	CursorSlug      inferenceSpec.ProviderName   `json:"c,omitempty"` //nolint:tagliatelle // PageToken Specific.
}

type ListProviderPresetsRequest struct {
	Names           []inferenceSpec.ProviderName `query:"names"`
	IncludeDisabled bool                         `query:"includeDisabled"`
	PageSize        int                          `query:"pageSize"`
	PageToken       string                       `query:"pageToken"`
}
type ListProviderPresetsResponseBody struct {
	Providers     []ProviderPreset `json:"providers"`
	NextPageToken *string          `json:"nextPageToken,omitempty"`
}
type ListProviderPresetsResponse struct {
	Body *ListProviderPresetsResponseBody
}
