package aggregate

import (
	"context"
	"errors"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	mcpAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

// ArtifactServerResolver translates a runtime-owned opaque ServerID only at
// the Aggregate boundary, then delegates Store resolution to the narrow port.
type ArtifactServerResolver struct {
	store mcpAPI.ServerStore
}

func NewArtifactServerResolver(
	store mcpAPI.ServerStore,
) (*ArtifactServerResolver, error) {
	if store == nil {
		return nil, errors.New("MCP server Store is required")
	}
	return &ArtifactServerResolver{store: store}, nil
}

func (r *ArtifactServerResolver) ResolveMCPServer(
	ctx context.Context,
	serverID mcpServer.ServerID,
) (serverMCPDomain.Resolved, error) {
	if r == nil || r.store == nil {
		return serverMCPDomain.Resolved{}, mcpServer.ErrClosed
	}

	ref, err := artifactRefForRuntimeServerID(serverID)
	if err != nil {
		return serverMCPDomain.Resolved{}, err
	}
	read, err := r.store.ResolveMCPServer(ctx, ref)
	if err != nil {
		return serverMCPDomain.Resolved{}, err
	}
	return read.Resolved, nil
}

func (r *ArtifactServerResolver) InspectMCPServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (serverMCPDomain.Resolved, error) {
	if r == nil || r.store == nil {
		return serverMCPDomain.Resolved{}, mcpServer.ErrClosed
	}
	read, err := r.store.ResolveMCPServer(ctx, ref)
	if err != nil {
		return serverMCPDomain.Resolved{}, err
	}
	return read.Resolved, nil
}
