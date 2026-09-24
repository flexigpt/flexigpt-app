package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/tool/llmtoolsadapter"
	toolRuntime "github.com/flexigpt/flexigpt-app/internal/tool/runtime"
)

type ToolRuntimeWrapper struct {
	adapter *llmtoolsadapter.Adapter
	service *toolRuntime.Service
}

func InitToolRuntimeWrapper(
	wrapper *ToolRuntimeWrapper,
) error {
	if wrapper == nil {
		return errors.New("tool runtime wrapper is required")
	}

	adapter, err := llmtoolsadapter.New()
	if err != nil {
		return err
	}

	service, err := toolRuntime.New(adapter)
	if err != nil {
		return err
	}

	wrapper.adapter = adapter
	wrapper.service = service
	return nil
}

func withToolRuntime[T any](
	w *ToolRuntimeWrapper,
	fn func(*toolRuntime.Service) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.service == nil {
			return zero, basespec.ErrClosed
		}
		return fn(w.service)
	})
}

// InvokeTool is the low-level runtime endpoint. Normal frontend flows should
// usually call ToolAggregateWrapper.InvokeMappedTool so artifact identity
// and enablement are resolved before execution.
func (w *ToolRuntimeWrapper) InvokeTool(
	request *toolRuntime.InvokeRequest,
) (*toolRuntime.InvokeResponse, error) {
	return withToolRuntime(
		w,
		func(service *toolRuntime.Service) (
			*toolRuntime.InvokeResponse,
			error,
		) {
			if request == nil {
				return nil, errors.New("tool runtime request is required")
			}
			return service.Invoke(context.Background(), *request)
		},
	)
}

// goToolLocator exposes the adapter only to application composition. It is
// intentionally unexported so Wails does not generate a frontend endpoint for
// the internal Go-tool registry.
func (w *ToolRuntimeWrapper) goToolLocator() (
	*llmtoolsadapter.Adapter,
	error,
) {
	if w == nil || w.adapter == nil {
		return nil, basespec.ErrClosed
	}
	return w.adapter, nil
}

func (w *ToolRuntimeWrapper) close() {
	if w == nil {
		return
	}
	w.service = nil
	w.adapter = nil
}
