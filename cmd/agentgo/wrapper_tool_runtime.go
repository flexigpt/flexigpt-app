package main

import (
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/toolruntime"
)

type ToolRuntimeWrapper struct {
	adapter *toolruntime.Adapter
}

func InitToolRuntimeWrapper(
	wrapper *ToolRuntimeWrapper,
) error {
	if wrapper == nil {
		return errors.New("tool runtime wrapper is required")
	}

	adapter, err := toolruntime.NewAdapter()
	if err != nil {
		return err
	}

	wrapper.adapter = adapter
	return nil
}

func (w *ToolRuntimeWrapper) close() {
	if w == nil {
		return
	}
	w.adapter = nil
}
