package mcp

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type ArtifactReader interface {
	Get(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)
}

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
	Artifact        artifact.ArtifactRef     `json:"artifact"`
	Server          mcpDomainServer.Resolved `json:"server"`
	RuntimeDisabled bool                     `json:"runtimeDisabled"`
}

type LoadPlan struct {
	Workspace workspaceDomain.WorkspaceRef `json:"workspace"`
	Servers   []WorkspaceServer            `json:"servers"`
}

type Adapter struct {
	artifacts ArtifactReader
	servers   ServerResolver
}

func New(
	artifacts ArtifactReader,
	servers ServerResolver,
) (*Adapter, error) {
	if artifacts == nil || servers == nil {
		return nil, fmt.Errorf(
			"%w: Workspace MCP adapter dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Adapter{
		artifacts: artifacts,
		servers:   servers,
	}, nil
}

// Load resolves Workspace-selected MCP Artifacts into runtime-ready MCP
// server material. It intentionally does not start or connect servers.
func (a *Adapter) Load(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifact.ArtifactRef,
) (LoadPlan, error) {
	if a == nil ||
		a.artifacts == nil ||
		a.servers == nil {
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

		record, err := a.artifacts.Get(ctx, ref)
		if err != nil {
			return LoadPlan{}, err
		}
		if !record.Enabled ||
			record.State != artifact.StateAvailable {
			return LoadPlan{}, fmt.Errorf(
				"%w: Workspace MCP Artifact %q is unavailable",
				workspaceDomain.ErrReferenceUnresolved,
				ref.ArtifactID,
			)
		}
		settings, err := workspaceDomain.DecodeArtifactData(
			record.Data,
		)
		if err != nil {
			return LoadPlan{}, err
		}

		server, err := a.servers.ResolveMCPServer(ctx, ref)
		if err != nil {
			return LoadPlan{}, err
		}
		if settings.RuntimeDisabled {
			server.RuntimeEnabled = false
		}
		output.Servers = append(output.Servers, WorkspaceServer{
			Artifact:        ref,
			Server:          server,
			RuntimeDisabled: settings.RuntimeDisabled,
		})
	}
	return output, nil
}
