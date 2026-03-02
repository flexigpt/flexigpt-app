package spec

import inferencegoSpec "github.com/flexigpt/inference-go/spec"

type ReasoningCapabilitiesOverride struct {
	SupportedReasoningTypes  []inferencegoSpec.ReasoningType  `json:"supportedReasoningTypes,omitempty"`
	SupportedReasoningLevels []inferencegoSpec.ReasoningLevel `json:"supportedReasoningLevels,omitempty"`

	SupportsSummaryStyle             *bool `json:"supportsSummaryStyle,omitempty"`
	SupportsEncryptedReasoningInput  *bool `json:"supportsEncryptedReasoningInput,omitempty"`
	TemperatureDisallowedWhenEnabled *bool `json:"temperatureDisallowedWhenEnabled,omitempty"`
}

type StopSequenceCapabilitiesOverride struct {
	IsSupported             *bool `json:"isSupported,omitempty"`
	DisallowedWithReasoning *bool `json:"disallowedWithReasoning,omitempty"`
	MaxSequences            *int  `json:"maxSequences,omitempty"`
}

type OutputCapabilitiesOverride struct {
	SupportedOutputFormats []inferencegoSpec.OutputFormatKind `json:"supportedOutputFormats,omitempty"`
	SupportsVerbosity      *bool                              `json:"supportsVerbosity,omitempty"`
}

type ToolCapabilitiesOverride struct {
	SupportedToolTypes        []inferencegoSpec.ToolType       `json:"supportedToolTypes,omitempty"`
	SupportedToolPolicyModes  []inferencegoSpec.ToolPolicyMode `json:"supportedToolPolicyModes,omitempty"`
	SupportsParallelToolCalls *bool                            `json:"supportsParallelToolCalls,omitempty"`
	MaxForcedTools            *int                             `json:"maxForcedTools,omitempty"`
}

// ModelCapabilitiesOverride is a "patch-like" version of inference-go's ModelCapabilities.
//
// Semantics:
//   - nil slice => no override provided
//   - empty slice => override to empty (disable completely)
//   - pointer scalars => nil means "not provided", non-nil means "override"
//
// This struct is intended for storage and API transport as an override only.
// The effective/derived capabilities MUST be computed at runtime and should not be stored.
type ModelCapabilitiesOverride struct {
	ModalitiesIn  []inferencegoSpec.Modality `json:"modalitiesIn,omitempty"`
	ModalitiesOut []inferencegoSpec.Modality `json:"modalitiesOut,omitempty"`

	ReasoningCapabilities    *ReasoningCapabilitiesOverride    `json:"reasoningCapabilities,omitempty"`
	StopSequenceCapabilities *StopSequenceCapabilitiesOverride `json:"stopSequenceCapabilities,omitempty"`
	OutputCapabilities       *OutputCapabilitiesOverride       `json:"outputCapabilities,omitempty"`
	ToolCapabilities         *ToolCapabilitiesOverride         `json:"toolCapabilities,omitempty"`
}
