package aggregate

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
)

// ServerStore is the narrow Store read port needed by aggregate adapters.
// Store.API satisfies this interface without importing Aggregate.
type ServerStore interface {
	ResolveMCPServer(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (mcpStoreServer.Resolved, error)

	InspectMCPServer(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (mcpStoreServer.Resolved, error)
}

// ArtifactServerResolver translates a runtime-owned opaque ServerID only at
// the Aggregate boundary, then delegates Store resolution to the narrow port.
type ArtifactServerResolver struct {
	store ServerStore
}

func NewArtifactServerResolver(
	store ServerStore,
) (*ArtifactServerResolver, error) {
	if store == nil {
		return nil, errors.New("MCP server Store is required")
	}
	return &ArtifactServerResolver{store: store}, nil
}

func (r *ArtifactServerResolver) ResolveMCPServer(
	ctx context.Context,
	serverID mcpServer.ServerID,
) (mcpStoreServer.Resolved, error) {
	if r == nil || r.store == nil {
		return mcpStoreServer.Resolved{}, mcpServer.ErrClosed
	}

	ref, err := ArtifactRefForRuntimeServerID(serverID)
	if err != nil {
		return mcpStoreServer.Resolved{}, err
	}
	return r.store.ResolveMCPServer(ctx, ref)
}

func (r *ArtifactServerResolver) InspectMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpStoreServer.Resolved, error) {
	if r == nil || r.store == nil {
		return mcpStoreServer.Resolved{}, mcpServer.ErrClosed
	}
	return r.store.InspectMCPServer(ctx, ref)
}
