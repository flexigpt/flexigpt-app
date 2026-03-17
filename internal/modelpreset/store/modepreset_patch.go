package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
)

// PatchModelPreset updates a model preset.
func (s *ModelPresetStore) PatchModelPreset(
	ctx context.Context, req *spec.PatchModelPresetRequest,
) (*spec.PatchModelPresetResponse, error) {
	if req == nil || req.Body == nil ||
		req.ProviderName == "" || req.ModelPresetID == "" {
		return nil, fmt.Errorf("%w: providerName & modelPresetID required", spec.ErrInvalidDir)
	}

	if err := validateModelPresetID(req.ModelPresetID); err != nil {
		return nil, err
	}
	if err := validateModelPresetPatchRequestBody(req.Body); err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrInvalidDir, err)
	}

	// Built-in branch.
	if _, err := s.builtinData.GetBuiltInProvider(ctx, req.ProviderName); err == nil {
		if hasAnyReadOnlyBuiltInModelPatch(req.Body) {
			return nil, fmt.Errorf("%w: only isEnabled can be patched for built-in model presets",
				spec.ErrBuiltInReadOnly)
		}
		if req.Body.IsEnabled == nil {
			return nil, fmt.Errorf("%w: isEnabled must be supplied for built-in model presets",
				spec.ErrInvalidDir)
		}
		currentMP, err := s.builtinData.GetBuiltInModelPreset(ctx, req.ProviderName, req.ModelPresetID)
		if err != nil {
			return nil, err
		}
		if currentMP.IsEnabled == *req.Body.IsEnabled {
			return &spec.PatchModelPresetResponse{}, nil
		}

		if _, err := s.builtinData.SetModelPresetEnabled(
			ctx,
			req.ProviderName, req.ModelPresetID, *req.Body.IsEnabled,
		); err != nil {
			return nil, err
		}
		slog.Info("patchModelPreset.builtin",
			"provider", req.ProviderName, "modelPresetID", req.ModelPresetID,
			"enabled", *req.Body.IsEnabled)
		return &spec.PatchModelPresetResponse{}, nil
	}

	// User branch.
	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUserPresets(false)
	if err != nil {
		return nil, err
	}
	pp, ok := all.ProviderPresets[req.ProviderName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrProviderNotFound, req.ProviderName)
	}
	mp, ok := pp.ModelPresets[req.ModelPresetID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrModelPresetNotFound, req.ModelPresetID)
	}
	changed := applyModelPresetPatch(&mp, req.Body)

	if err := validateModelPreset(&mp); err != nil {
		return nil, fmt.Errorf("invalid patched model preset: %w", err)
	}
	if !changed {
		return &spec.PatchModelPresetResponse{}, nil
	}

	mp.ModifiedAt = time.Now().UTC()
	pp.ModelPresets[req.ModelPresetID] = mp
	pp.ModifiedAt = mp.ModifiedAt
	all.ProviderPresets[req.ProviderName] = pp

	if err := s.writeAllUserPresets(all); err != nil {
		return nil, err
	}
	slog.Info("patchModelPreset",
		"provider", req.ProviderName, "modelPresetID", req.ModelPresetID,
		"enabled", req.Body.IsEnabled)
	return &spec.PatchModelPresetResponse{}, nil
}

func hasModelPresetPatchValue(p spec.ModelPresetPatch) bool {
	return p.Stream != nil ||
		p.MaxPromptLength != nil ||
		p.MaxOutputLength != nil ||
		p.Temperature != nil ||
		p.Reasoning != nil ||
		p.SystemPrompt != nil ||
		p.Timeout != nil ||
		p.OutputParam != nil ||
		len(p.StopSequences) > 0 ||
		p.AdditionalParametersRawJSON != nil
}

func hasAnyModelPresetPatchMutation(body *spec.PatchModelPresetRequestBody) bool {
	if body == nil {
		return false
	}
	return body.Name != nil ||
		body.Slug != nil ||
		body.DisplayName != nil ||
		body.IsEnabled != nil ||
		hasModelPresetPatchValue(body.ModelPresetPatch) ||
		body.CapabilitiesOverride != nil ||
		body.ClearStopSequences
}

func hasAnyReadOnlyBuiltInModelPatch(body *spec.PatchModelPresetRequestBody) bool {
	if body == nil {
		return false
	}
	return body.Name != nil ||
		body.Slug != nil ||
		body.DisplayName != nil ||
		hasModelPresetPatchValue(body.ModelPresetPatch) ||
		body.CapabilitiesOverride != nil ||
		body.ClearStopSequences
}

func validateModelPresetPatchRequestBody(body *spec.PatchModelPresetRequestBody) error {
	if body == nil {
		return errors.New("body is required")
	}
	if !hasAnyModelPresetPatchMutation(body) {
		return errors.New("at least one model preset field must be supplied")
	}

	if len(body.StopSequences) > 0 && body.ClearStopSequences {
		return errors.New("stopSequences and clearStopSequences cannot both be supplied")
	}

	return nil
}

func applyModelPresetPatch(dst *spec.ModelPreset, body *spec.PatchModelPresetRequestBody) bool {
	before := cloneModelPresetForInference(*dst)

	if body.Name != nil {
		dst.Name = *body.Name
	}
	if body.Slug != nil {
		dst.Slug = *body.Slug
	}
	if body.DisplayName != nil {
		dst.DisplayName = *body.DisplayName
	}
	if body.IsEnabled != nil {
		dst.IsEnabled = *body.IsEnabled
	}

	if body.Stream != nil {
		dst.Stream = cloneBoolPtr(body.Stream)
	}
	if body.MaxPromptLength != nil {
		dst.MaxPromptLength = cloneIntPtr(body.MaxPromptLength)
	}
	if body.MaxOutputLength != nil {
		dst.MaxOutputLength = cloneIntPtr(body.MaxOutputLength)
	}
	if body.Temperature != nil {
		dst.Temperature = cloneFloat64Ptr(body.Temperature)
	}
	if body.Reasoning != nil {
		dst.Reasoning = cloneReasoningParam(body.Reasoning)
	}
	if body.SystemPrompt != nil {
		dst.SystemPrompt = cloneStringPtr(body.SystemPrompt)
	}
	if body.Timeout != nil {
		dst.Timeout = cloneIntPtr(body.Timeout)
	}
	if body.OutputParam != nil {
		dst.OutputParam = cloneOutputParam(body.OutputParam)
	}
	if body.ClearStopSequences {
		dst.StopSequences = nil
	} else if len(body.StopSequences) > 0 {
		dst.StopSequences = slices.Clone(body.StopSequences)
	}
	if body.AdditionalParametersRawJSON != nil {
		dst.AdditionalParametersRawJSON = cloneStringPtr(body.AdditionalParametersRawJSON)
	}

	if body.CapabilitiesOverride != nil {
		dst.CapabilitiesOverride = cloneModelCapabilitiesOverride(body.CapabilitiesOverride)
	}

	after := cloneModelPresetForInference(*dst)
	return !reflect.DeepEqual(before, after)
}
