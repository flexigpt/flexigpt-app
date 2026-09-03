package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	mcpAggregate "github.com/flexigpt/flexigpt-app/internal/mcp/aggregate"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpStore "github.com/flexigpt/flexigpt-app/internal/mcp/store"
	mcpSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/secret"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

// MCPAggregateWrapper is the only Wails surface allowed to translate durable
// artifact identities into opaque runtime identities or coordinate Store
// mutation with Runtime invalidation.
type MCPAggregateWrapper struct {
	service        *mcpAggregate.Service
	serverResolver *mcpAggregate.ArtifactServerResolver
}

func withMCPAggregate[T any](
	w *MCPAggregateWrapper,
	fn func(*mcpAggregate.Service) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if err := w.ready(); err != nil {
			return zero, err
		}
		return fn(w.service)
	})
}

func withMCPAggregateError(
	w *MCPAggregateWrapper,
	fn func(*mcpAggregate.Service) error,
) error {
	return middleware.WithRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		return fn(w.service)
	})
}

func (w *MCPAggregateWrapper) RuntimeServerIDForArtifact(
	ref artifact.ArtifactRef,
) (mcpServer.ServerID, error) {
	return withMCPAggregate(w, func(*mcpAggregate.Service) (mcpServer.ServerID, error) {
		return mcpAggregate.RuntimeServerIDForArtifact(ref)
	})
}

func (w *MCPAggregateWrapper) ArtifactRefForRuntimeServerID(
	id mcpServer.ServerID,
) (artifact.ArtifactRef, error) {
	return withMCPAggregate(w, func(*mcpAggregate.Service) (artifact.ArtifactRef, error) {
		return mcpAggregate.ArtifactRefForRuntimeServerID(id)
	})
}

func (w *MCPAggregateWrapper) RuntimeCatalogIDForCollection(
	ref collection.CollectionRef,
) (mcpServer.CatalogID, error) {
	return withMCPAggregate(w, func(*mcpAggregate.Service) (mcpServer.CatalogID, error) {
		return mcpAggregate.RuntimeCatalogIDForCollection(ref)
	})
}

func (w *MCPAggregateWrapper) CollectionRefForRuntimeCatalogID(
	id mcpServer.CatalogID,
) (collection.CollectionRef, error) {
	return withMCPAggregate(w, func(*mcpAggregate.Service) (collection.CollectionRef, error) {
		return mcpAggregate.CollectionRefForRuntimeCatalogID(id)
	})
}

func (w *MCPAggregateWrapper) ReplaceMCPBundleDocument(
	request *mcpStore.ReplaceDocumentRequest,
) (mcpStore.Bundle, error) {
	return withMCPAggregate(w, func(service *mcpAggregate.Service) (mcpStore.Bundle, error) {
		if request == nil {
			return mcpStore.Bundle{}, basespec.ErrInvalid
		}
		return service.ReplaceDocument(context.Background(), *request)
	})
}

func (w *MCPAggregateWrapper) RefreshMCPBundle(
	ref collection.CollectionRef,
) (mcpStore.Bundle, error) {
	return withMCPAggregate(w, func(service *mcpAggregate.Service) (mcpStore.Bundle, error) {
		return service.RefreshBundle(context.Background(), ref, false)
	})
}

func (w *MCPAggregateWrapper) UpdateMCPBundleEnabled(
	ref collection.CollectionRef,
	expectedRevision uint64,
	enabled bool,
) (mcpStore.Bundle, error) {
	return withMCPAggregate(w, func(service *mcpAggregate.Service) (mcpStore.Bundle, error) {
		return service.UpdateBundleEnabled(
			context.Background(),
			ref,
			expectedRevision,
			enabled,
		)
	})
}

func (w *MCPAggregateWrapper) RetireMCPBundle(
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	return withMCPAggregate(w, func(service *mcpAggregate.Service) (collection.Collection, error) {
		return service.RetireBundle(context.Background(), ref, expectedRevision)
	})
}

func (w *MCPAggregateWrapper) PurgeMCPBundle(
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	return withMCPAggregateError(w, func(service *mcpAggregate.Service) error {
		return service.PurgeBundle(context.Background(), ref, expectedRevision)
	})
}

func (w *MCPAggregateWrapper) UpdateMCPServerInstallation(
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpStoreServer.ServerData,
) (artifact.Artifact, error) {
	return withMCPAggregate(w, func(service *mcpAggregate.Service) (artifact.Artifact, error) {
		return service.UpdateServerInstallation(
			context.Background(),
			ref,
			expectedArtifactRevision,
			data,
		)
	})
}

func (w *MCPAggregateWrapper) UpdateProtectedMCPServerInstallation(
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
	data mcpStoreServer.ServerData,
) error {
	return withMCPAggregateError(w, func(service *mcpAggregate.Service) error {
		return service.UpdateProtectedServerInstallation(
			context.Background(),
			ref,
			expectedOverlayRevision,
			runtimeEnabled,
			data,
		)
	})
}

func (w *MCPAggregateWrapper) UpdateProtectedMCPBundleInstallation(
	ref collection.CollectionRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
) error {
	return withMCPAggregateError(w, func(service *mcpAggregate.Service) error {
		return service.UpdateProtectedBundleInstallation(
			context.Background(),
			ref,
			expectedOverlayRevision,
			runtimeEnabled,
		)
	})
}

func (w *MCPAggregateWrapper) PutMCPServerSecret(
	ref artifact.ArtifactRef,
	kind mcpSecret.MCPSecretKind,
	slot string,
	value string,
) (mcpAggregate.SecretWriteResult, error) {
	return withMCPAggregate(w, func(service *mcpAggregate.Service) (mcpAggregate.SecretWriteResult, error) {
		return service.PutServerSecret(context.Background(), ref, kind, slot, value)
	})
}

func (w *MCPAggregateWrapper) DeleteMCPServerSecret(
	ref artifact.ArtifactRef,
	kind mcpSecret.MCPSecretKind,
	slot string,
) error {
	return withMCPAggregateError(w, func(service *mcpAggregate.Service) error {
		return service.DeleteServerSecret(context.Background(), ref, kind, slot)
	})
}

func (w *MCPAggregateWrapper) GetMCPServerAuthHealth(
	ref artifact.ArtifactRef,
) (mcpAuth.MCPAuthHealth, error) {
	return withMCPAggregate(w, func(service *mcpAggregate.Service) (mcpAuth.MCPAuthHealth, error) {
		return service.GetServerAuthHealth(context.Background(), ref)
	})
}

func (w *MCPAggregateWrapper) ready() error {
	if w == nil || w.service == nil || w.serverResolver == nil {
		return basespec.ErrClosed
	}
	return nil
}

func (w *MCPAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.service = nil
	w.serverResolver = nil
}
