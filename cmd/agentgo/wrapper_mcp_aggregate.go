package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	mcpAggregate "github.com/flexigpt/flexigpt-app/internal/mcp/aggregate"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpDomainSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/secret"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

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

func (w *MCPAggregateWrapper) RootIDForRuntimeCatalogID(
	id mcpServer.CatalogID,
) (root.RootID, error) {
	return withMCPAggregate(w, func(*mcpAggregate.Service) (root.RootID, error) {
		return mcpAggregate.RootIDForRuntimeCatalogID(id)
	})
}

func (w *MCPAggregateWrapper) UpdateMCPServerInstallation(
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpDomainServer.ServerData,
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
	data mcpDomainServer.ServerData,
) error {
	return withMCPAggregateError(w, func(service *mcpAggregate.Service) error {
		return service.UpdateProtectedServerInstallation(
			context.Background(),
			ref,
			expectedOverlayRevision,
			data,
		)
	})
}

func (w *MCPAggregateWrapper) PutMCPServerSecret(
	ref artifact.ArtifactRef,
	kind mcpDomainSecret.MCPSecretKind,
	slot string,
	value string,
) (mcpAggregate.SecretWriteResult, error) {
	return withMCPAggregate(w, func(service *mcpAggregate.Service) (mcpAggregate.SecretWriteResult, error) {
		return service.PutServerSecret(
			context.Background(),
			ref,
			kind,
			slot,
			value,
		)
	})
}

func (w *MCPAggregateWrapper) DeleteMCPServerSecret(
	ref artifact.ArtifactRef,
	kind mcpDomainSecret.MCPSecretKind,
	slot string,
) error {
	return withMCPAggregateError(w, func(service *mcpAggregate.Service) error {
		return service.DeleteServerSecret(
			context.Background(),
			ref,
			kind,
			slot,
		)
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
