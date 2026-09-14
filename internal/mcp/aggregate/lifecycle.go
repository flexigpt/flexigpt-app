package aggregate

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

type RuntimeInvalidator interface {
	Invalidate(
		ctx context.Context,
		server mcpServer.ServerID,
	) error
}

type Lifecycle struct {
	store interface {
		UpdateServerInstallation(
			ctx context.Context,
			ref artifact.ArtifactRef,
			expectedArtifactRevision uint64,
			data mcpDomainServer.ServerData,
		) (artifact.Artifact, error)

		UpdateProtectedServerInstallation(
			ctx context.Context,
			ref artifact.ArtifactRef,
			expectedOverlayRevision uint64,
			runtimeEnabled bool,
			data mcpDomainServer.ServerData,
		) error
	}
	runtime RuntimeInvalidator
}

func NewLifecycle(
	store interface {
		UpdateServerInstallation(
			ctx context.Context,
			ref artifact.ArtifactRef,
			expectedArtifactRevision uint64,
			data mcpDomainServer.ServerData,
		) (artifact.Artifact, error)

		UpdateProtectedServerInstallation(
			ctx context.Context,
			ref artifact.ArtifactRef,
			expectedOverlayRevision uint64,
			runtimeEnabled bool,
			data mcpDomainServer.ServerData,
		) error
	},
	runtime RuntimeInvalidator,
) (*Lifecycle, error) {
	if store == nil || runtime == nil {
		return nil, errors.New("MCP lifecycle dependencies are incomplete")
	}
	return &Lifecycle{
		store:   store,
		runtime: runtime,
	}, nil
}

func (l *Lifecycle) InvalidateServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if l == nil || l.runtime == nil {
		return mcpServer.ErrClosed
	}
	serverID, err := RuntimeServerIDForArtifact(ref)
	if err != nil {
		return err
	}
	return l.runtime.Invalidate(ctx, serverID)
}

func (l *Lifecycle) UpdateServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpDomainServer.ServerData,
) (artifact.Artifact, error) {
	if err := l.InvalidateServer(ctx, ref); err != nil {
		return artifact.Artifact{}, err
	}
	return l.store.UpdateServerInstallation(
		ctx,
		ref,
		expectedArtifactRevision,
		data,
	)
}

func (l *Lifecycle) UpdateProtectedServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
	data mcpDomainServer.ServerData,
) error {
	if err := l.InvalidateServer(ctx, ref); err != nil {
		return err
	}
	return l.store.UpdateProtectedServerInstallation(
		ctx,
		ref,
		expectedOverlayRevision,
		runtimeEnabled,
		data,
	)
}
