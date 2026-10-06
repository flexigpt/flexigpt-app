package mcp

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
)

// WorkspaceServerResolver is the only MCP capability that Workspace runtime
// planning needs. It intentionally exposes no installation, policy mutation,
// secret, overlay, or plugin APIs.
type WorkspaceServerResolver struct {
	api *Service
}

func NewWorkspaceServerResolver(
	api *Service,
) (*WorkspaceServerResolver, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: Workspace MCP resolver requires an API",
			spec.ErrInvalid,
		)
	}
	return &WorkspaceServerResolver{api: api}, nil
}

func (r *WorkspaceServerResolver) ResolveMCPServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (serverMCPDomain.Resolved, error) {
	if r == nil || r.api == nil {
		return serverMCPDomain.Resolved{}, spec.ErrClosed
	}
	read, err := r.api.ResolveMCPServer(ctx, ref)
	if err != nil {
		return serverMCPDomain.Resolved{}, err
	}
	return read.Resolved, nil
}
