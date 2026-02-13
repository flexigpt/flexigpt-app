package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
	"github.com/flexigpt/flexigpt-app/internal/toolruntime"
	"github.com/flexigpt/flexigpt-app/internal/toolruntime/spec"
)

type ToolRuntimeWrapper struct {
	store *toolStore.ToolStore
	tr    *toolruntime.ToolRuntime
}

func InitToolRuntimeWrapper(
	trw *ToolRuntimeWrapper,
	store *toolStore.ToolStore,
) error {
	tr := toolruntime.NewToolRuntime(store)
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
