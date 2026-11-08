package sessionv3

import "slices"

func (s WorkspaceSelectionOverrides) Empty() bool {
	return len(s.Context) == 0 &&
		len(s.Skills) == 0 &&
		len(s.MCP) == 0
}

// ModelDefaultField identifies one provider-neutral request-default field.
// The values match modelruntime.RuntimeDefaultField at the setup boundary.
type ModelDefaultField string

const (
	ModelDefaultFieldAdapterParameters ModelDefaultField = "adapterParameters"
	ModelDefaultFieldCacheControl      ModelDefaultField = "cacheControl"
	ModelDefaultFieldMaxOutputTokens   ModelDefaultField = "maxOutputTokens"
	ModelDefaultFieldMaxPromptTokens   ModelDefaultField = "maxPromptTokens"
	ModelDefaultFieldOutput            ModelDefaultField = "output"
	ModelDefaultFieldReasoning         ModelDefaultField = "reasoning"
	ModelDefaultFieldStopSequences     ModelDefaultField = "stopSequences"
	ModelDefaultFieldStream            ModelDefaultField = "stream"
	ModelDefaultFieldSystemPrompt      ModelDefaultField = "systemPrompt"
	ModelDefaultFieldTemperature       ModelDefaultField = "temperature"
	ModelDefaultFieldTimeoutMS         ModelDefaultField = "timeoutMS"
)

type ModelReasoningOverride struct {
	Type         *string `json:"type,omitempty"`
	Level        *string `json:"level,omitempty"`
	Tokens       *int    `json:"tokens,omitempty"`
	SummaryStyle *string `json:"summaryStyle,omitempty"`
	Context      *string `json:"context,omitempty"`
	Mode         *string `json:"mode,omitempty"`
}

type ModelCacheControlOverride struct {
	Kind *string `json:"kind,omitempty"`
	TTL  *string `json:"ttl,omitempty"`
	Key  *string `json:"key,omitempty"`
}

type ModelOutputJSONSchemaOverride struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Schema      CanonicalValue `json:"schema,omitempty"`
	Strict      *bool          `json:"strict,omitempty"`
}

type ModelOutputFormatOverride struct {
	Kind       *string                        `json:"kind,omitempty"`
	JSONSchema *ModelOutputJSONSchemaOverride `json:"jsonSchema,omitempty"`
}

type ModelOutputOverride struct {
	Verbosity *string                    `json:"verbosity,omitempty"`
	Format    *ModelOutputFormatOverride `json:"format,omitempty"`
}

// ModelDefaultsOverride is the runtime-neutral V3 copy of the typed Model
// request/defaults surface. Setup translates it once through the existing
// modelruntime RuntimeRequestPatch boundary.
type ModelDefaultsOverride struct {
	Stream            *bool                      `json:"stream,omitempty"`
	MaxPromptTokens   *int                       `json:"maxPromptTokens,omitempty"`
	MaxOutputTokens   *int                       `json:"maxOutputTokens,omitempty"`
	Temperature       *float64                   `json:"temperature,omitempty"`
	SystemPrompt      *string                    `json:"systemPrompt,omitempty"`
	TimeoutMS         *int                       `json:"timeoutMS,omitempty"`
	Reasoning         *ModelReasoningOverride    `json:"reasoning,omitempty"`
	CacheControl      *ModelCacheControlOverride `json:"cacheControl,omitempty"`
	Output            *ModelOutputOverride       `json:"output,omitempty"`
	StopSequences     *[]string                  `json:"stopSequences,omitempty"`
	AdapterParameters map[string]CanonicalValue  `json:"adapterParameters,omitempty"`
	Clear             []ModelDefaultField        `json:"clear,omitempty"`
}

