package inferencewrapper

import (
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func cloneModelCapabilities(in inferenceSpec.ModelCapabilities) inferenceSpec.ModelCapabilities {
	out := inferenceSpec.ModelCapabilities{
		ModalitiesIn:  slices.Clone(in.ModalitiesIn),
		ModalitiesOut: slices.Clone(in.ModalitiesOut),
	}
	if in.ReasoningCapabilities != nil {
		c := *in.ReasoningCapabilities
		c.SupportedReasoningTypes = slices.Clone(c.SupportedReasoningTypes)
		c.SupportedReasoningLevels = slices.Clone(c.SupportedReasoningLevels)
		out.ReasoningCapabilities = &c
	}
	if in.StopSequenceCapabilities != nil {
		c := *in.StopSequenceCapabilities
		out.StopSequenceCapabilities = &c
	}
	if in.OutputCapabilities != nil {
		c := *in.OutputCapabilities
		c.SupportedOutputFormats = slices.Clone(c.SupportedOutputFormats)
		out.OutputCapabilities = &c
	}
	if in.ToolCapabilities != nil {
		c := *in.ToolCapabilities
		c.SupportedToolTypes = slices.Clone(c.SupportedToolTypes)
		c.SupportedToolPolicyModes = slices.Clone(c.SupportedToolPolicyModes)
		c.SupportedClientToolOutputFormats = slices.Clone(c.SupportedClientToolOutputFormats)
		out.ToolCapabilities = &c
	}
	if in.CacheCapabilities != nil {
		c := *in.CacheCapabilities
		c.TopLevel = cloneCacheControlCapabilities(in.CacheCapabilities.TopLevel)
		c.InputOutputContent = cloneCacheControlCapabilities(in.CacheCapabilities.InputOutputContent)
		c.ReasoningContent = cloneCacheControlCapabilities(in.CacheCapabilities.ReasoningContent)
		c.ToolChoice = cloneCacheControlCapabilities(in.CacheCapabilities.ToolChoice)
		c.ToolCall = cloneCacheControlCapabilities(in.CacheCapabilities.ToolCall)
		c.ToolOutput = cloneCacheControlCapabilities(in.CacheCapabilities.ToolOutput)
		out.CacheCapabilities = &c
	}
	if in.ParamDialect != nil {
		c := *in.ParamDialect
		out.ParamDialect = &c
	}
	return out
}

func cloneCacheControlCapabilities(
	in *inferenceSpec.CacheControlCapabilities,
) *inferenceSpec.CacheControlCapabilities {
	if in == nil {
		return nil
	}
	out := *in
	out.SupportedKinds = slices.Clone(in.SupportedKinds)
	out.SupportedTTLs = slices.Clone(in.SupportedTTLs)
	return &out
}

func applyModelCapabilitiesOverride(
	dst *inferenceSpec.ModelCapabilities,
	ov *spec.ModelCapabilitiesOverride,
) {
	if dst == nil || ov == nil {
		return
	}

	if ov.ModalitiesIn != nil {
		dst.ModalitiesIn = slices.Clone(ov.ModalitiesIn)
	}
	if ov.ModalitiesOut != nil {
		dst.ModalitiesOut = slices.Clone(ov.ModalitiesOut)
	}
	if ov.ParamDialect != nil {
		dst.ParamDialect = &inferenceSpec.ParamDialect{
			MaxOutputTokensParamName: *ov.ParamDialect.MaxOutputTokensParamName,
			ToolChoiceParamStyle:     *ov.ParamDialect.ToolChoiceParamStyle,
		}
	}

	if ov.ReasoningCapabilities != nil {
		if dst.ReasoningCapabilities == nil {
			dst.ReasoningCapabilities = &inferenceSpec.ReasoningCapabilities{}
		}
		if ov.ReasoningCapabilities.SupportedReasoningTypes != nil {
			dst.ReasoningCapabilities.SupportedReasoningTypes = slices.Clone(
				ov.ReasoningCapabilities.SupportedReasoningTypes,
			)
		}
		if ov.ReasoningCapabilities.SupportedReasoningLevels != nil {
			dst.ReasoningCapabilities.SupportedReasoningLevels = slices.Clone(
				ov.ReasoningCapabilities.SupportedReasoningLevels,
			)
		}
		if ov.ReasoningCapabilities.SupportsReasoningConfig != nil {
			dst.ReasoningCapabilities.SupportsReasoningConfig = *ov.ReasoningCapabilities.SupportsReasoningConfig
		}
		if ov.ReasoningCapabilities.SupportsSummaryStyle != nil {
			dst.ReasoningCapabilities.SupportsSummaryStyle = *ov.ReasoningCapabilities.SupportsSummaryStyle
		}
		if ov.ReasoningCapabilities.SupportsEncryptedReasoningInput != nil {
			dst.ReasoningCapabilities.SupportsEncryptedReasoningInput = *ov.ReasoningCapabilities.SupportsEncryptedReasoningInput
		}
		if ov.ReasoningCapabilities.TemperatureDisallowedWhenEnabled != nil {
			dst.ReasoningCapabilities.TemperatureDisallowedWhenEnabled = *ov.ReasoningCapabilities.TemperatureDisallowedWhenEnabled
		}
	}

	if ov.StopSequenceCapabilities != nil {
		if dst.StopSequenceCapabilities == nil {
			dst.StopSequenceCapabilities = &inferenceSpec.StopSequenceCapabilities{}
		}
		if ov.StopSequenceCapabilities.IsSupported != nil {
			dst.StopSequenceCapabilities.IsSupported = *ov.StopSequenceCapabilities.IsSupported
		}
		if ov.StopSequenceCapabilities.DisallowedWithReasoning != nil {
			dst.StopSequenceCapabilities.DisallowedWithReasoning = *ov.StopSequenceCapabilities.DisallowedWithReasoning
		}
		if ov.StopSequenceCapabilities.MaxSequences != nil {
			dst.StopSequenceCapabilities.MaxSequences = *ov.StopSequenceCapabilities.MaxSequences
		}
	}

	if ov.OutputCapabilities != nil {
		if dst.OutputCapabilities == nil {
			dst.OutputCapabilities = &inferenceSpec.OutputCapabilities{}
		}
		if ov.OutputCapabilities.SupportedOutputFormats != nil {
			dst.OutputCapabilities.SupportedOutputFormats = slices.Clone(ov.OutputCapabilities.SupportedOutputFormats)
		}
		if ov.OutputCapabilities.SupportsVerbosity != nil {
			dst.OutputCapabilities.SupportsVerbosity = *ov.OutputCapabilities.SupportsVerbosity
		}
	}

	if ov.ToolCapabilities != nil {
		if dst.ToolCapabilities == nil {
			dst.ToolCapabilities = &inferenceSpec.ToolCapabilities{}
		}
		if ov.ToolCapabilities.SupportedToolTypes != nil {
			dst.ToolCapabilities.SupportedToolTypes = slices.Clone(ov.ToolCapabilities.SupportedToolTypes)
		}
		if ov.ToolCapabilities.SupportedToolPolicyModes != nil {
			dst.ToolCapabilities.SupportedToolPolicyModes = slices.Clone(ov.ToolCapabilities.SupportedToolPolicyModes)
		}
		if ov.ToolCapabilities.SupportedClientToolOutputFormats != nil {
			dst.ToolCapabilities.SupportedClientToolOutputFormats = slices.Clone(
				ov.ToolCapabilities.SupportedClientToolOutputFormats,
			)
		}
		if ov.ToolCapabilities.SupportsParallelToolCalls != nil {
			dst.ToolCapabilities.SupportsParallelToolCalls = *ov.ToolCapabilities.SupportsParallelToolCalls
		}
		if ov.ToolCapabilities.MaxForcedTools != nil {
			dst.ToolCapabilities.MaxForcedTools = *ov.ToolCapabilities.MaxForcedTools
		}
	}

	if ov.CacheCapabilities != nil {
		if dst.CacheCapabilities == nil {
			dst.CacheCapabilities = &inferenceSpec.CacheCapabilities{}
		}
		if ov.CacheCapabilities.SupportsAutomaticCaching != nil {
			dst.CacheCapabilities.SupportsAutomaticCaching = *ov.CacheCapabilities.SupportsAutomaticCaching
		}

		applyCacheControlCapabilitiesOverride(
			&dst.CacheCapabilities.TopLevel,
			ov.CacheCapabilities.TopLevel,
		)
		applyCacheControlCapabilitiesOverride(
			&dst.CacheCapabilities.InputOutputContent,
			ov.CacheCapabilities.InputOutputContent,
		)
		applyCacheControlCapabilitiesOverride(
			&dst.CacheCapabilities.ReasoningContent,
			ov.CacheCapabilities.ReasoningContent,
		)
		applyCacheControlCapabilitiesOverride(
			&dst.CacheCapabilities.ToolChoice,
			ov.CacheCapabilities.ToolChoice,
		)
		applyCacheControlCapabilitiesOverride(
			&dst.CacheCapabilities.ToolCall,
			ov.CacheCapabilities.ToolCall,
		)
		applyCacheControlCapabilitiesOverride(
			&dst.CacheCapabilities.ToolOutput,
			ov.CacheCapabilities.ToolOutput,
		)
	}
}

func applyCacheControlCapabilitiesOverride(
	dst **inferenceSpec.CacheControlCapabilities,
	ov *spec.CacheControlCapabilitiesOverride,
) {
	if ov == nil {
		return
	}
	if *dst == nil {
		*dst = &inferenceSpec.CacheControlCapabilities{}
	}
	if ov.SupportedKinds != nil {
		(*dst).SupportedKinds = slices.Clone(ov.SupportedKinds)
	}
	if ov.SupportedTTLs != nil {
		(*dst).SupportedTTLs = slices.Clone(ov.SupportedTTLs)
	}
	if ov.SupportsKey != nil {
		(*dst).SupportsKey = *ov.SupportsKey
	}
	if ov.SupportsTTL != nil {
		(*dst).SupportsTTL = *ov.SupportsTTL
	}
}
