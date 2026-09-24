package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
)

const skillRuntimeCloseTimeout = 30 * time.Second

type SkillRuntimeWrapper struct {
	service *skillRuntime.Service
}

func withSkillRuntime[T any](
	w *SkillRuntimeWrapper,
	fn func(*skillRuntime.Service) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if w == nil {
			return zero, basespec.ErrClosed
		}

		service := w.service
		if service == nil {
			return zero, basespec.ErrClosed
		}
		return fn(service)
	})
}

func InitSkillRuntimeWrapper(
	wrapper *SkillRuntimeWrapper,
	catalogSource skillRuntime.CatalogSource,
	options ...skillRuntime.Option,
) error {
	if wrapper == nil {
		return errors.New("skill runtime wrapper is required")
	}

	configuredOptions := make(
		[]skillRuntime.Option,
		0,
		len(options)+1,
	)
	configuredOptions = append(
		configuredOptions,
		skillRuntime.WithCatalogSource(catalogSource),
	)
	configuredOptions = append(configuredOptions, options...)

	service, err := skillRuntime.New(configuredOptions...)
	if err != nil {
		return err
	}
	wrapper.service = service
	return nil
}

func (w *SkillRuntimeWrapper) CreateSkillSession(
	request *skillRuntime.CreateSkillSessionRequest,
) (*skillRuntime.CreateSkillSessionResponse, error) {
	return withSkillRuntime(
		w,
		func(service *skillRuntime.Service) (
			*skillRuntime.CreateSkillSessionResponse,
			error,
		) {
			return service.CreateSkillSession(
				context.Background(),
				request,
			)
		},
	)
}

func (w *SkillRuntimeWrapper) CloseSkillSession(
	request *skillRuntime.CloseSkillSessionRequest,
) (*skillRuntime.CloseSkillSessionResponse, error) {
	return withSkillRuntime(
		w,
		func(service *skillRuntime.Service) (
			*skillRuntime.CloseSkillSessionResponse,
			error,
		) {
			return service.CloseSkillSession(
				context.Background(),
				request,
			)
		},
	)
}

func (w *SkillRuntimeWrapper) GetSkillsPrompt(
	request *skillRuntime.GetSkillsPromptRequest,
) (*skillRuntime.GetSkillsPromptResponse, error) {
	return withSkillRuntime(
		w,
		func(service *skillRuntime.Service) (
			*skillRuntime.GetSkillsPromptResponse,
			error,
		) {
			return service.GetSkillsPrompt(
				context.Background(),
				request,
			)
		},
	)
}

func (w *SkillRuntimeWrapper) ListSkills(
	request *skillRuntime.ListSkillsRequest,
) (*skillRuntime.ListSkillsResponse, error) {
	return withSkillRuntime(
		w,
		func(service *skillRuntime.Service) (
			*skillRuntime.ListSkillsResponse,
			error,
		) {
			return service.ListSkills(context.Background(), request)
		},
	)
}

func (w *SkillRuntimeWrapper) RenderSkill(
	request *skillRuntime.RenderSkillRequest,
) (*skillRuntime.RenderSkillResponse, error) {
	return withSkillRuntime(
		w,
		func(service *skillRuntime.Service) (
			*skillRuntime.RenderSkillResponse,
			error,
		) {
			return service.RenderSkill(context.Background(), request)
		},
	)
}

func (w *SkillRuntimeWrapper) InvokeSkillTool(
	request *skillRuntime.InvokeSkillToolRequest,
) (*skillRuntime.InvokeSkillToolResponse, error) {
	return withSkillRuntime(
		w,
		func(service *skillRuntime.Service) (
			*skillRuntime.InvokeSkillToolResponse,
			error,
		) {
			return service.InvokeSkillTool(
				context.Background(),
				request,
			)
		},
	)
}

func (w *SkillRuntimeWrapper) close() {
	if w == nil || w.service == nil {
		return
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		skillRuntimeCloseTimeout,
	)
	defer cancel()

	if err := w.service.Close(ctx); err != nil {
		slog.Error("close Skill runtime", "error", err)
	}
	w.service = nil
}
