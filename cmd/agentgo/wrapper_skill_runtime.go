package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
)

const skillRuntimeCloseTimeout = 30 * time.Second

type SkillRuntimeWrapper struct {
	service *skillRuntime.Service
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

func (w *SkillRuntimeWrapper) SyncSkillCatalog(
	request *skillRuntime.SyncCatalogRequest,
) (*skillRuntime.SyncCatalogResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillRuntime.SyncCatalogResponse, error) {
			if request == nil {
				return nil, errors.New(
					"skill runtime catalog request is required",
				)
			}
			if err := w.service.SyncCatalog(
				context.Background(),
				request.CatalogID,
			); err != nil {
				return nil, err
			}
			return &skillRuntime.SyncCatalogResponse{}, nil
		},
	)
}

func (w *SkillRuntimeWrapper) RemoveSkillCatalog(
	request *skillRuntime.RemoveCatalogRequest,
) (*skillRuntime.RemoveCatalogResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillRuntime.RemoveCatalogResponse, error) {
			if request == nil {
				return nil, errors.New(
					"skill runtime catalog request is required",
				)
			}
			if err := w.service.RemoveCatalog(
				context.Background(),
				request.CatalogID,
			); err != nil {
				return nil, err
			}
			return &skillRuntime.RemoveCatalogResponse{}, nil
		},
	)
}

func (w *SkillRuntimeWrapper) CreateSkillSession(
	request *skillRuntime.CreateSkillSessionRequest,
) (*skillRuntime.CreateSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillRuntime.CreateSkillSessionResponse, error) {
			return w.service.CreateSkillSession(
				context.Background(),
				request,
			)
		},
	)
}

func (w *SkillRuntimeWrapper) CloseSkillSession(
	request *skillRuntime.CloseSkillSessionRequest,
) (*skillRuntime.CloseSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillRuntime.CloseSkillSessionResponse, error) {
			return w.service.CloseSkillSession(
				context.Background(),
				request,
			)
		},
	)
}

func (w *SkillRuntimeWrapper) GetSkillsPrompt(
	request *skillRuntime.GetSkillsPromptRequest,
) (*skillRuntime.GetSkillsPromptResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillRuntime.GetSkillsPromptResponse, error) {
			return w.service.GetSkillsPrompt(
				context.Background(),
				request,
			)
		},
	)
}

func (w *SkillRuntimeWrapper) ListSkills(
	request *skillRuntime.ListSkillsRequest,
) (*skillRuntime.ListSkillsResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillRuntime.ListSkillsResponse, error) {
			return w.service.ListSkills(context.Background(), request)
		},
	)
}

func (w *SkillRuntimeWrapper) RenderSkill(
	request *skillRuntime.RenderSkillRequest,
) (*skillRuntime.RenderSkillResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillRuntime.RenderSkillResponse, error) {
			return w.service.RenderSkill(context.Background(), request)
		},
	)
}

func (w *SkillRuntimeWrapper) InvokeSkillTool(
	request *skillRuntime.InvokeSkillToolRequest,
) (*skillRuntime.InvokeSkillToolResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillRuntime.InvokeSkillToolResponse, error) {
			return w.service.InvokeSkillTool(
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
