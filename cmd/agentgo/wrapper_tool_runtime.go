package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/flexigpt-app/internal/tool/llmtoolsadapter"
)

type ToolRuntimeWrapper struct {
	adapter *llmtoolsadapter.Adapter
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

	wrapper.adapter = adapter
	return nil
}

// InvokeTool is the low-level local Go Tool runtime endpoint.
//
// It deliberately accepts only a Go function identity. SDK Tools are not
// local executables: normal inference flows hydrate them through
// ToolAggregateWrapper.HydrateInferenceToolChoice.
func (w *ToolRuntimeWrapper) InvokeTool(
	req *llmtoolsutil.InvokeRequest,
) (*llmtoolsutil.InvokeResponse, error) {
	return withRecoveryResp(func() (*llmtoolsutil.InvokeResponse, error) {
		if req == nil {
			return nil, errors.New("invalid arguments: nil request received")
		}
		return llmtoolsutil.Invoke(context.Background(), w.adapter, *req)
	})
}

func (w *ToolRuntimeWrapper) close() {
	if w == nil {
		return
	}
	w.adapter = nil
}
