package main

import (
	"context"
	"errors"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	toolnewAggregate "github.com/flexigpt/flexigpt-app/internal/toolnew/aggregate"
	toolnewRuntime "github.com/flexigpt/flexigpt-app/internal/toolnew/runtime"
	toolnewDomain "github.com/flexigpt/flexigpt-app/internal/toolnew/store/domain"
)

type ToolNewAggregateWrapper struct {
	service *toolnewAggregate.Service
}

func InitToolNewAggregateWrapper(
	wrapper *ToolNewAggregateWrapper,
	storeWrapper *ToolNewStoreWrapper,
	runtimeWrapper *ToolNewRuntimeWrapper,
) error {
	if wrapper == nil ||
		storeWrapper == nil ||
		runtimeWrapper == nil {
		return errors.New("tool aggregate wrapper dependencies are incomplete")
	}
	if storeWrapper.api == nil || runtimeWrapper.service == nil {
		return basespec.ErrClosed
	}

	service, err := toolnewAggregate.New(
		storeWrapper.api,
		runtimeWrapper.service,
	)
	if err != nil {
		return err
	}

	wrapper.service = service
	return nil
}

func withToolNewAggregate[T any](
	w *ToolNewAggregateWrapper,
	fn func(*toolnewAggregate.Service) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if err := w.ready(); err != nil {
			return zero, err
		}
		return fn(w.service)
	})
}

func (w *ToolNewAggregateWrapper) ResolveMappedTool(
	target resolve.MappedTarget,
) (toolnewDomain.ResolvedTool, error) {
	return withToolNewAggregate(
		w,
		func(service *toolnewAggregate.Service) (
			toolnewDomain.ResolvedTool,
			error,
		) {
			return service.ResolveMappedTool(
				context.Background(),
				target,
			)
		},
	)
}

func (w *ToolNewAggregateWrapper) InvokeMappedTool(
	request toolnewAggregate.InvokeRequest,
) (*toolnewRuntime.InvokeResponse, error) {
	return withToolNewAggregate(
		w,
		func(service *toolnewAggregate.Service) (
			*toolnewRuntime.InvokeResponse,
			error,
		) {
			return service.Invoke(context.Background(), request)
		},
	)
}

func (w *ToolNewAggregateWrapper) HydrateInferenceToolChoice(
	selection toolnewAggregate.ToolSelection,
) (inferenceSpec.ToolChoice, error) {
	return withToolNewAggregate(
		w,
		func(service *toolnewAggregate.Service) (
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

func (w *ToolNewAggregateWrapper) ready() error {
	if w == nil || w.service == nil {
		return basespec.ErrClosed
	}
	return nil
}

// targetMappers is composition-only. The aggregate owns ArtifactRef to mapped
// target translation because it resolves enabled Tool Artifacts and their
// containing Tool Collections.
func (w *ToolNewAggregateWrapper) targetMappers() (
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

func (w *ToolNewAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.service = nil
}
