package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
)

type SkillAggregateWrapper struct {
	service *skillAggregate.Service
}

func InitSkillAggregateWrapper(
	wrapper *SkillAggregateWrapper,
	storeWrapper *SkillStoreWrapper,
	runtimeWrapper *SkillRuntimeWrapper,
) error {
	if wrapper == nil ||
		storeWrapper == nil ||
		storeWrapper.router == nil ||
		runtimeWrapper == nil ||
		runtimeWrapper.service == nil {
		return errors.New("skill aggregate dependencies are incomplete")
	}

	service, err := skillAggregate.New(
		storeWrapper.router,
		runtimeWrapper.service,
	)
	if err != nil {
		return err
	}
	wrapper.service = service
	return nil
}

func (w *SkillAggregateWrapper) CreateSkillSession(
	request *skillAggregate.CreateSkillSessionRequest,
) (*skillAggregate.CreateSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillAggregate.CreateSkillSessionResponse, error) {
			return w.service.CreateSkillSession(context.Background(), request)
		},
	)
}

func (w *SkillAggregateWrapper) CloseSkillSession(
	request *skillAggregate.CloseSkillSessionRequest,
) (*skillAggregate.CloseSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillAggregate.CloseSkillSessionResponse, error) {
			return w.service.CloseSkillSession(context.Background(), request)
		},
	)
}

func (w *SkillAggregateWrapper) GetSkillsPrompt(
	request *skillAggregate.GetSkillsPromptRequest,
) (*skillAggregate.GetSkillsPromptResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillAggregate.GetSkillsPromptResponse, error) {
			return w.service.GetSkillsPrompt(context.Background(), request)
		},
	)
}

func (w *SkillAggregateWrapper) ListRuntimeSkills(
	request *skillAggregate.ListRuntimeSkillsRequest,
) (*skillAggregate.ListRuntimeSkillsResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillAggregate.ListRuntimeSkillsResponse, error) {
			return w.service.ListRuntimeSkills(context.Background(), request)
		},
	)
}

func (w *SkillAggregateWrapper) RenderSkill(
	request *skillAggregate.RenderSkillRequest,
) (*skillAggregate.RenderSkillResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillAggregate.RenderSkillResponse, error) {
			return w.service.RenderSkill(context.Background(), request)
		},
	)
}

func (w *SkillAggregateWrapper) InvokeSkillTool(
	request *skillAggregate.InvokeSkillToolRequest,
) (*skillAggregate.InvokeSkillToolResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*skillAggregate.InvokeSkillToolResponse, error) {
			return w.service.InvokeSkillTool(context.Background(), request)
		},
	)
}

// SyncSkillCollection is the Wails-facing store-to-runtime bridge.
// It transfers only Collection identity. Runtime pulls registrations through
// its configured CatalogSource.
func (w *SkillAggregateWrapper) SyncSkillCollection(
	ref collection.CollectionRef,
) error {
	return middleware.WithRecovery(func() error {
		return w.service.ResyncCollection(context.Background(), ref)
	})
}

func (w *SkillAggregateWrapper) RemoveSkillCollection(
	ref collection.CollectionRef,
) error {
	return middleware.WithRecovery(func() error {
		return w.service.RemoveCollection(context.Background(), ref)
	})
}

func (w *SkillAggregateWrapper) close() {
	if w == nil || w.service == nil {
		return
	}
	w.service.Close()
	w.service = nil
}
