package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/mcp"
	workspaceConsumerAPI "github.com/flexigpt/flexigpt-app/internal/workspace/store/consumerapi"
)

type WorkspaceStoreWrapper struct {
	api                     *workspaceConsumerAPI.StoreAPI
	ensureArtifactBaselines func(context.Context, root.RootID) error
}

func InitWorkspaceWrappers(
	storeWrapper *WorkspaceStoreWrapper,
	runtimeWrapper *WorkspaceRuntimeWrapper,
	roots compositionapi.RootAPI,
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	locatorResolvers []providerapi.LocatorResolverFactory,
	fallbackProviders map[declaration.Type]resolve.FallbackProvider,
	targetMappers map[declaration.Type]resolve.ArtifactTargetMapper,
	mcpServers mcp.ServerResolver,
	ensureArtifactBaselines func(context.Context, root.RootID) error,
) error {
	if storeWrapper == nil ||
		runtimeWrapper == nil ||
		roots == nil ||
		mcpServers == nil ||
		ensureArtifactBaselines == nil {
		return errors.New("workspace wrapper receivers are incomplete")
	}

	config := workspaceConsumerAPI.DefaultConfig()
	config.LocatorResolvers = append(
		[]providerapi.LocatorResolverFactory(nil),
		locatorResolvers...,
	)
	config.FallbackProviders = fallbackProviders
	config.TargetMappers = targetMappers
	config.MCPServers = mcpServers
	api, err := workspaceConsumerAPI.NewStoreAPI(
		sources,
		discovery,
		artifacts,
		resources,
		roots,
		config,
	)
	if err != nil {
		return err
	}
	storeWrapper.api = api
	storeWrapper.ensureArtifactBaselines = ensureArtifactBaselines
	runtimeWrapper.api = api
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

func withWorkspaceStoreError(
	w *WorkspaceStoreWrapper,
	fn func(*workspaceConsumerAPI.StoreAPI) error,
) error {
	return middleware.WithRecovery(func() error {
		if w == nil || w.api == nil {
			return basespec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *WorkspaceStoreWrapper) RegisterWorkspaceDirectory(
	path string,
) (workspaceConsumerAPI.WorkspaceDirectoryView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceDirectoryView, error) {
			ctx := context.Background()
			view, err := api.RegisterWorkspaceDirectory(ctx, path)
			if err != nil {
				return workspaceConsumerAPI.WorkspaceDirectoryView{}, err
			}
			if w.ensureArtifactBaselines == nil {
				return workspaceConsumerAPI.WorkspaceDirectoryView{}, basespec.ErrClosed
			}
			if err := w.ensureArtifactBaselines(
				ctx,
				view.Ref.RootID,
			); err != nil {
				return workspaceConsumerAPI.WorkspaceDirectoryView{}, err
			}
			return view, nil
		},
	)
}

func (w *WorkspaceStoreWrapper) GetWorkspaceDirectory(
	ref workspaceConsumerAPI.WorkspaceDirectoryRef,
) (workspaceConsumerAPI.WorkspaceDirectoryView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceDirectoryView, error) {
			return api.GetWorkspaceDirectory(context.Background(), ref)
		},
	)
}

func (w *WorkspaceStoreWrapper) RefreshWorkspaceDirectory(
	ref workspaceConsumerAPI.WorkspaceDirectoryRef,
) (workspaceConsumerAPI.WorkspaceDirectoryView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceDirectoryView, error) {
			return api.RefreshWorkspaceDirectory(context.Background(), ref)
		},
	)
}

func (w *WorkspaceStoreWrapper) SetWorkspaceDirectoryEnabled(
	ref workspaceConsumerAPI.WorkspaceDirectoryRef,
	expectedRevision uint64,
	enabled bool,
) (workspaceConsumerAPI.WorkspaceDirectoryView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceDirectoryView, error) {
			return api.SetWorkspaceDirectoryEnabled(context.Background(), ref, expectedRevision, enabled)
		},
	)
}

func (w *WorkspaceStoreWrapper) RemoveWorkspaceDirectory(
	ref workspaceConsumerAPI.WorkspaceDirectoryRef,
	expectedRevision uint64,
) error {
	return withWorkspaceStoreError(w, func(api *workspaceConsumerAPI.StoreAPI) error {
		return api.RemoveWorkspaceDirectory(context.Background(), ref, expectedRevision)
	})
}

func (w *WorkspaceStoreWrapper) ListWorkspaceDirectories(
	request workspaceConsumerAPI.WorkspacePageRequest,
) (workspaceConsumerAPI.WorkspacePage, error) {
	return withWorkspaceStore(w, func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspacePage, error) {
		return api.ListWorkspaceDirectories(context.Background(), request)
	})
}

func (w *WorkspaceStoreWrapper) GetWorkspaceDefaultPolicy() (
	workspaceConsumerAPI.WorkspaceDefaultPolicyView,
	error,
) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (
			workspaceConsumerAPI.WorkspaceDefaultPolicyView,
			error,
		) {
			return api.WorkspaceDefaultPolicy(), nil
		},
	)
}

func (w *WorkspaceStoreWrapper) ListWorkspaceDirectoryArtifacts(
	ref workspaceConsumerAPI.WorkspaceDirectoryRef,
) ([]workspaceConsumerAPI.WorkspaceArtifactView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (
			[]workspaceConsumerAPI.WorkspaceArtifactView,
			error,
		) {
			return api.ListWorkspaceDirectoryArtifacts(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *WorkspaceStoreWrapper) SetWorkspaceDirectoryArtifactEnabled(
	directory workspaceConsumerAPI.WorkspaceDirectoryRef,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (workspaceConsumerAPI.WorkspaceArtifactView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (
			workspaceConsumerAPI.WorkspaceArtifactView,
			error,
		) {
			return api.SetWorkspaceDirectoryArtifactEnabled(
				context.Background(),
				directory,
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
	w.ensureArtifactBaselines = nil
}
