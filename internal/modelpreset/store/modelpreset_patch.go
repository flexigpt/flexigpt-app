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
		"enabled", mp.IsEnabled)
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
		p.CacheControl != nil ||
		p.OutputParam != nil ||
		p.StopSequences != nil ||
		p.AdditionalParametersRawJSON != nil ||
		p.CapabilitiesOverride != nil
}

func hasAnyModelPresetPatchMutation(body *spec.PatchModelPresetRequestBody) bool {
	if body == nil {
		return false
	}
	return body.Name != nil ||
		body.Slug != nil ||
		body.DisplayName != nil ||
		body.IsEnabled != nil ||
		hasModelPresetPatchValue(body.ModelPresetPatch)
}

func hasAnyReadOnlyBuiltInModelPatch(body *spec.PatchModelPresetRequestBody) bool {
	if body == nil {
		return false
	}
	return body.Name != nil ||
		body.Slug != nil ||
		body.DisplayName != nil ||
		hasModelPresetPatchValue(body.ModelPresetPatch)
}

func validateModelPresetPatchRequestBody(body *spec.PatchModelPresetRequestBody) error {
	if body == nil {
		return errors.New("body is required")
	}
	if !hasAnyModelPresetPatchMutation(body) {
		return errors.New("at least one model preset field must be supplied")
	}

	return nil
}

func applyModelPresetPatch(dst *spec.ModelPreset, body *spec.PatchModelPresetRequestBody) bool {
	before := cloneModelPreset(*dst)

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
	if body.CacheControl != nil {
		dst.CacheControl = cloneCacheControl(body.CacheControl)
	}
	if body.OutputParam != nil {
		dst.OutputParam = cloneOutputParam(body.OutputParam)
	}
	if body.StopSequences != nil {
		s := slices.Clone(*body.StopSequences)
		dst.StopSequences = &s
	}
	if body.AdditionalParametersRawJSON != nil {
		dst.AdditionalParametersRawJSON = cloneStringPtr(body.AdditionalParametersRawJSON)
	}

	if body.CapabilitiesOverride != nil {
		dst.CapabilitiesOverride = cloneModelCapabilitiesOverride(body.CapabilitiesOverride)
	}

	after := cloneModelPreset(*dst)
	return !reflect.DeepEqual(before, after)
}
