package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/tool/artifactfallback"
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

type ToolStoreWrapper struct {
	store            *toolStore.ToolStore
	artifactFallback *artifactfallback.Service
}

func InitToolStoreWrapper(
	t *ToolStoreWrapper,
	toolDir string,
) error {
	toolStoreAPI, err := toolStore.NewToolStore(
		toolDir,
	)
	if err != nil {
		return err
	}

	fallback, err := artifactfallback.NewService(
		context.Background(),
		toolStoreAPI,
	)
	if err != nil {
		toolStoreAPI.Close()
		return err
	}
	t.store = toolStoreAPI
	t.artifactFallback = fallback
	return nil
}

func (tbw *ToolStoreWrapper) PatchToolBundle(
	req *spec.PatchToolBundleRequest,
) (*spec.PatchToolBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchToolBundleResponse, error) {
		return tbw.store.PatchToolBundle(context.Background(), req)
	})
}

func (tbw *ToolStoreWrapper) ListToolBundles(
	req *spec.ListToolBundlesRequest,
) (*spec.ListToolBundlesResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListToolBundlesResponse, error) {
		return tbw.store.ListToolBundles(context.Background(), req)
	})
}

func (tbw *ToolStoreWrapper) PatchTool(
	req *spec.PatchToolRequest,
) (*spec.PatchToolResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchToolResponse, error) {
		return tbw.store.PatchTool(context.Background(), req)
	})
}

func (tbw *ToolStoreWrapper) GetTool(
	req *spec.GetToolRequest,
) (*spec.GetToolResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetToolResponse, error) {
		return tbw.store.GetTool(context.Background(), req)
	})
}

func (tbw *ToolStoreWrapper) ListTools(
	req *spec.ListToolsRequest,
) (*spec.ListToolsResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListToolsResponse, error) {
		return tbw.store.ListTools(context.Background(), req)
	})
}

func (tbw *ToolStoreWrapper) ResolveMappedToolTarget(
	req *artifactfallback.ResolveTargetRequest,
) (*artifactfallback.ResolveTargetResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactfallback.ResolveTargetResponse, error) {
			if tbw == nil || tbw.artifactFallback == nil {
				return nil, errors.New("tool artifact fallback is not initialized")
			}
			if req == nil {
				return nil, errors.New(
					"mapped Tool target request is nil",
				)
			}

			ref, err := tbw.artifactFallback.ResolveTarget(
				context.Background(),
				req.Target,
			)
			if err != nil {
				return nil, err
			}

			return &artifactfallback.ResolveTargetResponse{
				Body: &artifactfallback.ResolveTargetResponseBody{
					ToolRef: ref,
				},
			}, nil
		},
	)
}

func (t *ToolStoreWrapper) close() {
	if t == nil || t.store == nil {
		return
	}
	t.store.Close()
	t.store = nil
	t.artifactFallback = nil
}
