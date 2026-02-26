package store

import (
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	inferencegoSpec "github.com/flexigpt/inference-go/spec"
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
			return fmt.Errorf("provider %q: defaultModelPresetID %q not present",
				pp.Name, pp.DefaultModelPresetID)
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
	return nil
}

func validateStopSequences(stops []string) error {
	// Keep this conservative across providers (OpenAI chat-completions allows up to 4).
	if len(stops) > 4 {
		return fmt.Errorf("too many stop sequences: %d (max 4)", len(stops))
	}
	for i, s := range stops {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("stopSequences[%d] is empty", i)
		}
	}
	return nil
}

func validateOutputParam(op *inferencegoSpec.OutputParam) error {
	if op == nil {
		return nil
	}
	if op.Verbosity != nil {
		switch *op.Verbosity {
		case inferencegoSpec.OutputVerbosityLow,
			inferencegoSpec.OutputVerbosityMedium,
			inferencegoSpec.OutputVerbosityHigh,
			inferencegoSpec.OutputVerbosityMax:
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

func validateOutputFormat(of *inferencegoSpec.OutputFormat) error {
	if of == nil {
		return nil
	}
	switch of.Kind {
	case inferencegoSpec.OutputFormatKindText:
		if of.JSONSchemaParam != nil {
			return errors.New("jsonSchemaParam must be nil when format.kind is text")
		}
	case inferencegoSpec.OutputFormatKindJSONSchema:
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

func validateJSONSchemaParam(j *inferencegoSpec.JSONSchemaParam) error {
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
func validateProviderName(n inferencegoSpec.ProviderName) error {
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
func validateReasoning(r *inferencegoSpec.ReasoningParam) error {
	switch r.Type {
	case inferencegoSpec.ReasoningTypeHybridWithTokens:
		if r.Tokens <= 0 {
			return errors.New("tokens must be >0 for hybridWithTokens")
		}
	case inferencegoSpec.ReasoningTypeSingleWithLevels:
		switch r.Level {
		case
			inferencegoSpec.ReasoningLevelNone,
			inferencegoSpec.ReasoningLevelMinimal,
			inferencegoSpec.ReasoningLevelLow,
			inferencegoSpec.ReasoningLevelMedium,
			inferencegoSpec.ReasoningLevelHigh,
			inferencegoSpec.ReasoningLevelXHigh:
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
		case inferencegoSpec.ReasoningSummaryStyleAuto,
			inferencegoSpec.ReasoningSummaryStyleConcise,
			inferencegoSpec.ReasoningSummaryStyleDetailed:
			// OK.
		default:
			return fmt.Errorf("unknown summaryStyle %q", *r.SummaryStyle)
		}
	}
	return nil
}
