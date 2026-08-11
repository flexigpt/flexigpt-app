package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/flexigpt/flexigpt-app/internal/assistantpreset/lookupimpl"
	"github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	assistantpresetStore "github.com/flexigpt/flexigpt-app/internal/assistantpreset/store"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

type AssistantPresetStoreWrapper struct {
	store *assistantpresetStore.AssistantPresetStore
}

func InitAssistantPresetStoreWrapper(
	wrapper *AssistantPresetStoreWrapper,
	baseDirectory string,
	modelPresets *modelpresetStore.ModelPresetStore,
	tools *toolStore.ToolStore,
	skills *skillRuntime.SkillRuntime,
	mcpResolver lookupimpl.MCPServerResolver,
	mcpDiscovery lookupimpl.MCPDiscoveryLookup,
) error {
	if wrapper == nil {
		return errors.New("assistant preset store wrapper is required")
	}
	if modelPresets == nil ||
		tools == nil ||
		skills == nil ||
		mcpResolver == nil ||
		mcpDiscovery == nil {
		return errors.New(
			"artifact-backed assistant preset dependencies are incomplete",
		)
	}

	lookups := lookupimpl.NewAssistantPresetReferenceLookups(
		modelPresets,
		tools,
		skills,
		mcpResolver,
		mcpDiscovery,
	)

	store, err := assistantpresetStore.NewAssistantPresetStore(
		baseDirectory,
		assistantpresetStore.WithReferenceLookups(lookups),
	)
	if err != nil {
		return err
	}
	wrapper.store = store
	return nil
}

func (w *AssistantPresetStoreWrapper) PutAssistantPresetBundle(
	request *spec.PutAssistantPresetBundleRequest,
) (*spec.PutAssistantPresetBundleResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.PutAssistantPresetBundleResponse, error) {
			return w.store.PutAssistantPresetBundle(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) PatchAssistantPresetBundle(
	request *spec.PatchAssistantPresetBundleRequest,
) (*spec.PatchAssistantPresetBundleResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.PatchAssistantPresetBundleResponse, error) {
			return w.store.PatchAssistantPresetBundle(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) DeleteAssistantPresetBundle(
	request *spec.DeleteAssistantPresetBundleRequest,
) (*spec.DeleteAssistantPresetBundleResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.DeleteAssistantPresetBundleResponse, error) {
			return w.store.DeleteAssistantPresetBundle(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) ListAssistantPresetBundles(
	request *spec.ListAssistantPresetBundlesRequest,
) (*spec.ListAssistantPresetBundlesResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.ListAssistantPresetBundlesResponse, error) {
			return w.store.ListAssistantPresetBundles(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) PutAssistantPreset(
	request *spec.PutAssistantPresetRequest,
) (*spec.PutAssistantPresetResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.PutAssistantPresetResponse, error) {
			return w.store.PutAssistantPreset(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) PatchAssistantPreset(
	request *spec.PatchAssistantPresetRequest,
) (*spec.PatchAssistantPresetResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.PatchAssistantPresetResponse, error) {
			return w.store.PatchAssistantPreset(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) DeleteAssistantPreset(
	request *spec.DeleteAssistantPresetRequest,
) (*spec.DeleteAssistantPresetResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.DeleteAssistantPresetResponse, error) {
			return w.store.DeleteAssistantPreset(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) GetAssistantPreset(
	request *spec.GetAssistantPresetRequest,
) (*spec.GetAssistantPresetResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.GetAssistantPresetResponse, error) {
			return w.store.GetAssistantPreset(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) ListAssistantPresets(
	request *spec.ListAssistantPresetsRequest,
) (*spec.ListAssistantPresetsResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.ListAssistantPresetsResponse, error) {
			return w.store.ListAssistantPresets(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) close() {
	if w == nil || w.store == nil {
		return
	}
	if err := w.store.Close(); err != nil {
		slog.Error("close assistant preset store", "error", err)
	}
	w.store = nil
}
