package main

import (
	"context"

	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	workspaceConsumerAPI "github.com/flexigpt/flexigpt-app/internal/workspace/store/consumerapi"
)

type WorkspaceRuntimeWrapper struct {
	api *workspaceConsumerAPI.StoreAPI
}

func withWorkspaceRuntime[T any](
	w *WorkspaceRuntimeWrapper,
	fn func(*workspaceConsumerAPI.StoreAPI) (T, error),
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
	workspace artifact.ArtifactRef,
	artifacts []artifact.ArtifactRef,
) (workspaceConsumerAPI.WorkspacePromptPlan, error) {
	return withWorkspaceRuntime(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspacePromptPlan, error) {
			return api.ComposeWorkspacePrompt(
				context.Background(),
				workspace,
				artifacts,
			)
		},
	)
}

func (w *WorkspaceRuntimeWrapper) LoadWorkspaceSkills(
	workspace artifact.ArtifactRef,
	artifacts []artifact.ArtifactRef,
) (workspaceConsumerAPI.WorkspaceSkillLoadPlan, error) {
	return withWorkspaceRuntime(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceSkillLoadPlan, error) {
			return api.LoadWorkspaceSkills(
				context.Background(),
				workspace,
				artifacts,
			)
		},
	)
}

func (w *WorkspaceRuntimeWrapper) LoadWorkspaceMCPServers(
	workspace artifact.ArtifactRef,
	artifacts []artifact.ArtifactRef,
) (workspaceConsumerAPI.WorkspaceMCPServerLoadPlan, error) {
	return withWorkspaceRuntime(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceMCPServerLoadPlan, error) {
			return api.LoadWorkspaceMCPServers(
				context.Background(),
				workspace,
				artifacts,
			)
		},
	)
}

func (w *WorkspaceRuntimeWrapper) ResolveWorkspaceRuntimePlan(
	workspace artifact.ArtifactRef,
	selection workspaceConsumerAPI.WorkspaceRuntimeSelection,
) (workspaceConsumerAPI.WorkspaceRuntimePlan, error) {
	return withWorkspaceRuntime(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceRuntimePlan, error) {
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
