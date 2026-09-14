package mcp

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

// ServerResolver is satisfied by mcp/store/consumerapi.API. Workspace only
// needs resolved server material and does not own MCP installation or runtime
// connection behavior.
type ServerResolver interface {
	ResolveMCPServer(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (mcpDomainServer.Resolved, error)
}

type WorkspaceServer struct {
	Artifact artifact.ArtifactRef     `json:"artifact"`
	Server   mcpDomainServer.Resolved `json:"server"`
}

type LoadPlan struct {
	Workspace workspaceDomain.WorkspaceRef `json:"workspace"`
	Servers   []WorkspaceServer            `json:"servers"`
}

type Adapter struct {
	servers ServerResolver
}

func New(
	servers ServerResolver,
) (*Adapter, error) {
	if servers == nil {
		return nil, fmt.Errorf(
			"%w: Workspace MCP adapter requires an MCP server resolver",
			basespec.ErrInvalid,
		)
	}
	return &Adapter{
		servers: servers,
	}, nil
}

// Load resolves Workspace-selected MCP Artifacts into runtime-ready MCP
// server material. It intentionally does not start or connect servers.
func (a *Adapter) Load(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifact.ArtifactRef,
) (LoadPlan, error) {
	if a == nil || a.servers == nil {
		return LoadPlan{}, basespec.ErrClosed
	}
	if err := workspace.Validate(); err != nil {
		return LoadPlan{}, err
	}

	output := LoadPlan{
		Workspace: workspace.Ref(),
		Servers:   make([]WorkspaceServer, 0, len(refs)),
	}
	seen := make(map[artifact.ArtifactRef]struct{}, len(refs))
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return LoadPlan{}, err
		}
		if ref.RootID != workspace.Artifact.RootID {
			return LoadPlan{}, fmt.Errorf(
				"%w: Workspace MCP Artifact belongs to another Root",
				workspaceDomain.ErrReferenceUnresolved,
			)
		}
		if _, duplicate := seen[ref]; duplicate {
			continue
		}
		seen[ref] = struct{}{}

		server, err := a.servers.ResolveMCPServer(ctx, ref)
		if err != nil {
			return LoadPlan{}, err
		}
		output.Servers = append(output.Servers, WorkspaceServer{
			Artifact: ref,
			Server:   server,
		})
	}
	return output, nil
}
