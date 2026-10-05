package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/mcp"
	workspaceConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/consumerapi"
)

type WorkspaceStoreWrapper struct {
	api                     *workspaceConsumerAPI.StoreAPI
	ensureArtifactBaselines func(context.Context, rootModel.RootID) error
}

func InitWorkspaceWrappers(
	storeWrapper *WorkspaceStoreWrapper,
	runtimeWrapper *WorkspaceRuntimeWrapper,
	roots root.API,
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	cat catalog.API,
	resources resourceFlow.API,
	nativeResources resourceFlow.NativePathAPI,
	resolver *composition.Resolver,
	workspaceConfig workspaceConsumerAPI.Config,
	mcpServers mcp.ServerResolver,
	ensureArtifactBaselines func(context.Context, rootModel.RootID) error,
) error {
	if storeWrapper == nil ||
		runtimeWrapper == nil ||
		roots == nil ||
		mcpServers == nil ||
		ensureArtifactBaselines == nil || nativeResources == nil {
		return errors.New("workspace wrapper receivers are incomplete")
	}

	workspaceConfig.Composition = resolver
	workspaceConfig.MCPServers = mcpServers
	api, err := workspaceConsumerAPI.NewStoreAPI(
		sources,
		discovery,
		artifacts,
		resources,
		nativeResources,
		roots,
		cat,
		workspaceConfig,
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
	return withRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, spec.ErrClosed
		}
		return fn(w.api)
	})
}

func withWorkspaceStoreError(
	w *WorkspaceStoreWrapper,
	fn func(*workspaceConsumerAPI.StoreAPI) error,
) error {
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return spec.ErrClosed
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
				return workspaceConsumerAPI.WorkspaceDirectoryView{}, spec.ErrClosed
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
	ref artifactModel.ArtifactRef,
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
