package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	toolBuiltin "github.com/flexigpt/flexigpt-app/internal/tool/store/builtin"
	toolConsumerAPI "github.com/flexigpt/flexigpt-app/internal/tool/store/consumerapi"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

type ToolStoreWrapper struct {
	api *toolConsumerAPI.API
}

func InitToolStoreWrapper(
	wrapper *ToolStoreWrapper,
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	goTools toolDomain.GoToolLocator,
) error {
	if wrapper == nil {
		return errors.New("tool store wrapper is required")
	}

	api, err := toolConsumerAPI.New(
		sources,
		discovery,
		artifacts,
		resources,
		managedArtifacts,
		protection,
		documentTopology.BuiltinRootID(),
		goTools,
	)
	if err != nil {
		return err
	}
	wrapper.api = api
	return nil
}

func NewToolBuiltInInstaller(
	storeWrapper *ToolStoreWrapper,
	goTools toolDomain.GoToolLocator,
) (builtin.HydrationInstaller, error) {
	if storeWrapper == nil || storeWrapper.api == nil {
		return nil, basespec.ErrClosed
	}
	if goTools == nil {
		return nil, errors.New("tool built-in installer Go Tool locator is required")
	}

	tools, err := toolConsumerAPI.NewBuiltinStore(storeWrapper.api)
	if err != nil {
		return nil, err
	}
	packages, err := builtin.EmbeddedToolPackages()
	if err != nil {
		return nil, err
	}

	return toolBuiltin.NewInstaller(toolBuiltin.InstallerDependencies{
		Tools:    tools,
		Packages: packages,
		GoTools:  goTools,
	})
}

func withToolStore[T any](
	w *ToolStoreWrapper,
	fn func(*toolConsumerAPI.API) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, basespec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *ToolStoreWrapper) ListToolCollections() (
	[]collection.CollectionView,
	error,
) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) ([]collection.CollectionView, error) {
			return api.ListToolCollections(context.Background())
		},
	)
}

func (w *ToolStoreWrapper) GetToolCollection(
	ref artifact.ArtifactRef,
) (collection.CollectionView, error) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) (collection.CollectionView, error) {
			return api.GetToolCollection(context.Background(), ref)
		},
	)
}

func (w *ToolStoreWrapper) ListCollectionTools(
	ref artifact.ArtifactRef,
) ([]toolConsumerAPI.ToolView, error) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) ([]toolConsumerAPI.ToolView, error) {
			return api.ListTools(context.Background(), ref)
		},
	)
}

func (w *ToolStoreWrapper) GetTool(
	ref artifact.ArtifactRef,
) (toolConsumerAPI.ToolView, error) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) (toolConsumerAPI.ToolView, error) {
			return api.GetTool(context.Background(), ref)
		},
	)
}

func (w *ToolStoreWrapper) SetToolEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (toolConsumerAPI.ToolView, error) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) (toolConsumerAPI.ToolView, error) {
			return api.SetToolEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *ToolStoreWrapper) SetToolCollectionEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (collection.CollectionView, error) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) (collection.CollectionView, error) {
			return api.SetToolCollectionEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *ToolStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
}
