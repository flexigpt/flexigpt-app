package spec

import (
	"errors"
	"time"

	inferencegoSpec "github.com/flexigpt/inference-go/spec"
)

const (
	ModelPresetsFile                     = "modelpresets.json" // Single JSON file.
	ModelPresetsBuiltInOverlayDBFileName = "modelpresetsbuiltin.overlay.sqlite"
)

const (
	SchemaVersion = "2025-07-01"

	MaxPageSize           = 256 // Max allowed page size.
	DefaultPageSize       = 256 // Default page size.
	BuiltInSnapshotMaxAge = time.Hour
)

const (
	DefaultAuthorizationHeaderKey = "Authorization"
	DefaultAPITimeout             = 300 * time.Second

	DefaultAnthropicOrigin                 = "https://api.anthropic.com"
	DefaultAnthropicChatCompletionPrefix   = "/v1/messages"
	DefaultAnthropicAuthorizationHeaderKey = "x-api-key"

	DefaultOpenAIOrigin                = "https://api.openai.com"
	DefaultOpenAIChatCompletionsPrefix = "/v1/chat/completions"
)

var OpenAIChatCompletionsDefaultHeaders = map[string]string{"content-type": "application/json"}

var (
	ErrInvalidDir = errors.New("invalid directory")

	ErrProviderNotFound      = errors.New("provider not found")
	ErrModelPresetNotFound   = errors.New("model preset not found")
	ErrBuiltInProviderAbsent = errors.New("provider not found in built-in data")

	ErrNilProvider      = errors.New("provider preset is nil")
	ErrNilModelPreset   = errors.New("model preset is nil")
	ErrNoModelPresets   = errors.New("provider has no model presets")
	ErrInvalidTimestamp = errors.New("zero timestamp")
	ErrBuiltInReadOnly  = errors.New("built-in resource is read-only")
)

type (
	ModelName        string
	ModelDisplayName string
	ModelSlug        string
	ModelPresetID    string

	ProviderDisplayName string
)

// ModelPreset is the entire "model + default knobs" bundle the user can save.
// Anything not present in the preset is considered to be taken as default from any global or inbuilt model defaults.
type ModelPreset struct {
	SchemaVersion string           `json:"schemaVersion" required:"true"`
	ID            ModelPresetID    `json:"id"            required:"true"`
	Name          ModelName        `json:"name"          required:"true"`
	DisplayName   ModelDisplayName `json:"displayName"   required:"true"`
	Slug          ModelSlug        `json:"slug"          required:"true"`
	IsEnabled     bool             `json:"isEnabled"     required:"true"`

	Stream          *bool                           `json:"stream,omitempty"`
	MaxPromptLength *int                            `json:"maxPromptLength,omitempty"`
	MaxOutputLength *int                            `json:"maxOutputLength,omitempty"`
	Temperature     *float64                        `json:"temperature,omitempty"`
	Reasoning       *inferencegoSpec.ReasoningParam `json:"reasoning,omitempty"`
	SystemPrompt    *string                         `json:"systemPrompt,omitempty"`
	Timeout         *int                            `json:"timeout,omitempty"`

	OutputParam   *inferencegoSpec.OutputParam `json:"outputParam,omitempty"`
	StopSequences []string                     `json:"stopSequences,omitempty"`

	AdditionalParametersRawJSON *string `json:"additionalParametersRawJSON,omitempty"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
	IsBuiltIn  bool      `json:"isBuiltIn"`
}

type ProviderPreset struct {
	SchemaVersion string                          `json:"schemaVersion" required:"true"`
	Name          inferencegoSpec.ProviderName    `json:"name"          required:"true"`
	DisplayName   ProviderDisplayName             `json:"displayName"   required:"true"`
	SDKType       inferencegoSpec.ProviderSDKType `json:"sdkType"       required:"true"`
	IsEnabled     bool                            `json:"isEnabled"     required:"true"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
	IsBuiltIn  bool      `json:"isBuiltIn"`

	Origin                   string            `json:"origin"                   required:"true"`
	ChatCompletionPathPrefix string            `json:"chatCompletionPathPrefix" required:"true"`
	APIKeyHeaderKey          string            `json:"apiKeyHeaderKey"          required:"true"`
	DefaultHeaders           map[string]string `json:"defaultHeaders"`

	DefaultModelPresetID ModelPresetID                 `json:"defaultModelPresetID"`
	ModelPresets         map[ModelPresetID]ModelPreset `json:"modelPresets"`
}

type PresetsSchema struct {
	SchemaVersion   string                                          `json:"schemaVersion"`
	DefaultProvider inferencegoSpec.ProviderName                    `json:"defaultProvider"`
	ProviderPresets map[inferencegoSpec.ProviderName]ProviderPreset `json:"providerPresets"`
}
