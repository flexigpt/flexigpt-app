package sessionv3

import (
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

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
	Name        *string                 `json:"name,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Schema      jsonutil.CanonicalValue `json:"schema,omitempty"`
	Strict      *bool                   `json:"strict,omitempty"`
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
	Stream            *bool                              `json:"stream,omitempty"`
	MaxPromptTokens   *int                               `json:"maxPromptTokens,omitempty"`
	MaxOutputTokens   *int                               `json:"maxOutputTokens,omitempty"`
	Temperature       *float64                           `json:"temperature,omitempty"`
	SystemPrompt      *string                            `json:"systemPrompt,omitempty"`
	TimeoutMS         *int                               `json:"timeoutMS,omitempty"`
	Reasoning         *ModelReasoningOverride            `json:"reasoning,omitempty"`
	CacheControl      *ModelCacheControlOverride         `json:"cacheControl,omitempty"`
	Output            *ModelOutputOverride               `json:"output,omitempty"`
	StopSequences     *[]string                          `json:"stopSequences,omitempty"`
	AdapterParameters map[string]jsonutil.CanonicalValue `json:"adapterParameters,omitempty"`
	Clear             []ModelDefaultField                `json:"clear,omitempty"`
}

func MergeModelDefaultsOverride(
	base ModelDefaultsOverride,
	override ModelDefaultsOverride,
) ModelDefaultsOverride {
	output := cloneModelDefaultsOverride(base)

	if override.Stream != nil {
		output.Stream = cloneModelBool(override.Stream)
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldStream,
		)
	}
	if override.MaxPromptTokens != nil {
		output.MaxPromptTokens = cloneModelInt(override.MaxPromptTokens)
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldMaxPromptTokens,
		)
	}
	if override.MaxOutputTokens != nil {
		output.MaxOutputTokens = cloneModelInt(override.MaxOutputTokens)
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldMaxOutputTokens,
		)
	}
	if override.Temperature != nil {
		output.Temperature = cloneModelFloat(override.Temperature)
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldTemperature,
		)
	}
	if override.SystemPrompt != nil {
		output.SystemPrompt = cloneModelString(override.SystemPrompt)
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldSystemPrompt,
		)
	}
	if override.TimeoutMS != nil {
		output.TimeoutMS = cloneModelInt(override.TimeoutMS)
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldTimeoutMS,
		)
	}
	if override.Reasoning != nil {
		output.Reasoning = mergeModelReasoning(output.Reasoning, override.Reasoning)
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldReasoning,
		)
	}
	if override.CacheControl != nil {
		output.CacheControl = mergeModelCacheControl(output.CacheControl, override.CacheControl)
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldCacheControl,
		)
	}
	if override.Output != nil {
		output.Output = mergeModelOutput(output.Output, override.Output)
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldOutput,
		)
	}
	if override.StopSequences != nil {
		values := append([]string(nil), (*override.StopSequences)...)
		output.StopSequences = &values
		output.Clear = removeModelDefaultClear(
			output.Clear,
			ModelDefaultFieldStopSequences,
		)
	}
	if override.AdapterParameters != nil {
		output.AdapterParameters = make(
			map[string]jsonutil.CanonicalValue,
			len(override.AdapterParameters),
		)
		for key, value := range override.AdapterParameters {
			output.AdapterParameters[key] = value.Clone()
		}
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

func cloneModelDefaultsOverride(
	value ModelDefaultsOverride,
) ModelDefaultsOverride {
	output := value
	output.Stream = cloneModelBool(value.Stream)
	output.MaxPromptTokens = cloneModelInt(value.MaxPromptTokens)
	output.MaxOutputTokens = cloneModelInt(value.MaxOutputTokens)
	output.Temperature = cloneModelFloat(value.Temperature)
	output.SystemPrompt = cloneModelString(value.SystemPrompt)
	output.TimeoutMS = cloneModelInt(value.TimeoutMS)
	output.Reasoning = mergeModelReasoning(nil, value.Reasoning)
	output.CacheControl = mergeModelCacheControl(nil, value.CacheControl)
	output.Output = mergeModelOutput(nil, value.Output)
	if value.StopSequences != nil {
		items := append([]string(nil), (*value.StopSequences)...)
		output.StopSequences = &items
	}
	if value.AdapterParameters != nil {
		output.AdapterParameters = make(
			map[string]jsonutil.CanonicalValue,
			len(value.AdapterParameters),
		)
		for key, item := range value.AdapterParameters {
			output.AdapterParameters[key] = item.Clone()
		}
	}
	output.Clear = append([]ModelDefaultField(nil), value.Clear...)
	return output
}

func mergeModelReasoning(
	base *ModelReasoningOverride,
	override *ModelReasoningOverride,
) *ModelReasoningOverride {
	if base == nil && override == nil {
		return nil
	}
	output := ModelReasoningOverride{}
	if base != nil {
		output.Type = cloneModelString(base.Type)
		output.Level = cloneModelString(base.Level)
		output.Tokens = cloneModelInt(base.Tokens)
		output.SummaryStyle = cloneModelString(base.SummaryStyle)
		output.Context = cloneModelString(base.Context)
		output.Mode = cloneModelString(base.Mode)
	}
	if override != nil {
		if override.Type != nil {
			output.Type = cloneModelString(override.Type)
		}
		if override.Level != nil {
			output.Level = cloneModelString(override.Level)
		}
		if override.Tokens != nil {
			output.Tokens = cloneModelInt(override.Tokens)
		}
		if override.SummaryStyle != nil {
			output.SummaryStyle = cloneModelString(override.SummaryStyle)
		}
		if override.Context != nil {
			output.Context = cloneModelString(override.Context)
		}
		if override.Mode != nil {
			output.Mode = cloneModelString(override.Mode)
		}
	}
	return &output
}

func mergeModelCacheControl(
	base *ModelCacheControlOverride,
	override *ModelCacheControlOverride,
) *ModelCacheControlOverride {
	if base == nil && override == nil {
		return nil
	}
	output := ModelCacheControlOverride{}
	if base != nil {
		output.Kind = cloneModelString(base.Kind)
		output.TTL = cloneModelString(base.TTL)
		output.Key = cloneModelString(base.Key)
	}
	if override != nil {
		if override.Kind != nil {
			output.Kind = cloneModelString(override.Kind)
		}
		if override.TTL != nil {
			output.TTL = cloneModelString(override.TTL)
		}
		if override.Key != nil {
			output.Key = cloneModelString(override.Key)
		}
	}
	return &output
}

func mergeModelOutput(
	base *ModelOutputOverride,
	override *ModelOutputOverride,
) *ModelOutputOverride {
	if base == nil && override == nil {
		return nil
	}
	output := ModelOutputOverride{}
	if base != nil {
		output.Verbosity = cloneModelString(base.Verbosity)
		output.Format = mergeModelOutputFormat(nil, base.Format)
	}
	if override != nil {
		if override.Verbosity != nil {
			output.Verbosity = cloneModelString(override.Verbosity)
		}
		if override.Format != nil {
			output.Format = mergeModelOutputFormat(output.Format, override.Format)
		}
	}
	return &output
}

func mergeModelOutputFormat(
	base *ModelOutputFormatOverride,
	override *ModelOutputFormatOverride,
) *ModelOutputFormatOverride {
	if base == nil && override == nil {
		return nil
	}
	output := ModelOutputFormatOverride{}
	if base != nil {
		output.Kind = cloneModelString(base.Kind)
		output.JSONSchema = mergeModelOutputJSONSchema(nil, base.JSONSchema)
	}
	if override != nil {
		if override.Kind != nil {
			output.Kind = cloneModelString(override.Kind)
		}
		if override.JSONSchema != nil {
			output.JSONSchema = mergeModelOutputJSONSchema(
				output.JSONSchema,
				override.JSONSchema,
			)
		}
	}
	return &output
}

func mergeModelOutputJSONSchema(
	base *ModelOutputJSONSchemaOverride,
	override *ModelOutputJSONSchemaOverride,
) *ModelOutputJSONSchemaOverride {
	if base == nil && override == nil {
		return nil
	}
	output := ModelOutputJSONSchemaOverride{}
	if base != nil {
		output.Name = cloneModelString(base.Name)
		output.Description = cloneModelString(base.Description)
		output.Schema = base.Schema.Clone()
		output.Strict = cloneModelBool(base.Strict)
	}
	if override != nil {
		if override.Name != nil {
			output.Name = cloneModelString(override.Name)
		}
		if override.Description != nil {
			output.Description = cloneModelString(override.Description)
		}
		if len(override.Schema) != 0 {
			output.Schema = override.Schema.Clone()
		}
		if override.Strict != nil {
			output.Strict = cloneModelBool(override.Strict)
		}
	}
	return &output
}

func cloneModelString(value *string) *string { return jsonutil.CloneJSONInValue(value) }
func cloneModelBool(value *bool) *bool       { return jsonutil.CloneJSONInValue(value) }
func cloneModelInt(value *int) *int          { return jsonutil.CloneJSONInValue(value) }
func cloneModelFloat(value *float64) *float64 {
	return jsonutil.CloneJSONInValue(value)
}
