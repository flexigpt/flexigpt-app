package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	promptSpec "github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

func validateAssistantPresetBundle(bundle *spec.AssistantPresetBundle) error {
	if bundle == nil {
		return errors.New("bundle is nil")
	}
	if bundle.SchemaVersion != spec.SchemaVersion {
		return fmt.Errorf(
			"schemaVersion %q does not match expected %q",
			bundle.SchemaVersion,
			spec.SchemaVersion,
		)
	}
	if strings.TrimSpace(string(bundle.ID)) == "" {
		return errors.New("bundle id is empty")
	}
	if err := bundleitemutils.ValidateBundleSlug(bundle.Slug); err != nil {
		return fmt.Errorf("invalid bundle slug: %w", err)
	}
	if strings.TrimSpace(bundle.DisplayName) == "" {
		return errors.New("bundle displayName is empty")
	}
	if bundle.CreatedAt.IsZero() {
		return errors.New("bundle createdAt is zero")
	}
	if bundle.ModifiedAt.IsZero() {
		return errors.New("bundle modifiedAt is zero")
	}
	return nil
}

func validateAssistantPreset(
	ctx context.Context,
	preset *spec.AssistantPreset,
	lookups ReferenceLookups,
) error {
	if err := validateAssistantPresetStructure(preset); err != nil {
		return err
	}
	return validateAssistantPresetReferences(ctx, preset, lookups)
}

func validateAssistantPresetStructure(preset *spec.AssistantPreset) error {
	if preset == nil {
		return spec.ErrNilAssistantPreset
	}
	if preset.SchemaVersion != spec.SchemaVersion {
		return fmt.Errorf(
			"schemaVersion %q does not match expected %q",
			preset.SchemaVersion,
			spec.SchemaVersion,
		)
	}
	if strings.TrimSpace(string(preset.ID)) == "" {
		return errors.New("assistant preset id is empty")
	}
	if err := bundleitemutils.ValidateItemSlug(preset.Slug); err != nil {
		return fmt.Errorf("invalid assistant preset slug: %w", err)
	}
	if err := bundleitemutils.ValidateItemVersion(preset.Version); err != nil {
		return fmt.Errorf("invalid assistant preset version: %w", err)
	}
	if strings.TrimSpace(preset.DisplayName) == "" {
		return errors.New("displayName is empty")
	}
	if preset.CreatedAt.IsZero() {
		return errors.New("createdAt is zero")
	}
	if preset.ModifiedAt.IsZero() {
		return errors.New("modifiedAt is zero")
	}
	if err := validateStartingModelPresetPatch(preset.StartingModelPresetPatch); err != nil {
		return err
	}

	seenInstructionRefs := make(map[string]struct{}, len(preset.StartingInstructionTemplateRefs))
	for i, ref := range preset.StartingInstructionTemplateRefs {
		key, err := normalizedJSONKey(ref)
		if err != nil {
			return fmt.Errorf("startingInstructionTemplateRefs[%d]: %w", i, err)
		}
		if _, exists := seenInstructionRefs[key]; exists {
			return fmt.Errorf("startingInstructionTemplateRefs[%d]: duplicate ref", i)
		}
		seenInstructionRefs[key] = struct{}{}
	}

	seenToolRefs := make(map[string]struct{}, len(preset.StartingToolSelections))
	for i, selection := range preset.StartingToolSelections {
		key, err := toolSelectionRefKey(selection)
		if err != nil {
			return fmt.Errorf("startingToolSelections[%d]: %w", i, err)
		}
		if _, exists := seenToolRefs[key]; exists {
			return fmt.Errorf("startingToolSelections[%d]: duplicate toolRef", i)
		}
		seenToolRefs[key] = struct{}{}
	}

	seenSkillRefs := make(map[string]struct{}, len(preset.StartingSkillSelections))
	for i, selection := range preset.StartingSkillSelections {
		key, err := skillSelectionRefKey(selection)
		if err != nil {
			return fmt.Errorf("startingSkillSelections[%d]: %w", i, err)
		}
		if _, exists := seenSkillRefs[key]; exists {
			return fmt.Errorf("startingSkillSelections[%d]: duplicate skillRef", i)
		}
		seenSkillRefs[key] = struct{}{}
	}

	return nil
}

