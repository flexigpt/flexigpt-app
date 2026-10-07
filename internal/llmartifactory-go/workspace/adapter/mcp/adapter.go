package mcp

import (
	"context"
	"errors"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
)

type ArtifactReader interface {
	Get(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (artifactModel.Artifact, error)
}

// ServerResolver is satisfied directly by mcp.Service. Workspace reads the
// existing MCP service result and projects only the runtime-neutral portion it
// needs; there is no MCP-to-Workspace forwarding method.
type ServerResolver interface {
	ResolveMCPServer(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (mcpAPI.ServerRead, error)
}

type WorkspaceServer struct {
	Artifact artifactModel.ArtifactRef `json:"-"`
	Server   serverMCPDomain.Resolved  `json:"-"`
}

type LoadPlan struct {
	Workspace artifactModel.ArtifactRef `json:"-"`
	Servers   []WorkspaceServer         `json:"-"`
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
			spec.ErrInvalid,
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
	refs []artifactModel.ArtifactRef,
) (LoadPlan, error) {
	if a == nil ||
		a.artifacts == nil ||
		a.servers == nil {
		return LoadPlan{}, spec.ErrClosed
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
		if record.State != artifactModel.StateAvailable {
			continue
		}
		if !record.Enabled {
			continue
		}

		read, err := a.servers.ResolveMCPServer(ctx, ref)
		if err != nil {
			if fatal := fatalLoadError(ctx, err); fatal != nil {
				return LoadPlan{}, fatal
			}
			continue
		}
		output.Servers = append(output.Servers, WorkspaceServer{
			Artifact: ref,
			Server:   read.Resolved,
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
	if errors.Is(err, spec.ErrClosed) {
		return err
	}
	return nil
}
