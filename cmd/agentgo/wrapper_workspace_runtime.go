package main

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

type WorkspaceRuntimeWrapper struct {
	api *workspaceAPI.Service
}

func withWorkspaceRuntime[T any](
	w *WorkspaceRuntimeWrapper,
	fn func(*workspaceAPI.Service) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, spec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *WorkspaceRuntimeWrapper) ComposeWorkspacePrompt(
	workspace artifactModel.ArtifactRef,
	artifacts []artifactModel.ArtifactRef,
) (workspaceAPI.WorkspacePromptPlan, error) {
	return withWorkspaceRuntime(
		w,
		func(api *workspaceAPI.Service) (workspaceAPI.WorkspacePromptPlan, error) {
			return api.ComposeWorkspacePrompt(
				context.Background(),
				workspace,
				artifacts,
			)
		},
	)
}

func (w *WorkspaceRuntimeWrapper) LoadWorkspaceSkills(
	workspace artifactModel.ArtifactRef,
	artifacts []artifactModel.ArtifactRef,
) (workspaceAPI.WorkspaceSkillLoadPlan, error) {
	return withWorkspaceRuntime(
		w,
		func(api *workspaceAPI.Service) (workspaceAPI.WorkspaceSkillLoadPlan, error) {
			return api.LoadWorkspaceSkills(
				context.Background(),
				workspace,
				artifacts,
			)
		},
	)
}

func (w *WorkspaceRuntimeWrapper) LoadWorkspaceMCPServers(
	workspace artifactModel.ArtifactRef,
	artifacts []artifactModel.ArtifactRef,
) (workspaceAPI.WorkspaceMCPServerLoadPlan, error) {
	return withWorkspaceRuntime(
		w,
		func(api *workspaceAPI.Service) (workspaceAPI.WorkspaceMCPServerLoadPlan, error) {
			return api.LoadWorkspaceMCPServers(
				context.Background(),
				workspace,
				artifacts,
			)
		},
	)
}

func (w *WorkspaceRuntimeWrapper) ResolveWorkspaceRuntimePlan(
	workspace artifactModel.ArtifactRef,
	selection workspaceAPI.WorkspaceRuntimeSelection,
) (workspaceAPI.WorkspaceRuntimePlan, error) {
	return withWorkspaceRuntime(
		w,
		func(api *workspaceAPI.Service) (workspaceAPI.WorkspaceRuntimePlan, error) {
			return api.ResolveWorkspaceRuntimePlan(
				context.Background(),
				workspace,
				selection,
			)
		},
	)
}

func (w *WorkspaceRuntimeWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
}
