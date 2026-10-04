package main

import (
	"context"
	"errors"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	toolBuiltin "github.com/flexigpt/flexigpt-app/internal/tool/store/builtin"
	toolConsumerAPI "github.com/flexigpt/flexigpt-app/internal/tool/store/consumerapi"
)

type ToolStoreWrapper struct {
	api *toolConsumerAPI.API
}

func InitToolStoreWrapper(
	wrapper *ToolStoreWrapper,
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	managedArtifacts managepackageFlow.API,
	protection root.ProtectionAPI,
	cat catalog.API,
	definitions definition.API,
) error {
	if wrapper == nil {
		return errors.New("tool store wrapper is required")
	}

	api, err := toolConsumerAPI.New(
		sources,
		discovery,
		artifacts,
		managedArtifacts,
		protection,
		cat,
		definitions,
		documentTopology.BuiltinRootID(),
	)
	if err != nil {
		return err
	}
	wrapper.api = api
	return nil
}

func NewToolBuiltInInstaller(
	hydrator installModel.CompiledHydrationCoordinator,
) (installFlow.HydrationInstaller, error) {
	if hydrator == nil {
		return nil, errors.New("tool generated catalog installer hydrator is required")
	}

	return toolBuiltin.NewInstaller(toolBuiltin.InstallerDependencies{
		Hydrator: hydrator,
	})
}

func withToolStore[T any](
	w *ToolStoreWrapper,
	fn func(*toolConsumerAPI.API) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, spec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *ToolStoreWrapper) ListToolCollections() (
	[]collection.ListItem,
	error,
) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) ([]collection.ListItem, error) {
			return api.ListToolCollections(context.Background())
		},
	)
}

func (w *ToolStoreWrapper) GetToolCollection(
	ref artifactModel.ArtifactRef,
) (collection.CollectionView, error) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) (collection.CollectionView, error) {
			return api.GetToolCollection(context.Background(), ref)
		},
	)
}

func (w *ToolStoreWrapper) ListCollectionTools(
	ref artifactModel.ArtifactRef,
) ([]toolConsumerAPI.ToolListItem, error) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) ([]toolConsumerAPI.ToolListItem, error) {
			return api.ListTools(context.Background(), ref)
		},
	)
}

func (w *ToolStoreWrapper) GetTool(
	ref artifactModel.ArtifactRef,
) (toolConsumerAPI.ToolView, error) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) (toolConsumerAPI.ToolView, error) {
			return api.GetTool(context.Background(), ref)
		},
	)
}

func (w *ToolStoreWrapper) SetToolEnabled(
	ref artifactModel.ArtifactRef,
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
	ref artifactModel.ArtifactRef,
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
