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
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
	workspacemcp "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/mcp"
)

type workspaceDefaultPreferenceStore interface {
	GetDefaultWorkspace(
		ctx context.Context,
	) (*artifactModel.ArtifactRef, error)

	SetDefaultWorkspace(
		ctx context.Context,
		ref *artifactModel.ArtifactRef,
	) error
}

type WorkspaceStoreWrapper struct {
	api                     *workspaceAPI.Service
	preferences             workspaceDefaultPreferenceStore
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
	workspaceConfig workspaceAPI.Config,
	mcpServers workspacemcp.ServerResolver,
	preferences workspaceDefaultPreferenceStore,
	ensureArtifactBaselines func(context.Context, rootModel.RootID) error,
) error {
	if storeWrapper == nil ||
		runtimeWrapper == nil ||
		roots == nil ||
		mcpServers == nil ||
		preferences == nil ||
		ensureArtifactBaselines == nil ||
		nativeResources == nil {
		return errors.New("workspace wrapper receivers are incomplete")
	}

	workspaceConfig.Composition = resolver
	workspaceConfig.MCPServers = mcpServers
	api, err := workspaceAPI.New(
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
	storeWrapper.preferences = preferences
	storeWrapper.ensureArtifactBaselines = ensureArtifactBaselines
	runtimeWrapper.api = api
	return nil
}

func withWorkspaceStore[T any](
	w *WorkspaceStoreWrapper,
	fn func(*workspaceAPI.Service) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, storeSpec.ErrClosed
		}
		return fn(w.api)
	})
}

func withWorkspaceStoreError(
	w *WorkspaceStoreWrapper,
	fn func(*workspaceAPI.Service) error,
) error {
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return storeSpec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *WorkspaceStoreWrapper) RegisterWorkspaceDirectory(
	path string,
) (workspaceAPI.WorkspaceDirectoryView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceAPI.Service) (workspaceAPI.WorkspaceDirectoryView, error) {
			ctx := context.Background()
			view, err := api.RegisterWorkspaceDirectory(ctx, path)
			if err != nil {
				return workspaceAPI.WorkspaceDirectoryView{}, err
			}
			if w.ensureArtifactBaselines == nil {
				return workspaceAPI.WorkspaceDirectoryView{}, storeSpec.ErrClosed
			}
			if err := w.ensureArtifactBaselines(
				ctx,
				view.Ref.RootID,
			); err != nil {
				return workspaceAPI.WorkspaceDirectoryView{}, err
			}
			return view, nil
		},
	)
}

func (w *WorkspaceStoreWrapper) GetWorkspaceDirectory(
	ref workspaceAPI.WorkspaceDirectoryRef,
) (workspaceAPI.WorkspaceDirectoryView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceAPI.Service) (workspaceAPI.WorkspaceDirectoryView, error) {
			return api.GetWorkspaceDirectory(context.Background(), ref)
		},
	)
}

// GetDefaultWorkspace returns nil when no default Workspace is configured.
// It does not choose a fallback Workspace.
func (w *WorkspaceStoreWrapper) GetDefaultWorkspace() (
	*artifactModel.ArtifactRef,
	error,
) {
	return withWorkspaceStore(
		w,
		func(_ *workspaceAPI.Service) (
			*artifactModel.ArtifactRef,
			error,
		) {
			if w.preferences == nil {
				return nil, storeSpec.ErrClosed
			}
			return w.preferences.GetDefaultWorkspace(
				context.Background(),
			)
		},
	)
}

func (w *WorkspaceStoreWrapper) SetDefaultWorkspace(
	ref artifactModel.ArtifactRef,
) error {
	_, err := withWorkspaceStore(
		w,
		func(api *workspaceAPI.Service) (struct{}, error) {
			if w.preferences == nil {
				return struct{}{}, storeSpec.ErrClosed
			}

			ctx := context.Background()
			if err := api.RequireSettableDefaultWorkspace(ctx, ref); err != nil {
				return struct{}{}, err
			}
			return struct{}{}, w.preferences.SetDefaultWorkspace(
				ctx,
				&ref,
			)
		},
	)
	return err
}

