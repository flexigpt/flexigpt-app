package main

import (
	"context"
	"errors"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	toolAggregate "github.com/flexigpt/flexigpt-app/internal/tool/aggregate"
	toolRuntime "github.com/flexigpt/flexigpt-app/internal/tool/runtime"
	toolConsumerAPI "github.com/flexigpt/flexigpt-app/internal/tool/store/consumerapi"
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

func (w *ToolAggregateWrapper) MapToolTarget(
	ref artifact.ArtifactRef,
) (resolve.MappedTarget, error) {
	return withToolAggregate(
		w,
		func(service *toolAggregate.Service) (resolve.MappedTarget, error) {
			return service.MapToolTarget(context.Background(), ref)
		},
	)
}

func (w *ToolAggregateWrapper) ResolveMappedTool(
	target resolve.MappedTarget,
) (toolConsumerAPI.ResolvedToolView, error) {
	return withToolAggregate(
		w,
		func(service *toolAggregate.Service) (
			toolConsumerAPI.ResolvedToolView,
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
	request ToolAggregateInvokeRequest,
) (*toolRuntime.InvokeResponse, error) {
	return withToolAggregate(
		w,
		func(service *toolAggregate.Service) (
			*toolRuntime.InvokeResponse,
			error,
		) {
			args, err := toolArgumentsFromBridge(request.Args)
			if err != nil {
				return nil, err
			}
			return service.Invoke(context.Background(), toolAggregate.InvokeRequest{
				Target:    request.Target,
				Args:      args,
				TimeoutMS: request.TimeoutMS,
			})
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
