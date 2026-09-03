package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	mcpStore "github.com/flexigpt/flexigpt-app/internal/mcp/store"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

// MCPStoreWrapper exposes only pure Store operations. Runtime-affecting
// mutations intentionally live on MCPAggregateWrapper so they cannot bypass
// runtime invalidation.
type MCPStoreWrapper struct {
	api *mcpStore.API

	builtInInstaller artifactbuiltin.HydrationInstaller
}

func withMCPStore[T any](
	w *MCPStoreWrapper,
	fn func(*mcpStore.API) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if err := w.ready(); err != nil {
			return zero, err
		}
		return fn(w.api)
	})
}

func (w *MCPStoreWrapper) CreateMCPBundle(
	request *mcpStore.CreateRequest,
) (mcpStore.Bundle, error) {
	return withMCPStore(w, func(api *mcpStore.API) (mcpStore.Bundle, error) {
		if request == nil {
			return mcpStore.Bundle{}, errors.New("MCP Bundle create request is required")
		}
		return api.Create(context.Background(), *request)
	})
}

func (w *MCPStoreWrapper) GetMCPBundle(
	ref collection.CollectionRef,
) (mcpStore.Bundle, error) {
	return withMCPStore(w, func(api *mcpStore.API) (mcpStore.Bundle, error) {
		return api.Get(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) ListMCPBundles(
	rootID basespec.RootID,
) ([]mcpStore.Bundle, error) {
	return withMCPStore(w, func(api *mcpStore.API) ([]mcpStore.Bundle, error) {
		return api.List(context.Background(), rootID)
	})
}

func (w *MCPStoreWrapper) GetMCPBundleDocument(
	ref collection.CollectionRef,
) (mcpStore.BundleDocument, error) {
	return withMCPStore(w, func(api *mcpStore.API) (mcpStore.BundleDocument, error) {
		return api.GetDocument(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) ListMCPBundleServers(
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	return withMCPStore(w, func(api *mcpStore.API) ([]artifact.Artifact, error) {
		return api.ListServers(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) ListMCPBundlePolicies(
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	return withMCPStore(w, func(api *mcpStore.API) ([]artifact.Artifact, error) {
		return api.ListPolicies(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) GetMCPServerInstallation(
	ref artifact.ArtifactRef,
) (mcpStore.ServerInstallationView, error) {
	return withMCPStore(w, func(api *mcpStore.API) (mcpStore.ServerInstallationView, error) {
		return api.GetServerInstallation(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) InspectMCPServer(
	ref artifact.ArtifactRef,
) (mcpStoreServer.Resolved, error) {
	return withMCPStore(w, func(api *mcpStore.API) (mcpStoreServer.Resolved, error) {
		return api.InspectMCPServer(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) InspectMCPPolicy(
	ref artifact.ArtifactRef,
) (mcpStore.PolicyView, error) {
	return withMCPStore(w, func(api *mcpStore.API) (mcpStore.PolicyView, error) {
		return api.InspectMCPPolicy(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) GetMCPBundleInstallation(
	ref collection.CollectionRef,
) (mcpStore.BundleInstallationView, error) {
	return withMCPStore(w, func(api *mcpStore.API) (mcpStore.BundleInstallationView, error) {
		return api.GetBundleInstallation(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) ready() error {
	if w == nil || w.api == nil {
		return basespec.ErrClosed
	}
	return nil
}

func (w *MCPStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
	w.builtInInstaller = nil
}
