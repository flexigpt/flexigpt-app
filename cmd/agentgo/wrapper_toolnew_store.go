package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	toolnewBuiltin "github.com/flexigpt/flexigpt-app/internal/toolnew/store/builtin"
	toolnewConsumerAPI "github.com/flexigpt/flexigpt-app/internal/toolnew/store/consumerapi"
	toolnewDomain "github.com/flexigpt/flexigpt-app/internal/toolnew/store/domain"
)

type ToolNewStoreWrapper struct {
	api *toolnewConsumerAPI.API
}

func InitToolNewStoreWrapper(
	wrapper *ToolNewStoreWrapper,
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	goTools toolnewDomain.GoToolLocator,
) error {
	if wrapper == nil ||
		sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		managedArtifacts == nil ||
		protection == nil ||
		goTools == nil {
		return errors.New("tool store wrapper dependencies are incomplete")
	}

	api, err := toolnewConsumerAPI.New(
		sources,
		discovery,
		artifacts,
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

func NewToolNewBuiltInInstaller(
	storeWrapper *ToolNewStoreWrapper,
	goTools toolnewDomain.GoToolLocator,
) (builtin.HydrationInstaller, error) {
	if storeWrapper == nil || storeWrapper.api == nil {
		return nil, basespec.ErrClosed
	}
	if goTools == nil {
		return nil, errors.New("tool built-in installer Go Tool locator is required")
	}

	tools, err := toolnewConsumerAPI.NewBuiltinStore(storeWrapper.api)
	if err != nil {
		return nil, err
	}

	packages, err := builtin.EmbeddedToolPackages()
	if err != nil {
		return nil, err
	}

	return toolnewBuiltin.NewInstaller(
		toolnewBuiltin.InstallerDependencies{
			Tools:    tools,
			Packages: packages,
			GoTools:  goTools,
		},
	)
}

func withToolNewStore[T any](
	w *ToolNewStoreWrapper,
	fn func(*toolnewConsumerAPI.API) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, basespec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *ToolNewStoreWrapper) ListToolCollections() (
	[]toolnewDomain.ToolCollection,
	error,
) {
	return withToolNewStore(
		w,
		func(api *toolnewConsumerAPI.API) (
			[]toolnewDomain.ToolCollection,
			error,
		) {
			return api.ListToolCollections(context.Background())
		},
	)
}

func (w *ToolNewStoreWrapper) GetToolCollection(
	ref artifact.ArtifactRef,
) (toolnewDomain.ToolCollection, error) {
	return withToolNewStore(
		w,
		func(api *toolnewConsumerAPI.API) (
			toolnewDomain.ToolCollection,
			error,
		) {
			return api.GetToolCollection(context.Background(), ref)
		},
	)
}

func (w *ToolNewStoreWrapper) ListCollectionTools(
	ref artifact.ArtifactRef,
) ([]toolnewDomain.Tool, error) {
	return withToolNewStore(
		w,
		func(api *toolnewConsumerAPI.API) ([]toolnewDomain.Tool, error) {
			return api.ListTools(context.Background(), ref)
		},
	)
}

func (w *ToolNewStoreWrapper) SetToolEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (toolnewDomain.Tool, error) {
	return withToolNewStore(
		w,
		func(api *toolnewConsumerAPI.API) (toolnewDomain.Tool, error) {
			return api.SetToolEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *ToolNewStoreWrapper) SetToolCollectionEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (toolnewDomain.ToolCollection, error) {
	return withToolNewStore(
		w,
		func(api *toolnewConsumerAPI.API) (
			toolnewDomain.ToolCollection,
			error,
		) {
			return api.SetToolCollectionEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *ToolNewStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
}
