package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/tool/runtime"
	"github.com/flexigpt/flexigpt-app/internal/tool/runtime/spec"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

type ToolRuntimeWrapper struct {
	store *toolStore.ToolStore
	tr    *runtime.ToolRuntime
}

func InitToolRuntimeWrapper(
	trw *ToolRuntimeWrapper,
	store *toolStore.ToolStore,
) error {
	tr := runtime.NewToolRuntime(store)
	trw.store = store
	trw.tr = tr
	return nil
}

func (trw *ToolRuntimeWrapper) InvokeTool(
	req *spec.InvokeToolRequest,
) (*spec.InvokeToolResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.InvokeToolResponse, error) {
		return trw.tr.InvokeTool(context.Background(), req)
	})
}