func validateAssistantPresetReferences(
	ctx context.Context,
	preset *spec.AssistantPreset,
	lookups ReferenceLookups,
) error {
	if preset == nil {
		return spec.ErrNilAssistantPreset
	}

	if preset.StartingModelPresetRef != nil {
		if lookups.ModelPresets == nil {
			return errors.New("model preset lookup not configured")
		}
		summary, err := lookups.ModelPresets.GetModelPresetSummary(
			ctx,
			*preset.StartingModelPresetRef,
		)
		if err != nil {
			return fmt.Errorf("startingModelPresetRef: %w", err)
		}
		if !summary.IsEnabled {
			return errors.New("startingModelPresetRef references a disabled model preset")
		}
	}

	for i, ref := range preset.StartingInstructionTemplateRefs {
		if lookups.PromptTemplates == nil {
			return errors.New("prompt template lookup not configured")
		}
		summary, err := lookups.PromptTemplates.GetPromptTemplateSummary(ctx, ref)
		if err != nil {
			return fmt.Errorf("startingInstructionTemplateRefs[%d]: %w", i, err)
		}
		if !summary.IsEnabled {
			return fmt.Errorf(
				"startingInstructionTemplateRefs[%d]: referenced template is disabled",
				i,
			)
		}
		if summary.Kind != promptSpec.PromptTemplateKindInstructionsOnly {
			return fmt.Errorf(
				"startingInstructionTemplateRefs[%d]: template kind must be %q",
				i,
				promptSpec.PromptTemplateKindInstructionsOnly,
			)
		}
		if !summary.IsResolved {
			return fmt.Errorf(
				"startingInstructionTemplateRefs[%d]: template must be resolved",
				i,
			)
		}
	}

	for i, selection := range preset.StartingToolSelections {
		if lookups.ToolSelections == nil {
			return errors.New("tool selection lookup not configured")
		}
		summary, err := lookups.ToolSelections.GetToolSummaryForSelection(ctx, selection)
		if err != nil {
			return fmt.Errorf("startingToolSelections[%d]: %w", i, err)
		}
		if !summary.IsEnabled {
			return fmt.Errorf("startingToolSelections[%d]: referenced tool is disabled", i)
		}
	}

	for i, selection := range preset.StartingSkillSelections {
		if lookups.Skills == nil {
			return errors.New("skill lookup not configured")
		}
		summary, err := lookups.Skills.GetSkillSummaryForSelection(ctx, selection)
		if err != nil {
			return fmt.Errorf("startingSkillSelections[%d]: %w", i, err)
		}
		if !summary.IsEnabled {
			return fmt.Errorf("startingSkillSelections[%d]: referenced skill is disabled", i)
		}
	}

	return nil
}

func validateStartingModelPresetPatch(patch *modelpresetSpec.ModelPresetPatch) error {
	if patch == nil {
		return nil
	}

	hasSystemPrompt, err := jsonFieldPresentAndNonNull(patch, "systemPrompt")
	if err != nil {
		return fmt.Errorf("startingModelPresetPatch: %w", err)
	}
	if hasSystemPrompt {
		return errors.New("startingModelPresetPatch.systemPrompt must be nil")
	}

	hasCapabilitiesOverride, err := jsonFieldPresentAndNonNull(patch, "capabilitiesOverride")
	if err != nil {
		return fmt.Errorf("startingModelPresetPatch: %w", err)
	}
	if hasCapabilitiesOverride {
		return errors.New("startingModelPresetPatch.capabilitiesOverride must be nil")
	}

	return nil
}

func jsonFieldPresentAndNonNull(v any, field string) (bool, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return false, err
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false, err
	}

	fieldRaw, ok := obj[field]
	if !ok {
		return false, nil
	}

	trimmed := bytes.TrimSpace(fieldRaw)
	return !bytes.Equal(trimmed, []byte("null")), nil
}

func normalizedJSONKey(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func toolSelectionRefKey(selection toolSpec.ToolSelection) (string, error) {
	raw, err := json.Marshal(selection)
	if err != nil {
		return "", err
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", errors.New("json unmarshal error")
	}

	if refRaw, ok := obj["toolRef"]; ok && len(bytes.TrimSpace(refRaw)) > 0 {
		return string(refRaw), nil
	}

	// Fallback to the whole serialized selection if toolRef is not present.
	return string(raw), nil
}

func skillSelectionRefKey(selection skillSpec.SkillSelection) (string, error) {
	raw, err := json.Marshal(selection)
	if err != nil {
		return "", err
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", errors.New("json unmarshal error")
	}

	if refRaw, ok := obj["skillRef"]; ok && len(bytes.TrimSpace(refRaw)) > 0 {
		return string(refRaw), nil
	}

	// Fallback to the whole serialized selection if skillRef is not present.
	return string(raw), nil
}