func MergeModelDefaultsOverride(
	base ModelDefaultsOverride,
	override ModelDefaultsOverride,
) ModelDefaultsOverride {
	output := base

	if override.Stream != nil {
		output.Stream = override.Stream
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldStream,
		)
	}
	if override.MaxPromptTokens != nil {
		output.MaxPromptTokens = override.MaxPromptTokens
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldMaxPromptTokens,
		)
	}
	if override.MaxOutputTokens != nil {
		output.MaxOutputTokens = override.MaxOutputTokens
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldMaxOutputTokens,
		)
	}
	if override.Temperature != nil {
		output.Temperature = override.Temperature
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldTemperature,
		)
	}
	if override.SystemPrompt != nil {
		output.SystemPrompt = override.SystemPrompt
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldSystemPrompt,
		)
	}
	if override.TimeoutMS != nil {
		output.TimeoutMS = override.TimeoutMS
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldTimeoutMS,
		)
	}
	if override.Reasoning != nil {
		output.Reasoning = override.Reasoning
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldReasoning,
		)
	}
	if override.CacheControl != nil {
		output.CacheControl = override.CacheControl
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldCacheControl,
		)
	}
	if override.Output != nil {
		output.Output = override.Output
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldOutput,
		)
	}
	if override.StopSequences != nil {
		output.StopSequences = override.StopSequences
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldStopSequences,
		)
	}
	if override.AdapterParameters != nil {
		output.AdapterParameters = override.AdapterParameters
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldAdapterParameters,
		)
	}

	for _, field := range override.Clear {
		clearModelDefault(&output, field)
	}
	return output
}

// EffectiveModelDefaultsOverride retains the legacy V3 temperature field as
// an input compatibility layer while all new request-default fields use the
// typed override surface.
func EffectiveModelDefaultsOverride(
	value ModelDefaultsOverride,
	legacyTemperature *float64,
) ModelDefaultsOverride {
	output := value
	if legacyTemperature == nil ||
		output.Temperature != nil ||
		modelDefaultFieldCleared(
			output.Clear,
			ModelDefaultFieldTemperature,
		) {
		return output
	}

	temperature := *legacyTemperature
	output.Temperature = &temperature
	return output
}

func HasModelDefaultsOverride(value ModelDefaultsOverride) bool {
	return value.Stream != nil ||
		value.MaxPromptTokens != nil ||
		value.MaxOutputTokens != nil ||
		value.Temperature != nil ||
		value.SystemPrompt != nil ||
		value.TimeoutMS != nil ||
		value.Reasoning != nil ||
		value.CacheControl != nil ||
		value.Output != nil ||
		value.StopSequences != nil ||
		len(value.AdapterParameters) != 0 ||
		len(value.Clear) != 0
}

func clearModelDefault(
	value *ModelDefaultsOverride,
	field ModelDefaultField,
) {
	if value == nil {
		return
	}

	switch field {
	case ModelDefaultFieldAdapterParameters:
		value.AdapterParameters = nil
	case ModelDefaultFieldCacheControl:
		value.CacheControl = nil
	case ModelDefaultFieldMaxOutputTokens:
		value.MaxOutputTokens = nil
	case ModelDefaultFieldMaxPromptTokens:
		value.MaxPromptTokens = nil
	case ModelDefaultFieldOutput:
		value.Output = nil
	case ModelDefaultFieldReasoning:
		value.Reasoning = nil
	case ModelDefaultFieldStopSequences:
		value.StopSequences = nil
	case ModelDefaultFieldStream:
		value.Stream = nil
	case ModelDefaultFieldSystemPrompt:
		value.SystemPrompt = nil
	case ModelDefaultFieldTemperature:
		value.Temperature = nil
	case ModelDefaultFieldTimeoutMS:
		value.TimeoutMS = nil
	default:
	}

	if !modelDefaultFieldCleared(value.Clear, field) {
		value.Clear = append(value.Clear, field)
	}
}

func removeModelDefaultClear(
	values []ModelDefaultField,
	field ModelDefaultField,
) []ModelDefaultField {
	if len(values) == 0 {
		return nil
	}

	output := make([]ModelDefaultField, 0, len(values))
	for _, value := range values {
		if value != field {
			output = append(output, value)
		}
	}
	return output
}

func modelDefaultFieldCleared(
	values []ModelDefaultField,
	field ModelDefaultField,
) bool {
	return slices.Contains(values, field)
}
