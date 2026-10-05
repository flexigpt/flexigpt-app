package main

import (
	"context"
	"errors"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	toolAggregate "github.com/flexigpt/flexigpt-app/internal/tool/aggregate"
)

type ToolAggregateWrapper struct {
	service *toolAggregate.Service
}

func InitToolAggregateWrapper(
	wrapper *ToolAggregateWrapper,
	storeWrapper *ToolStoreWrapper,
	runtimeWrapper *ToolRuntimeWrapper,
) error {
	if wrapper == nil ||
		storeWrapper == nil ||
		runtimeWrapper == nil {
		return errors.New("tool aggregate wrapper dependencies are incomplete")
	}
	if storeWrapper.api == nil || runtimeWrapper.service == nil {
		return spec.ErrClosed
	}

	service, err := toolAggregate.New(
		storeWrapper.api,
		runtimeWrapper.service,
	)
	if err != nil {
		return err
	}

	wrapper.service = service
	return nil
}

func withToolAggregate[T any](
	w *ToolAggregateWrapper,
	fn func(*toolAggregate.Service) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if err := w.ready(); err != nil {
			return zero, err
		}
		return fn(w.service)
	})
}

func (w *ToolAggregateWrapper) HydrateInferenceToolChoice(
	selection toolAggregate.ToolSelection,
) (inferenceSpec.ToolChoice, error) {
	return withToolAggregate(
		w,
		func(service *toolAggregate.Service) (
			inferenceSpec.ToolChoice,
			error,
		) {
			return service.HydrateInferenceToolChoice(
				context.Background(),
				selection,
			)
		},
	)
}

func (w *ToolAggregateWrapper) ready() error {
	if w == nil || w.service == nil {
		return spec.ErrClosed
	}
	return nil
}

func (w *ToolAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.service = nil
}
