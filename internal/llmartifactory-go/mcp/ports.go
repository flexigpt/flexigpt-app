package mcp

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
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

type BaselineEnsurer interface {
	EnsureMCPBaselinePlugin(
		ctx context.Context,
		rootID rootModel.RootID,
	) (pluginAPI.PluginView, error)
}

type baselineEnsurer struct {
	api *Service
}

func NewBaselineEnsurer(api *Service) (BaselineEnsurer, error) {
	if api == nil || api.plugins == nil {
		return nil, fmt.Errorf(
			"%w: MCP baseline ensurer requires plugins",
			spec.ErrInvalid,
		)
	}
	return &baselineEnsurer{api: api}, nil
}

func (s *baselineEnsurer) EnsureMCPBaselinePlugin(
	ctx context.Context,
	rootID rootModel.RootID,
) (pluginAPI.PluginView, error) {
	if s == nil || s.api == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return s.api.ensureMCPBaselinePlugin(ctx, rootID)
}
