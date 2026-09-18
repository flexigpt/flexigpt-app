package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
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
	api                     *workspaceConsumerAPI.StoreAPI
	roots                   compositionapi.RootAPI
	ensureArtifactBaselines func(context.Context, root.RootID) error
}

func InitWorkspaceWrappers(
	storeWrapper *WorkspaceStoreWrapper,
	runtimeWrapper *WorkspaceRuntimeWrapper,
	aggregateWrapper *WorkspaceAggregateWrapper,
	roots compositionapi.RootAPI,
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	locatorResolvers []providerapi.LocatorResolverFactory,
	fallbackProviders map[declaration.Type]resolve.FallbackProvider,
	mcpServers mcp.ServerResolver,
	ensureArtifactBaselines func(context.Context, root.RootID) error,
) error {
	if storeWrapper == nil ||
		runtimeWrapper == nil ||
		aggregateWrapper == nil ||
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
	storeWrapper.roots = roots
	storeWrapper.ensureArtifactBaselines = ensureArtifactBaselines
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

// CreateWorkspaceRoot exposes the existing generic Root creation flow for
// user-owned Workspace Sources. It does not create a Workspace Store entity.
func (w *WorkspaceStoreWrapper) CreateWorkspaceRoot(
	draft root.RootDraft,
) (root.Root, error) {
	return middleware.WithRecoveryResp(
		func() (root.Root, error) {
			if w == nil || w.roots == nil {
				return root.Root{}, basespec.ErrClosed
			}
			value, err := w.roots.Create(context.Background(), draft)
			if err != nil {
				return root.Root{}, err
			}
			if w.ensureArtifactBaselines == nil {
				return value, basespec.ErrClosed
			}
			if err := w.ensureArtifactBaselines(
				context.Background(),
				value.ID,
			); err != nil {
				return value, fmt.Errorf(
					"root was created but baseline Collection provisioning failed: %w",
					err,
				)
			}
			return value, nil
		},
	)
}

func (w *WorkspaceStoreWrapper) ListWorkspaceRoots() (
	[]root.Root,
	error,
) {
	return middleware.WithRecoveryResp(
		func() ([]root.Root, error) {
			if w == nil || w.roots == nil {
				return nil, basespec.ErrClosed
			}
			return w.roots.List(context.Background())
		},
	)
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
	ref artifact.ArtifactRef,
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
	ref artifact.ArtifactRef,
) (workspaceConsumerAPI.WorkspaceLoad, error) {
	return withWorkspaceStore(w, func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceLoad, error) {
		return api.LoadWorkspace(context.Background(), ref)
	})
}

func (w *WorkspaceStoreWrapper) ResolveWorkspaceCapabilities(
	ref artifact.ArtifactRef,
) (resolve.CapabilityPlan, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (resolve.CapabilityPlan, error) {
			return api.ResolveWorkspaceCapabilities(context.Background(), ref)
		},
	)
}

func (w *WorkspaceStoreWrapper) ResolveWorkspaceArtifactCapabilities(
	ref artifact.ArtifactRef,
) (resolve.CapabilityPlan, error) {
	return withWorkspaceStore(w, func(api *workspaceConsumerAPI.StoreAPI) (resolve.CapabilityPlan, error) {
		return api.ResolveArtifactCapabilities(context.Background(), ref)
	})
}

func (w *WorkspaceStoreWrapper) RefreshWorkspace(
	ref artifact.ArtifactRef,
) (workspaceConsumerAPI.WorkspaceRefresh, error) {
	return withWorkspaceStore(
		w,
		func(api *workspaceConsumerAPI.StoreAPI) (workspaceConsumerAPI.WorkspaceRefresh, error) {
			return api.RefreshWorkspace(context.Background(), ref)
		},
	)
}

func (w *WorkspaceStoreWrapper) ListWorkspaceArtifacts(
	ref artifact.ArtifactRef,
) ([]artifact.Artifact, error) {
	return withWorkspaceStore(w, func(api *workspaceConsumerAPI.StoreAPI) ([]artifact.Artifact, error) {
		return api.ListWorkspaceArtifacts(context.Background(), ref)
	})
}

func (w *WorkspaceStoreWrapper) SetWorkspaceArtifactEnabled(
	workspace artifact.ArtifactRef,
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
	w.roots = nil
	w.ensureArtifactBaselines = nil
}
