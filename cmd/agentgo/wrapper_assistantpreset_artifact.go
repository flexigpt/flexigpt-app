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
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

type AssistantPresetStoreWrapper struct {
	store *assistantpresetStore.AssistantPresetStore
}

func InitAssistantPresetStoreWrapper(
	w *AssistantPresetStoreWrapper,
	baseDir string,
	modelPresetSt *modelpresetStore.ModelPresetStore,
	toolSt *toolStore.ToolStore,
	skillRt *skillAggregate.Service,
	mcpResolver lookupimpl.MCPServerResolver,
	mcpDiscovery lookupimpl.MCPDiscoveryLookup,
) error {
	if w == nil {
		return errors.New(
			"initializing AssistantPresetStoreWrapper on nil receiver",
		)
	}
	if modelPresetSt == nil {
		return errors.New("model preset store is nil")
	}
	if toolSt == nil {
		return errors.New("tool store is nil")
	}
	if skillRt == nil {
		return errors.New("skill runtime is nil")
	}
	if mcpResolver == nil {
		return errors.New("artifact-backed MCP resolver is nil")
	}
	if mcpDiscovery == nil {
		return errors.New("artifact-backed MCP discovery runtime is nil")
	}

	lookups := lookupimpl.NewAssistantPresetReferenceLookups(
		modelPresetSt,
		toolSt,
		skillRt,
		mcpResolver,
		mcpDiscovery,
	)

	store, err := assistantpresetStore.NewAssistantPresetStore(
		baseDir,
		assistantpresetStore.WithReferenceLookups(lookups),
	)
	if err != nil {
		return err
	}

	w.store = store
	return nil
}

func (w *AssistantPresetStoreWrapper) PutAssistantPresetBundle(
	req *spec.PutAssistantPresetBundleRequest,
) (*spec.PutAssistantPresetBundleResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.PutAssistantPresetBundleResponse, error) {
			return w.store.PutAssistantPresetBundle(
				context.Background(),
				req,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) PatchAssistantPresetBundle(
	req *spec.PatchAssistantPresetBundleRequest,
) (*spec.PatchAssistantPresetBundleResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.PatchAssistantPresetBundleResponse, error) {
			return w.store.PatchAssistantPresetBundle(
				context.Background(),
				req,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) DeleteAssistantPresetBundle(
	req *spec.DeleteAssistantPresetBundleRequest,
) (*spec.DeleteAssistantPresetBundleResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.DeleteAssistantPresetBundleResponse, error) {
			return w.store.DeleteAssistantPresetBundle(
				context.Background(),
				req,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) ListAssistantPresetBundles(
	req *spec.ListAssistantPresetBundlesRequest,
) (*spec.ListAssistantPresetBundlesResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.ListAssistantPresetBundlesResponse, error) {
			return w.store.ListAssistantPresetBundles(
				context.Background(),
				req,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) PutAssistantPreset(
	req *spec.PutAssistantPresetRequest,
) (*spec.PutAssistantPresetResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.PutAssistantPresetResponse, error) {
			return w.store.PutAssistantPreset(
				context.Background(),
				req,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) PatchAssistantPreset(
	req *spec.PatchAssistantPresetRequest,
) (*spec.PatchAssistantPresetResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.PatchAssistantPresetResponse, error) {
			return w.store.PatchAssistantPreset(
				context.Background(),
				req,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) DeleteAssistantPreset(
	req *spec.DeleteAssistantPresetRequest,
) (*spec.DeleteAssistantPresetResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.DeleteAssistantPresetResponse, error) {
			return w.store.DeleteAssistantPreset(
				context.Background(),
				req,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) GetAssistantPreset(
	req *spec.GetAssistantPresetRequest,
) (*spec.GetAssistantPresetResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.GetAssistantPresetResponse, error) {
			return w.store.GetAssistantPreset(
				context.Background(),
				req,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) ListAssistantPresets(
	req *spec.ListAssistantPresetsRequest,
) (*spec.ListAssistantPresetsResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*spec.ListAssistantPresetsResponse, error) {
			return w.store.ListAssistantPresets(
				context.Background(),
				req,
			)
		},
	)
}

func (w *AssistantPresetStoreWrapper) close() {
	if w == nil || w.store == nil {
		return
	}
	if err := w.store.Close(); err != nil {
		slog.Error(
			"failed to close artifact-backed assistant preset store",
			"error",
			err,
		)
	}
	w.store = nil
}
