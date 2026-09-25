package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/tool/llmtoolsadapter"
	toolRuntime "github.com/flexigpt/flexigpt-app/internal/tool/runtime"
)

type ToolRuntimeInvokeRequest struct {
	Function  string                 `json:"function"`
	Args      jsonutil.JSONRawString `json:"args,omitempty"`
	TimeoutMS int                    `json:"timeoutMS,omitempty"`
}

type ToolAggregateInvokeRequest struct {
	Target    resolve.MappedTarget   `json:"target"`
	Args      jsonutil.JSONRawString `json:"args,omitempty"`
	TimeoutMS int                    `json:"timeoutMS,omitempty"`
}

func toolArgumentsFromBridge(
	value jsonutil.JSONRawString,
) (json.RawMessage, error) {
	if strings.TrimSpace(string(value)) == "" {
		return json.RawMessage(`{}`), nil
	}
	return jsonutil.DecodeJSONStringRawInto[json.RawMessage](value)
}

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
	request *ToolRuntimeInvokeRequest,
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
			args, err := toolArgumentsFromBridge(request.Args)
			if err != nil {
				return nil, err
			}
			return service.Invoke(context.Background(), toolRuntime.InvokeRequest{
				Function:  request.Function,
				Args:      args,
				TimeoutMS: request.TimeoutMS,
			})
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
