package store

import (
	"maps"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func cloneProviderPresetForInference(pp spec.ProviderPreset) spec.ProviderPreset {
	out := cloneProviderPreset(pp)
	// Never return the full model map for this endpoint (perf + safety).
	out.ModelPresets = nil
	return out
}

func cloneProviderPreset(pp spec.ProviderPreset) spec.ProviderPreset {
	out := pp
	out.DefaultHeaders = maps.Clone(pp.DefaultHeaders)
	out.ModelPresets = cloneModelPresetMap(pp.ModelPresets)
	out.CapabilitiesOverride = cloneModelCapabilitiesOverride(pp.CapabilitiesOverride)
	return out
}

func cloneProviderPresetMap(
	src map[inferenceSpec.ProviderName]spec.ProviderPreset,
) map[inferenceSpec.ProviderName]spec.ProviderPreset {
	dst := make(map[inferenceSpec.ProviderName]spec.ProviderPreset, len(src))
	for k, v := range src {
		dst[k] = cloneProviderPreset(v)
	}
	return dst
}

func cloneModelPresetMap(
	src map[spec.ModelPresetID]spec.ModelPreset,
) map[spec.ModelPresetID]spec.ModelPreset {
	dst := make(map[spec.ModelPresetID]spec.ModelPreset, len(src))
	for k, v := range src {
		dst[k] = cloneModelPreset(v)
	}
	return dst
}

func cloneModelPresetNestedMap(
	src map[inferenceSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset,
) map[inferenceSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset {
	dst := make(map[inferenceSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset, len(src))
	for k, v := range src {
		dst[k] = cloneModelPresetMap(v)
	}
	return dst
}

func cloneModelPreset(mp spec.ModelPreset) spec.ModelPreset {
	out := mp
	out.ModelPresetPatch = cloneModelPresetPatch(mp.ModelPresetPatch)
	return out
}

func cloneModelPresetPatch(in spec.ModelPresetPatch) spec.ModelPresetPatch {
	var stopSequences *[]string
	if in.StopSequences != nil {
		s := slices.Clone(*in.StopSequences)
		stopSequences = &s
	}
	return spec.ModelPresetPatch{
		Stream:                      cloneBoolPtr(in.Stream),
		MaxPromptLength:             cloneIntPtr(in.MaxPromptLength),
		MaxOutputLength:             cloneIntPtr(in.MaxOutputLength),
		Temperature:                 cloneFloat64Ptr(in.Temperature),
		Reasoning:                   cloneReasoningParam(in.Reasoning),
		SystemPrompt:                cloneStringPtr(in.SystemPrompt),
		Timeout:                     cloneIntPtr(in.Timeout),
		OutputParam:                 cloneOutputParam(in.OutputParam),
		StopSequences:               stopSequences,
		AdditionalParametersRawJSON: cloneStringPtr(in.AdditionalParametersRawJSON),
		CapabilitiesOverride:        cloneModelCapabilitiesOverride(in.CapabilitiesOverride),
	}
}

func cloneReasoningParam(in *inferenceSpec.ReasoningParam) *inferenceSpec.ReasoningParam {
	if in == nil {
		return nil
	}
	out := *in
	if in.SummaryStyle != nil {
		ss := *in.SummaryStyle
		out.SummaryStyle = &ss
	}
	return &out
}

func cloneOutputParam(in *inferenceSpec.OutputParam) *inferenceSpec.OutputParam {
	if in == nil {
		return nil
	}
	out := *in
	if in.Verbosity != nil {
		v := *in.Verbosity
		out.Verbosity = &v
	}
	if in.Format != nil {
		f := *in.Format
		if f.JSONSchemaParam != nil {
			j := *f.JSONSchemaParam
			if j.Schema != nil {
				j.Schema = maps.Clone(j.Schema)
			}
			f.JSONSchemaParam = &j
		}
		out.Format = &f
	}
	return &out
}

func cloneModelCapabilitiesOverride(in *spec.ModelCapabilitiesOverride) *spec.ModelCapabilitiesOverride {
	if in == nil {
		return nil
	}
	out := &spec.ModelCapabilitiesOverride{
		ModalitiesIn:  slices.Clone(in.ModalitiesIn),
		ModalitiesOut: slices.Clone(in.ModalitiesOut),
	}
	if in.ReasoningCapabilities != nil {
		out.ReasoningCapabilities = &spec.ReasoningCapabilitiesOverride{
			SupportedReasoningTypes:          slices.Clone(in.ReasoningCapabilities.SupportedReasoningTypes),
			SupportedReasoningLevels:         slices.Clone(in.ReasoningCapabilities.SupportedReasoningLevels),
			SupportsSummaryStyle:             cloneBoolPtr(in.ReasoningCapabilities.SupportsSummaryStyle),
			SupportsEncryptedReasoningInput:  cloneBoolPtr(in.ReasoningCapabilities.SupportsEncryptedReasoningInput),
			TemperatureDisallowedWhenEnabled: cloneBoolPtr(in.ReasoningCapabilities.TemperatureDisallowedWhenEnabled),
		}
	}
	if in.StopSequenceCapabilities != nil {
		out.StopSequenceCapabilities = &spec.StopSequenceCapabilitiesOverride{
			IsSupported:             cloneBoolPtr(in.StopSequenceCapabilities.IsSupported),
			DisallowedWithReasoning: cloneBoolPtr(in.StopSequenceCapabilities.DisallowedWithReasoning),
			MaxSequences:            cloneIntPtr(in.StopSequenceCapabilities.MaxSequences),
		}
	}
	if in.OutputCapabilities != nil {
		out.OutputCapabilities = &spec.OutputCapabilitiesOverride{
			SupportedOutputFormats: slices.Clone(in.OutputCapabilities.SupportedOutputFormats),
			SupportsVerbosity:      cloneBoolPtr(in.OutputCapabilities.SupportsVerbosity),
		}
	}
	if in.ToolCapabilities != nil {
		out.ToolCapabilities = &spec.ToolCapabilitiesOverride{
			SupportedToolTypes:        slices.Clone(in.ToolCapabilities.SupportedToolTypes),
			SupportedToolPolicyModes:  slices.Clone(in.ToolCapabilities.SupportedToolPolicyModes),
			SupportsParallelToolCalls: cloneBoolPtr(in.ToolCapabilities.SupportsParallelToolCalls),
			MaxForcedTools:            cloneIntPtr(in.ToolCapabilities.MaxForcedTools),
		}
	}
	return out
}

func cloneStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneFloat64Ptr(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneBoolPtr(p *bool) *bool {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
