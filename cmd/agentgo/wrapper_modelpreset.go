package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/modelpreset/artifactfallback"
	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
)

type ModelPresetStoreWrapper struct {
	store            *modelpresetStore.ModelPresetStore
	artifactFallback *artifactfallback.Service
}

// InitModelPresetStoreWrapper initialises the wrapped store in `baseDir`.
func InitModelPresetStoreWrapper(
	m *ModelPresetStoreWrapper,
	baseDir string,
) error {
	if m == nil {
		panic("initialising model-preset store wrapper on nil receivers")
	}
	s, err := modelpresetStore.NewModelPresetStore(baseDir)
	if err != nil {
		return err
	}

	fallback, err := artifactfallback.NewService(
		context.Background(),
		s,
		artifactfallback.DefaultBindings()...,
	)
	if err != nil {
		_ = s.Close()
		return err
	}
	m.store = s
	m.artifactFallback = fallback
	return nil
}

func (w *ModelPresetStoreWrapper) PatchDefaultProvider(
	req *spec.PatchDefaultProviderRequest,
) (*spec.PatchDefaultProviderResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchDefaultProviderResponse, error) {
		return w.store.PatchDefaultProvider(context.Background(), req)
	})
}

func (w *ModelPresetStoreWrapper) GetDefaultProvider(
	req *spec.GetDefaultProviderRequest,
) (*spec.GetDefaultProviderResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetDefaultProviderResponse, error) {
		return w.store.GetDefaultProvider(context.Background(), req)
	})
}

func (w *ModelPresetStoreWrapper) PatchProviderPreset(
	req *spec.PatchProviderPresetRequest,
) (*spec.PatchProviderPresetResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchProviderPresetResponse, error) {
		return w.store.PatchProviderPreset(context.Background(), req)
	})
}

func (w *ModelPresetStoreWrapper) ListProviderPresets(
	req *spec.ListProviderPresetsRequest,
) (*spec.ListProviderPresetsResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListProviderPresetsResponse, error) {
		return w.store.ListProviderPresets(context.Background(), req)
	})
}

func (w *ModelPresetStoreWrapper) PostModelPreset(
	req *spec.PostModelPresetRequest,
) (*spec.PostModelPresetResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PostModelPresetResponse, error) {
		return w.store.PostModelPreset(context.Background(), req)
	})
}

func (w *ModelPresetStoreWrapper) PatchModelPreset(
	req *spec.PatchModelPresetRequest,
) (*spec.PatchModelPresetResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchModelPresetResponse, error) {
		return w.store.PatchModelPreset(context.Background(), req)
	})
}

func (w *ModelPresetStoreWrapper) DeleteModelPreset(
	req *spec.DeleteModelPresetRequest,
) (*spec.DeleteModelPresetResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeleteModelPresetResponse, error) {
		return w.store.DeleteModelPreset(context.Background(), req)
	})
}

func (w *ModelPresetStoreWrapper) GetModelPreset(
	req *spec.GetModelPresetRequest,
) (*spec.GetModelPresetResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetModelPresetResponse, error) {
		return w.store.GetModelPreset(context.Background(), req)
	})
}

func (w *ModelPresetStoreWrapper) ResolveMappedModelTarget(
	req *artifactfallback.ResolveTargetRequest,
) (*artifactfallback.ResolveTargetResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactfallback.ResolveTargetResponse, error) {
			if w == nil || w.artifactFallback == nil {
				return nil, errors.New("model Preset artifact fallback is not initialized")
			}
			if req == nil {
				return nil, errors.New("mapped model target request is nil")
			}

			ref, err := w.artifactFallback.ResolveTarget(
				context.Background(),
				req.Target,
			)
			if err != nil {
				return nil, err
			}

			return &artifactfallback.ResolveTargetResponse{
				Body: &artifactfallback.ResolveTargetResponseBody{
					ModelPresetRef: ref,
				},
			}, nil
		},
	)
}

func (s *ModelPresetStoreWrapper) close() {
	if s == nil || s.store == nil {
		return
	}
	s.store.Close()
	s.artifactFallback = nil
}
