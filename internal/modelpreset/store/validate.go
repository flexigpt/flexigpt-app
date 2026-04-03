package store

import (
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

// validateProviderPreset performs structural and referential checks for a
// provider together with its embedded model presets.
func validateProviderPreset(pp *spec.ProviderPreset) error {
	if pp == nil {
		return spec.ErrNilProvider
	}
	if pp.SchemaVersion != spec.SchemaVersion {
		return fmt.Errorf("provider %q: schemaVersion %q not equal to %q",
			pp.Name, pp.SchemaVersion, spec.SchemaVersion)
	}
	if err := validateProviderName(pp.Name); err != nil {
		return fmt.Errorf("provider %q: %w", pp.Name, err)
	}
	if strings.TrimSpace(string(pp.DisplayName)) == "" {
		return fmt.Errorf("provider %q: displayName is empty", pp.Name)
	}
	if pp.CreatedAt.IsZero() || pp.ModifiedAt.IsZero() {
		return fmt.Errorf("provider %q: %w", pp.Name, spec.ErrInvalidTimestamp)
	}
	if strings.TrimSpace(pp.Origin) == "" {
		return fmt.Errorf("provider %q: origin is empty", pp.Name)
	}
	if strings.TrimSpace(pp.ChatCompletionPathPrefix) == "" {
		return fmt.Errorf("provider %q: chatCompletionPathPrefix is empty", pp.Name)
	}
	if err := validateModelCapabilitiesOverride(pp.CapabilitiesOverride); err != nil {
		return fmt.Errorf("provider %q: capabilitiesOverride: %w", pp.Name, err)
	}
	// Per-model validation and duplicate ID detection.
	seenModel := map[spec.ModelPresetID]string{}
	for mid, mp := range pp.ModelPresets {
		if err := validateModelPreset(&mp); err != nil {
			return fmt.Errorf("provider %q, model %q: %w", pp.Name, mid, err)
		}
		if prev := seenModel[mid]; prev != "" {
			return fmt.Errorf("provider %q: duplicate modelPresetID %q (also in %s)",
				pp.Name, mid, prev)
		}
		seenModel[mid] = string(mid)
	}

	// DefaultModelPresetID must exist if set.
	if pp.DefaultModelPresetID != "" {
		if _, ok := pp.ModelPresets[pp.DefaultModelPresetID]; !ok {
			return fmt.Errorf("provider %q: defaultModelPresetID %q not present: %w",
				pp.Name, pp.DefaultModelPresetID, spec.ErrModelPresetNotFound)
		}
	}
	return nil
}

// validateModelPreset performs structural validation for a single model preset.
func validateModelPreset(mp *spec.ModelPreset) error {
	if mp == nil {
		return spec.ErrNilModelPreset
	}
	if mp.SchemaVersion != spec.SchemaVersion {
		return fmt.Errorf("schemaVersion %q not equal to %q",
			mp.SchemaVersion, spec.SchemaVersion)
	}
	if err := validateModelPresetID(mp.ID); err != nil {
		return err
	}
	if err := validateModelName(mp.Name); err != nil {
		return err
	}
	if err := validateModelSlug(mp.Slug); err != nil {
		return err
	}
	if strings.TrimSpace(string(mp.DisplayName)) == "" {
		return errors.New("displayName is empty")
	}
	if mp.CreatedAt.IsZero() || mp.ModifiedAt.IsZero() {
		return spec.ErrInvalidTimestamp
	}

	// Either Reasoning or Temperature must be provided (both cannot be nil).
	if mp.Reasoning == nil && mp.Temperature == nil {
		return errors.New("either reasoning or temperature must be set")
	}

	if mp.MaxPromptLength != nil && *mp.MaxPromptLength < 0 {
		return errors.New("maxPromptLength must be >= 0")
	}
	if mp.MaxOutputLength != nil && *mp.MaxOutputLength < 0 {
		return errors.New("maxOutputLength must be >= 0")
	}
	if mp.Timeout != nil && *mp.Timeout < 0 {
		return errors.New("timeout must be >= 0")
	}

	if err := validateCacheControl(mp.CacheControl); err != nil {
		return fmt.Errorf("invalid cacheControl: %w", err)
	}

	// Reasoning checks (optional).
	if mp.Reasoning != nil {
		if err := validateReasoning(mp.Reasoning); err != nil {
			return fmt.Errorf("invalid reasoning: %w", err)
		}
	}

	if mp.OutputParam != nil {
		if err := validateOutputParam(mp.OutputParam); err != nil {
			return fmt.Errorf("invalid outputParam: %w", err)
		}
	}

	if err := validateStopSequences(mp.StopSequences); err != nil {
		return fmt.Errorf("invalid stopSequences: %w", err)
	}

	if err := validateModelCapabilitiesOverride(mp.CapabilitiesOverride); err != nil {
		return fmt.Errorf("capabilitiesOverride: %w", err)
	}

	return nil
}

func validateStopSequences(stops *[]string) error {
	if stops == nil {
		return nil
	}
	// Keep this conservative across providers (OpenAI chat-completions allows up to 4).
	if len(*stops) > 4 {
		return fmt.Errorf("too many stop sequences: %d (max 4)", len(*stops))
	}
	for i, s := range *stops {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("stopSequences[%d] is empty", i)
		}
	}
	return nil
}

