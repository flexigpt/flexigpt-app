package store

import (
	"maps"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
)

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

func cloneProviderPresetForInference(pp spec.ProviderPreset) spec.ProviderPreset {
	out := pp
	// Never return the full model map for this endpoint (perf + safety).
	out.ModelPresets = nil

	out.DefaultHeaders = maps.Clone(pp.DefaultHeaders)
	out.CapabilitiesOverride = cloneModelCapabilitiesOverride(pp.CapabilitiesOverride)
	return out
}

func cloneModelPresetForInference(mp spec.ModelPreset) spec.ModelPreset {
	out := mp

	out.StopSequences = slices.Clone(mp.StopSequences)
	out.Stream = cloneBoolPtr(mp.Stream)
	out.MaxPromptLength = cloneIntPtr(mp.MaxPromptLength)
	out.MaxOutputLength = cloneIntPtr(mp.MaxOutputLength)
	out.Temperature = cloneFloat64Ptr(mp.Temperature)
	out.SystemPrompt = cloneStringPtr(mp.SystemPrompt)
	out.Timeout = cloneIntPtr(mp.Timeout)
	out.AdditionalParametersRawJSON = cloneStringPtr(mp.AdditionalParametersRawJSON)

	// Clone nested inference-go structs (shallow + deep where needed).
	if mp.Reasoning != nil {
		r := *mp.Reasoning
		if mp.Reasoning.SummaryStyle != nil {
			ss := *mp.Reasoning.SummaryStyle
			r.SummaryStyle = &ss
		}
		out.Reasoning = &r
	}

	if mp.OutputParam != nil {
		op := *mp.OutputParam
		if mp.OutputParam.Verbosity != nil {
			v := *mp.OutputParam.Verbosity
			op.Verbosity = &v
		}
		if mp.OutputParam.Format != nil {
			f := *mp.OutputParam.Format
			if f.JSONSchemaParam != nil {
				j := *f.JSONSchemaParam
				// Schema is typically map[string]any; clone defensively if present.
				if j.Schema != nil {
					j.Schema = maps.Clone(j.Schema)
				}
				f.JSONSchemaParam = &j
			}
			op.Format = &f
		}
		out.OutputParam = &op
	}

	out.CapabilitiesOverride = cloneModelCapabilitiesOverride(mp.CapabilitiesOverride)
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
