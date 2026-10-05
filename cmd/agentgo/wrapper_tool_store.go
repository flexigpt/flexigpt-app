package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	toolConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/consumerapi"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/domain"
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
	builtin toolDomain.BuiltinCatalog,
	resolver *composition.Resolver,
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
		builtin,
		resolver,
	)
	if err != nil {
		return err
	}
	wrapper.api = api
	return nil
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

func (w *ToolStoreWrapper) ListToolPlugins() (
	[]plugin.ListItem,
	error,
) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) ([]plugin.ListItem, error) {
			return api.ListToolPlugins(context.Background())
		},
	)
}

func (w *ToolStoreWrapper) GetToolPlugin(
	ref artifactModel.ArtifactRef,
) (plugin.PluginView, error) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) (plugin.PluginView, error) {
			return api.GetToolPlugin(context.Background(), ref)
		},
	)
}

func (w *ToolStoreWrapper) ListPluginTools(
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

func (w *ToolStoreWrapper) SetToolPluginEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (plugin.PluginView, error) {
	return withToolStore(
		w,
		func(api *toolConsumerAPI.API) (plugin.PluginView, error) {
			return api.SetToolPluginEnabled(
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
