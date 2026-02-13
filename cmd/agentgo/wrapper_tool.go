package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

type ToolStoreWrapper struct {
	store *toolStore.ToolStore
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
	t.store = toolStoreAPI
	return nil
}

func (tbw *ToolStoreWrapper) PutToolBundle(
	req *spec.PutToolBundleRequest,
) (*spec.PutToolBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutToolBundleResponse, error) {
		return tbw.store.PutToolBundle(context.Background(), req)
	})
}

func (tbw *ToolStoreWrapper) DeleteToolBundle(
	req *spec.DeleteToolBundleRequest,
) (*spec.DeleteToolBundleResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeleteToolBundleResponse, error) {
		return tbw.store.DeleteToolBundle(context.Background(), req)
	})
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

func (tbw *ToolStoreWrapper) PutTool(
	req *spec.PutToolRequest,
) (*spec.PutToolResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutToolResponse, error) {
		return tbw.store.PutTool(context.Background(), req)
	})
}

func (tbw *ToolStoreWrapper) DeleteTool(
	req *spec.DeleteToolRequest,
) (*spec.DeleteToolResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeleteToolResponse, error) {
		return tbw.store.DeleteTool(context.Background(), req)
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
