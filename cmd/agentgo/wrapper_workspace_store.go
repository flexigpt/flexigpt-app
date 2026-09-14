package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/mcp"
	workspaceConsumerAPI "github.com/flexigpt/flexigpt-app/internal/workspace/store/consumerapi"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type WorkspaceStoreWrapper struct {
	api *workspaceConsumerAPI.StoreAPI
}

func InitWorkspaceWrappers(
	storeWrapper *WorkspaceStoreWrapper,
	runtimeWrapper *WorkspaceRuntimeWrapper,
	aggregateWrapper *WorkspaceAggregateWrapper,
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	locatorResolvers []providerapi.LocatorResolverFactory,
	mcpServers mcp.ServerResolver,
) error {
	if storeWrapper == nil ||
		runtimeWrapper == nil ||
		aggregateWrapper == nil ||
		mcpServers == nil {
		return errors.New("workspace wrapper receivers are incomplete")
	}

	config := workspaceConsumerAPI.DefaultConfig()
	config.LocatorResolvers = append(
		[]providerapi.LocatorResolverFactory(nil),
		locatorResolvers...,
	)
	config.MCPServers = mcpServers
	api, err := workspaceConsumerAPI.NewStoreAPI(
		sources,
		discovery,
		artifacts,
		resources,
		config,
	)
	if err != nil {
		return err
	}
	storeWrapper.api = api
	runtimeWrapper.api = api
	aggregateWrapper.api = api
	return nil
}

func withWorkspaceStore[T any](
	w *WorkspaceStoreWrapper,
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

func (w *WorkspaceStoreWrapper) RegisterFilesystemWorkspaceSource(
	request workspaceConsumerAPI.FilesystemSourceRegistration,
) (source.Summary, error) {
	return withWorkspaceStore(w, func(api *workspaceConsumerAPI.StoreAPI) (source.Summary, error) {
		return api.RegisterFilesystemSource(context.Background(), request)
	})
}

func (w *WorkspaceStoreWrapper) AddWorkspacePath(
	request workspaceConsumerAPI.WorkspacePathRegistration,
) (workspaceConsumerAPI.WorkspacePathRegistrationResult, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspacePathRegistrationResult, error) {
			return api.AddWorkspacePath(context.Background(), request)
		},
	)
}

func (w *WorkspaceStoreWrapper) GetWorkspace(
	ref workspaceConsumerAPI.WorkspaceRef,
) (workspaceDomain.Workspace, error) {
	return withWorkspaceStore(w, func(api *workspaceConsumerAPI.StoreAPI) (workspaceDomain.Workspace, error) {
		return api.GetWorkspace(context.Background(), ref)
	})
}

func (w *WorkspaceStoreWrapper) ListWorkspaces(
	rootID root.RootID,
) ([]workspaceDomain.Workspace, error) {
	return withWorkspaceStore(w, func(api *workspaceConsumerAPI.StoreAPI) ([]workspaceDomain.Workspace, error) {
		return api.ListWorkspaces(context.Background(), rootID)
	})
}

func (w *WorkspaceStoreWrapper) LoadWorkspace(
	ref workspaceConsumerAPI.WorkspaceRef,
) (workspaceConsumerAPI.WorkspaceLoad, error) {
	return withWorkspaceStore(w, func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceLoad, error) {
		return api.LoadWorkspace(context.Background(), ref)
	})
}

func (w *WorkspaceStoreWrapper) RefreshWorkspace(
	ref workspaceConsumerAPI.WorkspaceRef,
) (workspaceConsumerAPI.WorkspaceRefresh, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceRefresh, error) {
			return api.RefreshWorkspace(context.Background(), ref)
		},
	)
}

func (w *WorkspaceStoreWrapper) ListWorkspaceArtifacts(
	ref workspaceConsumerAPI.WorkspaceRef,
) ([]artifact.Artifact, error) {
	return withWorkspaceStore(w, func(api *workspaceConsumerAPI.StoreAPI) ([]artifact.Artifact, error) {
		return api.ListWorkspaceArtifacts(context.Background(), ref)
	})
}

func (w *WorkspaceStoreWrapper) SetWorkspaceArtifactEnabled(
	workspace workspaceConsumerAPI.WorkspaceRef,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (workspaceConsumerAPI.WorkspaceArtifactView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceArtifactView, error) {
			return api.SetWorkspaceArtifactEnabled(
				context.Background(),
				workspace,
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *WorkspaceStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
}
