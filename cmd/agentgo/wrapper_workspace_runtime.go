package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/prompt"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/skill"
	workspaceConsumerAPI "github.com/flexigpt/flexigpt-app/internal/workspace/store/consumerapi"
)

type WorkspaceRuntimeWrapper struct {
	api *workspaceConsumerAPI.StoreAPI
}

func withWorkspaceRuntime[T any](
	w *WorkspaceRuntimeWrapper,
	fn func(*workspaceConsumerAPI.StoreAPI) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, basespec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *WorkspaceRuntimeWrapper) ComposeWorkspacePrompt(
	workspace workspaceConsumerAPI.WorkspaceRef,
	artifacts []artifact.ArtifactRef,
) (prompt.Plan, error) {
	return withWorkspaceRuntime(w, func(api *workspaceConsumerAPI.StoreAPI) (prompt.Plan, error) {
		return api.ComposeWorkspacePrompt(
			context.Background(),
			workspace,
			artifacts,
		)
	})
}

func (w *WorkspaceRuntimeWrapper) ListWorkspaceSkills(
	workspace workspaceConsumerAPI.WorkspaceRef,
) ([]skill.WorkspaceSkill, error) {
	return withWorkspaceRuntime(w, func(api *workspaceConsumerAPI.StoreAPI) ([]skill.WorkspaceSkill, error) {
		return api.ListWorkspaceSkills(context.Background(), workspace)
	})
}

func (w *WorkspaceRuntimeWrapper) LoadWorkspaceSkills(
	workspace workspaceConsumerAPI.WorkspaceRef,
	artifacts []artifact.ArtifactRef,
) (skill.LoadPlan, error) {
	return withWorkspaceRuntime(w, func(api *workspaceConsumerAPI.StoreAPI) (skill.LoadPlan, error) {
		return api.LoadWorkspaceSkills(
			context.Background(),
			workspace,
			artifacts,
		)
	})
}

func (w *WorkspaceRuntimeWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
}
