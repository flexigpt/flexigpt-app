package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
)

// PatchProviderPreset updates a provider preset.
//
// User providers support partial updates to metadata, default model, and capabilities override.
//
// Built-in providers only support overlaying:
//   - isEnabled
//   - defaultModelPresetID
func (s *ModelPresetStore) PatchProviderPreset(
	ctx context.Context, req *spec.PatchProviderPresetRequest,
) (*spec.PatchProviderPresetResponse, error) {
	if req == nil || req.Body == nil || req.ProviderName == "" {
		return nil, fmt.Errorf("%w: providerName required", spec.ErrInvalidDir)
	}
	if err := validateProviderPresetPatchRequestBody(req.Body); err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrInvalidDir, err)
	}
	if req.Body.DefaultModelPresetID != nil {
		if *req.Body.DefaultModelPresetID == "" {
			return nil, fmt.Errorf("%w: defaultModelPresetID cannot be empty", spec.ErrInvalidDir)
		}
		if err := validateModelPresetID(*req.Body.DefaultModelPresetID); err != nil {
			return nil, err
		}
	}

	if currentPP, err := s.builtinData.GetBuiltInProvider(ctx, req.ProviderName); err == nil {
		if hasAnyReadOnlyBuiltInProviderPatch(req.Body) {
			return nil, fmt.Errorf("%w: only isEnabled and defaultModelPresetID can be patched for built-in providers",
				spec.ErrBuiltInReadOnly)
		}
		changed := false
		if req.Body.IsEnabled != nil && currentPP.IsEnabled != *req.Body.IsEnabled {
			if _, err := s.builtinData.SetProviderEnabled(ctx,
				req.ProviderName, *req.Body.IsEnabled,
			); err != nil {
				return nil, err
			}
			changed = true
		}

		// Change default model-preset.
		if req.Body.DefaultModelPresetID != nil &&
			currentPP.DefaultModelPresetID != *req.Body.DefaultModelPresetID {
			if _, err := s.builtinData.SetDefaultModelPreset(
				ctx, req.ProviderName, *req.Body.DefaultModelPresetID,
			); err != nil {
				return nil, err
			}
			changed = true
		}
		if changed {
			slog.Info("patchProviderPreset.builtin", "provider", req.ProviderName)
		}

		return &spec.PatchProviderPresetResponse{}, nil
	}

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

	changed := applyProviderPresetPatch(&pp, req.Body)
	if err := validateProviderPreset(&pp); err != nil {
		return nil, fmt.Errorf("invalid patched provider preset: %w", err)
	}

	if !changed {
		return &spec.PatchProviderPresetResponse{}, nil
	}

	pp.ModifiedAt = time.Now().UTC()
	all.ProviderPresets[req.ProviderName] = pp

	if err := s.writeAllUserPresets(all); err != nil {
		return nil, err
	}

	slog.Info("patchProviderPreset", "provider", req.ProviderName)

	return &spec.PatchProviderPresetResponse{}, nil
}

func hasAnyProviderPatchMutation(body *spec.PatchProviderPresetRequestBody) bool {
	if body == nil {
		return false
	}
	return body.DisplayName != nil ||
		body.SDKType != nil ||
		body.IsEnabled != nil ||
		body.Origin != nil ||
		body.ChatCompletionPathPrefix != nil ||
		body.APIKeyHeaderKey != nil ||
		body.DefaultHeaders != nil ||
		body.DefaultModelPresetID != nil ||
		body.CapabilitiesOverride != nil
}

func hasAnyReadOnlyBuiltInProviderPatch(body *spec.PatchProviderPresetRequestBody) bool {
	if body == nil {
		return false
	}
	return body.DisplayName != nil ||
		body.SDKType != nil ||
		body.Origin != nil ||
		body.ChatCompletionPathPrefix != nil ||
		body.APIKeyHeaderKey != nil ||
		body.DefaultHeaders != nil ||
		body.CapabilitiesOverride != nil
}

func validateProviderPresetPatchRequestBody(body *spec.PatchProviderPresetRequestBody) error {
	if body == nil {
		return errors.New("body is required")
	}
	if !hasAnyProviderPatchMutation(body) {
		return errors.New("at least one provider preset field must be supplied")
	}

	return nil
}

func applyProviderPresetPatch(dst *spec.ProviderPreset, body *spec.PatchProviderPresetRequestBody) bool {
	before := cloneProviderPreset(*dst)

	if body.DisplayName != nil {
		dst.DisplayName = *body.DisplayName
	}
	if body.SDKType != nil {
		dst.SDKType = *body.SDKType
	}
	if body.IsEnabled != nil {
		dst.IsEnabled = *body.IsEnabled
	}
	if body.Origin != nil {
		dst.Origin = *body.Origin
	}
	if body.ChatCompletionPathPrefix != nil {
		dst.ChatCompletionPathPrefix = *body.ChatCompletionPathPrefix
	}
	if body.APIKeyHeaderKey != nil {
		dst.APIKeyHeaderKey = *body.APIKeyHeaderKey
	}
	if body.DefaultHeaders != nil {
		dst.DefaultHeaders = maps.Clone(body.DefaultHeaders)
	}
	if body.DefaultModelPresetID != nil {
		dst.DefaultModelPresetID = *body.DefaultModelPresetID
	}
	if body.CapabilitiesOverride != nil {
		dst.CapabilitiesOverride = cloneModelCapabilitiesOverride(body.CapabilitiesOverride)
	}

	after := cloneProviderPreset(*dst)
	return !reflect.DeepEqual(before, after)
}
