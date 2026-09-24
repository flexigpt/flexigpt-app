package main

import (
	"context"
	"errors"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	toolAggregate "github.com/flexigpt/flexigpt-app/internal/tool/aggregate"
	toolRuntime "github.com/flexigpt/flexigpt-app/internal/tool/runtime"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
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
		return basespec.ErrClosed
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
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if err := w.ready(); err != nil {
			return zero, err
		}
		return fn(w.service)
	})
}

func (w *ToolAggregateWrapper) ResolveMappedTool(
	target resolve.MappedTarget,
) (toolDomain.ResolvedTool, error) {
	return withToolAggregate(
		w,
		func(service *toolAggregate.Service) (
			toolDomain.ResolvedTool,
			error,
		) {
			return service.ResolveMappedTool(
				context.Background(),
				target,
			)
		},
	)
}

func (w *ToolAggregateWrapper) InvokeMappedTool(
	request toolAggregate.InvokeRequest,
) (*toolRuntime.InvokeResponse, error) {
	return withToolAggregate(
		w,
		func(service *toolAggregate.Service) (
			*toolRuntime.InvokeResponse,
			error,
		) {
			return service.Invoke(context.Background(), request)
		},
	)
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
		return basespec.ErrClosed
	}
	return nil
}

// targetMappers is composition-only. The aggregate owns ArtifactRef to mapped
// target translation because it resolves enabled Tool Artifacts and their
// containing Tool Collections.
func (w *ToolAggregateWrapper) targetMappers() (
	map[declaration.Type]resolve.ArtifactTargetMapper,
	error,
) {
	if err := w.ready(); err != nil {
		return nil, err
	}

	return map[declaration.Type]resolve.ArtifactTargetMapper{
		declaration.TypeTool: w.service,
	}, nil
}

func (w *ToolAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.service = nil
}
