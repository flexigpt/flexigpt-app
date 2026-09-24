package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/toolnew/llmtoolsadapter"
	toolnewRuntime "github.com/flexigpt/flexigpt-app/internal/toolnew/runtime"
)

type ToolNewRuntimeWrapper struct {
	adapter *llmtoolsadapter.Adapter
	service *toolnewRuntime.Service
}

func InitToolNewRuntimeWrapper(
	wrapper *ToolNewRuntimeWrapper,
) error {
	if wrapper == nil {
		return errors.New("tool runtime wrapper is required")
	}

	adapter, err := llmtoolsadapter.New()
	if err != nil {
		return err
	}

	service, err := toolnewRuntime.New(adapter)
	if err != nil {
		return err
	}

	wrapper.adapter = adapter
	wrapper.service = service
	return nil
}

func withToolNewRuntime[T any](
	w *ToolNewRuntimeWrapper,
	fn func(*toolnewRuntime.Service) (T, error),
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
// usually call ToolNewAggregateWrapper.InvokeMappedTool so artifact identity
// and enablement are resolved before execution.
func (w *ToolNewRuntimeWrapper) InvokeTool(
	request *toolnewRuntime.InvokeRequest,
) (*toolnewRuntime.InvokeResponse, error) {
	return withToolNewRuntime(
		w,
		func(service *toolnewRuntime.Service) (
			*toolnewRuntime.InvokeResponse,
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
func (w *ToolNewRuntimeWrapper) goToolLocator() (
	*llmtoolsadapter.Adapter,
	error,
) {
	if w == nil || w.adapter == nil {
		return nil, basespec.ErrClosed
	}
	return w.adapter, nil
}

func (w *ToolNewRuntimeWrapper) close() {
	if w == nil {
		return
	}
	w.service = nil
	w.adapter = nil
}