func validateOutputParam(op *inferenceSpec.OutputParam) error {
	if op == nil {
		return nil
	}
	if op.Verbosity != nil {
		switch *op.Verbosity {
		case inferenceSpec.OutputVerbosityLow,
			inferenceSpec.OutputVerbosityMedium,
			inferenceSpec.OutputVerbosityHigh,
			inferenceSpec.OutputVerbosityMax:
			// OK.
		default:
			return fmt.Errorf("unknown verbosity %q", *op.Verbosity)
		}
	}
	if op.Format != nil {
		if err := validateOutputFormat(op.Format); err != nil {
			return err
		}
	}
	return nil
}

func validateOutputFormat(of *inferenceSpec.OutputFormat) error {
	if of == nil {
		return nil
	}
	switch of.Kind {
	case inferenceSpec.OutputFormatKindText:
		if of.JSONSchemaParam != nil {
			return errors.New("jsonSchemaParam must be nil when format.kind is text")
		}
	case inferenceSpec.OutputFormatKindJSONSchema:
		if of.JSONSchemaParam == nil {
			return errors.New("jsonSchemaParam is required when format.kind is jsonSchema")
		}
		if err := validateJSONSchemaParam(of.JSONSchemaParam); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown format.kind %q", of.Kind)
	}
	return nil
}

func validateJSONSchemaParam(j *inferenceSpec.JSONSchemaParam) error {
	if j == nil {
		return nil
	}
	if !isValidJSONSchemaName(j.Name) {
		return fmt.Errorf("invalid jsonSchemaParam.name %q", j.Name)
	}
	if j.Schema == nil {
		return errors.New("jsonSchemaParam.schema is required")
	}
	return nil
}