// ClearDefaultWorkspace creates the explicit no-default state. It does not
// alter the selected Workspace Artifact or its directory Sources.
func (w *WorkspaceStoreWrapper) ClearDefaultWorkspace() error {
	_, err := withWorkspaceStore(
		w,
		func(_ *workspaceAPI.Service) (struct{}, error) {
			if w.preferences == nil {
				return struct{}{}, storeSpec.ErrClosed
			}
			return struct{}{}, w.preferences.SetDefaultWorkspace(
				context.Background(),
				nil,
			)
		},
	)
	return err
}

func (w *WorkspaceStoreWrapper) RefreshWorkspaceDirectory(
	ref workspaceAPI.WorkspaceDirectoryRef,
) (workspaceAPI.WorkspaceDirectoryView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceAPI.Service) (workspaceAPI.WorkspaceDirectoryView, error) {
			return api.RefreshWorkspaceDirectory(context.Background(), ref)
		},
	)
}

func (w *WorkspaceStoreWrapper) SetWorkspaceDirectoryEnabled(
	ref workspaceAPI.WorkspaceDirectoryRef,
	expectedRevision uint64,
	enabled bool,
) (workspaceAPI.WorkspaceDirectoryView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceAPI.Service) (workspaceAPI.WorkspaceDirectoryView, error) {
			return api.SetWorkspaceDirectoryEnabled(context.Background(), ref, expectedRevision, enabled)
		},
	)
}

func (w *WorkspaceStoreWrapper) RemoveWorkspaceDirectory(
	ref workspaceAPI.WorkspaceDirectoryRef,
	expectedRevision uint64,
) error {
	return withWorkspaceStoreError(w, func(api *workspaceAPI.Service) error {
		return api.RemoveWorkspaceDirectory(context.Background(), ref, expectedRevision)
	})
}

func (w *WorkspaceStoreWrapper) ListWorkspaceDirectories(
	request workspaceAPI.WorkspacePageRequest,
) (workspaceAPI.WorkspacePage, error) {
	return withWorkspaceStore(w, func(api *workspaceAPI.Service) (workspaceAPI.WorkspacePage, error) {
		return api.ListWorkspaceDirectories(context.Background(), request)
	})
}

func (w *WorkspaceStoreWrapper) GetWorkspaceDefaultPolicy() (
	workspaceAPI.WorkspaceDefaultPolicyView,
	error,
) {
	return withWorkspaceStore(
		w,
		func(api *workspaceAPI.Service) (
			workspaceAPI.WorkspaceDefaultPolicyView,
			error,
		) {
			return api.WorkspaceDefaultPolicy(), nil
		},
	)
}

// ExportWorkspaceSource returns the verified portable YAML or JSON source for
// one Workspace manifest owned by directory.
func (w *WorkspaceStoreWrapper) ExportWorkspaceSource(
	directory workspaceAPI.WorkspaceDirectoryRef,
	workspace artifactModel.ArtifactRef,
) (workspaceAPI.WorkspaceSourceExport, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceAPI.Service) (
			workspaceAPI.WorkspaceSourceExport,
			error,
		) {
			return api.ExportWorkspaceSource(
				context.Background(),
				directory,
				workspace,
			)
		},
	)
}

func (w *WorkspaceStoreWrapper) ListWorkspaceDirectoryArtifacts(
	ref workspaceAPI.WorkspaceDirectoryRef,
) ([]workspaceAPI.WorkspaceArtifactView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceAPI.Service) (
			[]workspaceAPI.WorkspaceArtifactView,
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
	directory workspaceAPI.WorkspaceDirectoryRef,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (workspaceAPI.WorkspaceArtifactView, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceAPI.Service) (
			workspaceAPI.WorkspaceArtifactView,
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
	w.preferences = nil
	w.ensureArtifactBaselines = nil
}
