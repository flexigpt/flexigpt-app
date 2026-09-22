package mcp

import (
	"context"
	"errors"
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
	Artifact artifact.ArtifactRef     `json:"-"`
	Server   mcpDomainServer.Resolved `json:"-"`
}

type LoadPlan struct {
	Workspace artifact.ArtifactRef `json:"-"`
	Servers   []WorkspaceServer    `json:"-"`
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

// Load resolves capability-authorized MCP Artifacts into runtime-ready server
// material. Protected built-in ArtifactRefs may belong to another Root. It
// intentionally does not start or connect servers.
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

	output := LoadPlan{
		Workspace: workspace.Ref(),
		Servers:   make([]WorkspaceServer, 0, len(refs)),
	}
	for _, ref := range refs {
		record, err := a.artifacts.Get(ctx, ref)
		if err != nil {
			if fatal := fatalLoadError(ctx, err); fatal != nil {
				return LoadPlan{}, fatal
			}
			continue
		}
		if record.State != artifact.StateAvailable {
			continue
		}
		if !record.Enabled {
			continue
		}

		server, err := a.servers.ResolveMCPServer(ctx, ref)
		if err != nil {
			if fatal := fatalLoadError(ctx, err); fatal != nil {
				return LoadPlan{}, fatal
			}
			continue
		}
		output.Servers = append(output.Servers, WorkspaceServer{
			Artifact: ref,
			Server:   server,
		})
	}
	return output, nil
}

// fatalLoadError distinguishes operation-level failure from one server being
// unavailable or not runtime-ready. Per-server failures do not discard other
// successfully loaded servers.
func fatalLoadError(
	ctx context.Context,
	err error,
) error {
	if contextErr := ctx.Err(); contextErr != nil {
		return contextErr
	}
	if errors.Is(err, basespec.ErrClosed) {
		return err
	}
	return nil
}