func isValidJSONSchemaName(s string) bool {
	// Must be a-z, A-Z, 0-9, underscore or dash, max length 64.
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

// validateProviderName currently only trims blanks; extend as required.
func validateProviderName(n inferenceSpec.ProviderName) error {
	if strings.TrimSpace(string(n)) == "" {
		return errors.New("name is empty")
	}
	return nil
}

// validateModelName is similar free-form stub.
func validateModelName(n spec.ModelName) error {
	if strings.TrimSpace(string(n)) == "" {
		return errors.New("name is empty")
	}
	return nil
}

// validateModelSlug uses existing tag validator.
func validateModelSlug(s spec.ModelSlug) error {
	return bundleitemutils.ValidateTag(string(s))
}

// validateModelPresetID uses the same rule set as slugs for now.
func validateModelPresetID(id spec.ModelPresetID) error {
	return bundleitemutils.ValidateTag(string(id))
}

// validateReasoning verifies the type/level/tokens combos.
func validateReasoning(r *inferenceSpec.ReasoningParam) error {
	switch r.Type {
	case inferenceSpec.ReasoningTypeHybridWithTokens:
		if r.Tokens <= 0 {
			return errors.New("tokens must be >0 for hybridWithTokens")
		}
	case inferenceSpec.ReasoningTypeSingleWithLevels:
		switch r.Level {
		case
			inferenceSpec.ReasoningLevelNone,
			inferenceSpec.ReasoningLevelMinimal,
			inferenceSpec.ReasoningLevelLow,
			inferenceSpec.ReasoningLevelMedium,
			inferenceSpec.ReasoningLevelHigh,
			inferenceSpec.ReasoningLevelXHigh:
			// Valid.
		default:
			return fmt.Errorf("invalid level %q for singleWithLevels", r.Level)
		}
	default:
		return fmt.Errorf("unknown type %q", r.Type)
	}

	// SummaryStyle is optional (OpenAI Responses only), but if provided it must be valid.
	if r.SummaryStyle != nil {
		switch *r.SummaryStyle {
		case inferenceSpec.ReasoningSummaryStyleAuto,
			inferenceSpec.ReasoningSummaryStyleConcise,
			inferenceSpec.ReasoningSummaryStyleDetailed:
			// OK.
		default:
			return fmt.Errorf("unknown summaryStyle %q", *r.SummaryStyle)
		}
	}
	return nil
}

func validateModelCapabilitiesOverride(o *spec.ModelCapabilitiesOverride) error {
	if o == nil {
		return nil
	}
	if err := validateModalities(o.ModalitiesIn); err != nil {
		return fmt.Errorf("modalitiesIn: %w", err)
	}
	if err := validateModalities(o.ModalitiesOut); err != nil {
		return fmt.Errorf("modalitiesOut: %w", err)
	}
	if o.ReasoningCapabilities != nil {
		if err := validateReasoningCapabilitiesOverride(o.ReasoningCapabilities); err != nil {
			return fmt.Errorf("reasoningCapabilities: %w", err)
		}
	}
	if o.StopSequenceCapabilities != nil {
		if err := validateStopSequenceCapabilitiesOverride(o.StopSequenceCapabilities); err != nil {
			return fmt.Errorf("stopSequenceCapabilities: %w", err)
		}
	}
	if o.OutputCapabilities != nil {
		if err := validateOutputCapabilitiesOverride(o.OutputCapabilities); err != nil {
			return fmt.Errorf("outputCapabilities: %w", err)
		}
	}
	if o.ToolCapabilities != nil {
		if err := validateToolCapabilitiesOverride(o.ToolCapabilities); err != nil {
			return fmt.Errorf("toolCapabilities: %w", err)
		}
	}
	if o.CacheCapabilities != nil {
		if err := validateCacheCapabilitiesOverride(o.CacheCapabilities); err != nil {
			return fmt.Errorf("cacheCapabilities: %w", err)
		}
	}
	return nil
}

func validateCacheControl(cc *inferenceSpec.CacheControl) error {
	if cc == nil {
		return nil
	}
	if cc.Kind != "" {
		switch cc.Kind {
		case inferenceSpec.CacheControlKindEphemeral:
		default:
			return fmt.Errorf("unknown kind %q", cc.Kind)
		}
	}
	if cc.TTL != "" {
		switch cc.TTL {
		case inferenceSpec.CacheControlTTL5m,
			inferenceSpec.CacheControlTTL1h,
			inferenceSpec.CacheControlTTL24h,
			inferenceSpec.CacheControlTTLInMemory:
		default:
			return fmt.Errorf("unknown ttl %q", cc.TTL)
		}
	}
	return nil
}

func validateCacheCapabilitiesOverride(o *spec.CacheCapabilitiesOverride) error {
	if o == nil {
		return nil
	}
	scopes := []struct {
		name string
		val  *spec.CacheControlCapabilitiesOverride
	}{
		{"topLevel", o.TopLevel},
		{"inputOutputContent", o.InputOutputContent},
		{"reasoningContent", o.ReasoningContent},
		{"toolChoice", o.ToolChoice},
		{"toolCall", o.ToolCall},
		{"toolOutput", o.ToolOutput},
	}
	for _, s := range scopes {
		if s.val != nil {
			if err := validateCacheControlCapabilitiesOverride(s.val); err != nil {
				return fmt.Errorf("%s: %w", s.name, err)
			}
		}
	}
	return nil
}

func validateCacheControlCapabilitiesOverride(o *spec.CacheControlCapabilitiesOverride) error {
	if o == nil {
		return nil
	}
	if o.SupportedKinds != nil {
		seen := map[inferenceSpec.CacheControlKind]struct{}{}
		for i, k := range o.SupportedKinds {
			switch k {
			case inferenceSpec.CacheControlKindEphemeral:
				// OK.
			default:
				return fmt.Errorf("supportedKinds[%d] unknown kind %q", i, k)
			}
			if _, ok := seen[k]; ok {
				return fmt.Errorf("supportedKinds[%d] duplicate %q", i, k)
			}
			seen[k] = struct{}{}
		}
	}
	if o.SupportedTTLs != nil {
		seen := map[inferenceSpec.CacheControlTTL]struct{}{}
		for i, t := range o.SupportedTTLs {
			switch t {
			case inferenceSpec.CacheControlTTL5m,
				inferenceSpec.CacheControlTTL1h,
				inferenceSpec.CacheControlTTL24h,
				inferenceSpec.CacheControlTTLInMemory:
				// OK.
			default:
				return fmt.Errorf("supportedTTLs[%d] unknown TTL %q", i, t)
			}
			if _, ok := seen[t]; ok {
				return fmt.Errorf("supportedTTLs[%d] duplicate %q", i, t)
			}
			seen[t] = struct{}{}
		}
	}
	return nil
}

func validateModalities(mm []inferenceSpec.Modality) error {
	if mm == nil {
		return nil
	}
	seen := map[inferenceSpec.Modality]struct{}{}
	for i, m := range mm {
		if strings.TrimSpace(string(m)) == "" {
			return fmt.Errorf("[%d] empty modality", i)
		}
		switch m {
		case inferenceSpec.ModalityTextIn,
			inferenceSpec.ModalityTextOut,
			inferenceSpec.ModalityImageIn,
			inferenceSpec.ModalityImageOut,
			inferenceSpec.ModalityFileIn,
			inferenceSpec.ModalityFileOut,
			inferenceSpec.ModalityAudioIn,
			inferenceSpec.ModalityAudioOut,
			inferenceSpec.ModalityVideoIn,
			inferenceSpec.ModalityVideoOut:
			// OK.
		default:
			return fmt.Errorf("[%d] unknown modality %q", i, m)
		}
		if _, ok := seen[m]; ok {
			return fmt.Errorf("[%d] duplicate modality %q", i, m)
		}
		seen[m] = struct{}{}
	}
	return nil
}

func validateReasoningCapabilitiesOverride(o *spec.ReasoningCapabilitiesOverride) error {
	if o == nil {
		return nil
	}
	if o.SupportedReasoningTypes != nil {
		seen := map[inferenceSpec.ReasoningType]struct{}{}
		for i, t := range o.SupportedReasoningTypes {
			switch t {
			case inferenceSpec.ReasoningTypeHybridWithTokens,
				inferenceSpec.ReasoningTypeSingleWithLevels:
			default:
				return fmt.Errorf("supportedReasoningTypes[%d] unknown type %q", i, t)
			}
			if _, ok := seen[t]; ok {
				return fmt.Errorf("supportedReasoningTypes[%d] duplicate %q", i, t)
			}
			seen[t] = struct{}{}
		}
	}
	if o.SupportedReasoningLevels != nil {
		seen := map[inferenceSpec.ReasoningLevel]struct{}{}
		for i, l := range o.SupportedReasoningLevels {
			switch l {
			case inferenceSpec.ReasoningLevelNone,
				inferenceSpec.ReasoningLevelMinimal,
				inferenceSpec.ReasoningLevelLow,
				inferenceSpec.ReasoningLevelMedium,
				inferenceSpec.ReasoningLevelHigh,
				inferenceSpec.ReasoningLevelXHigh:
			default:
				return fmt.Errorf("supportedReasoningLevels[%d] unknown level %q", i, l)
			}
			if _, ok := seen[l]; ok {
				return fmt.Errorf("supportedReasoningLevels[%d] duplicate %q", i, l)
			}
			seen[l] = struct{}{}
		}
	}
	return nil
}

func validateStopSequenceCapabilitiesOverride(o *spec.StopSequenceCapabilitiesOverride) error {
	if o == nil {
		return nil
	}
	if o.MaxSequences != nil && *o.MaxSequences < 0 {
		return errors.New("maxSequences must be >= 0")
	}
	return nil
}

func validateOutputCapabilitiesOverride(o *spec.OutputCapabilitiesOverride) error {
	if o == nil {
		return nil
	}
	if o.SupportedOutputFormats != nil {
		seen := map[inferenceSpec.OutputFormatKind]struct{}{}
		for i, k := range o.SupportedOutputFormats {
			switch k {
			case inferenceSpec.OutputFormatKindText,
				inferenceSpec.OutputFormatKindJSONSchema:
			default:
				return fmt.Errorf("supportedOutputFormats[%d] unknown kind %q", i, k)
			}
			if _, ok := seen[k]; ok {
				return fmt.Errorf("supportedOutputFormats[%d] duplicate %q", i, k)
			}
			seen[k] = struct{}{}
		}
	}
	return nil
}

func validateToolCapabilitiesOverride(o *spec.ToolCapabilitiesOverride) error {
	if o == nil {
		return nil
	}
	if o.MaxForcedTools != nil && *o.MaxForcedTools < 0 {
		return errors.New("maxForcedTools must be >= 0")
	}
	if o.SupportedToolTypes != nil {
		seen := map[inferenceSpec.ToolType]struct{}{}
		for i, t := range o.SupportedToolTypes {
			switch t {
			case inferenceSpec.ToolTypeFunction,
				inferenceSpec.ToolTypeCustom,
				inferenceSpec.ToolTypeWebSearch:
			default:
				return fmt.Errorf("supportedToolTypes[%d] unknown type %q", i, t)
			}
			if _, ok := seen[t]; ok {
				return fmt.Errorf("supportedToolTypes[%d] duplicate %q", i, t)
			}
			seen[t] = struct{}{}
		}
	}
	if o.SupportedToolPolicyModes != nil {
		seen := map[inferenceSpec.ToolPolicyMode]struct{}{}
		for i, m := range o.SupportedToolPolicyModes {
			switch m {
			case inferenceSpec.ToolPolicyModeAuto,
				inferenceSpec.ToolPolicyModeAny,
				inferenceSpec.ToolPolicyModeTool,
				inferenceSpec.ToolPolicyModeNone:
			default:
				return fmt.Errorf("supportedToolPolicyModes[%d] unknown mode %q", i, m)
			}
			if _, ok := seen[m]; ok {
				return fmt.Errorf("supportedToolPolicyModes[%d] duplicate %q", i, m)
			}
			seen[m] = struct{}{}
		}
	}
	return nil
}
