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
	out.ModelPresets = maps.Clone(pp.ModelPresets)
	out.CapabilitiesOverride = cloneModelCapabilitiesOverride(pp.CapabilitiesOverride)
	return out
}

func cloneModelPresetForInference(mp spec.ModelPreset) spec.ModelPreset {
	out := mp

	out.ModelPresetPatch = cloneModelPresetPatch(mp.ModelPresetPatch)
	out.CapabilitiesOverride = cloneModelCapabilitiesOverride(mp.CapabilitiesOverride)
	return out
}

func cloneModelPresetPatch(in spec.ModelPresetPatch) spec.ModelPresetPatch {
	return spec.ModelPresetPatch{
		Stream:                      cloneBoolPtr(in.Stream),
		MaxPromptLength:             cloneIntPtr(in.MaxPromptLength),
		MaxOutputLength:             cloneIntPtr(in.MaxOutputLength),
		Temperature:                 cloneFloat64Ptr(in.Temperature),
		Reasoning:                   cloneReasoningParam(in.Reasoning),
		SystemPrompt:                cloneStringPtr(in.SystemPrompt),
		Timeout:                     cloneIntPtr(in.Timeout),
		OutputParam:                 cloneOutputParam(in.OutputParam),
		StopSequences:               slices.Clone(in.StopSequences),
		AdditionalParametersRawJSON: cloneStringPtr(in.AdditionalParametersRawJSON),
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
