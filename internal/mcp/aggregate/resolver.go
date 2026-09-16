package aggregate

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

// ArtifactServerResolver translates a runtime-owned opaque ServerID only at
// the Aggregate boundary, then delegates Store resolution to the narrow port.
type ArtifactServerResolver struct {
	store mcpConsumerAPI.ServerStore
}

func NewArtifactServerResolver(
	store mcpConsumerAPI.ServerStore,
) (*ArtifactServerResolver, error) {
	if store == nil {
		return nil, errors.New("MCP server Store is required")
	}
	return &ArtifactServerResolver{store: store}, nil
}

func (r *ArtifactServerResolver) ResolveMCPServer(
	ctx context.Context,
	serverID mcpServer.ServerID,
) (mcpDomainServer.Resolved, error) {
	if r == nil || r.store == nil {
		return mcpDomainServer.Resolved{}, mcpServer.ErrClosed
	}

	ref, err := ArtifactRefForRuntimeServerID(serverID)
	if err != nil {
		return mcpDomainServer.Resolved{}, err
	}
	return r.store.ResolveMCPServer(ctx, ref)
}

func (r *ArtifactServerResolver) InspectMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpDomainServer.Resolved, error) {
	if r == nil || r.store == nil {
		return mcpDomainServer.Resolved{}, mcpServer.ErrClosed
	}
	return r.store.ResolveMCPServer(ctx, ref)
}
