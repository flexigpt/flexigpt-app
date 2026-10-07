// Package mcpruntime adapts source-backed MCP installations to the
// application MCP runtime and the inference consumer's narrow capability.
// It supports both built-in and user-authored MCP installations.
package mcpruntime

import (
	"context"
	"fmt"

	mcpConnection "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/connection"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
)

// OAuthTokenStore is the narrow mutation capability needed for app-managed
// OAuth tokens. Installation-secret plaintext is supplied separately through
// serverMCPDomain.SecretResolver.
type OAuthTokenStore interface {
	SetMCPSecret(
		ctx context.Context,
		ref string,
		value string,
	) (hash string, nonEmpty bool, err error)

	DeleteSecret(ctx context.Context, ref string) error
}

// RuntimeAdapter is constructed with immutable resolution dependencies.
// BindRuntime completes the runtime/inference side of assembly exactly once.
//
// The runtime may call ResolveServer before binding is complete because that
// method uses only constructor dependencies. Inference must not receive the
// adapter until BindRuntime succeeds.
type RuntimeAdapter struct {
	store       mcpAPI.ServerStore
	secrets     serverMCPDomain.SecretResolver
	oauthTokens OAuthTokenStore
	environment serverMCPDomain.EnvironmentResolver

	runtime *mcpConnection.MCPRuntimeManager
}

func NewRuntimeAdapter(
	store mcpAPI.ServerStore,
	secrets serverMCPDomain.SecretResolver,
	oauthTokens OAuthTokenStore,
	environment serverMCPDomain.EnvironmentResolver,
) (*RuntimeAdapter, error) {
	if store == nil ||
		secrets == nil ||
		oauthTokens == nil {
		return nil, fmt.Errorf(
			"%w: MCP runtime adapter dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	return &RuntimeAdapter{
		store:       store,
		secrets:     secrets,
		oauthTokens: oauthTokens,
		environment: environment,
	}, nil
}

// BindRuntime completes application assembly. The caller retains ownership
// of runtime shutdown and must drain inference before closing the runtime.
func (a *RuntimeAdapter) BindRuntime(
	runtime *mcpConnection.MCPRuntimeManager,
) error {
	if runtime == nil {
		return fmt.Errorf(
			"%w: MCP runtime is required",
			spec.ErrInvalid,
		)
	}
	if a.runtime != nil {
		return fmt.Errorf(
			"%w: MCP runtime adapter is already bound",
			spec.ErrConflict,
		)
	}
	a.runtime = runtime
	return nil
}

// ResolveServer implements the MCP runtime's source capability. Artifact
// identities and family documents do not escape into the runtime manager.
func (a *RuntimeAdapter) ResolveServer(
	ctx context.Context,
	serverID mcpServer.ServerID,
) (mcpServer.ResolvedServer, error) {
	ref, err := ArtifactRefForServerID(serverID)
	if err != nil {
		return mcpServer.ResolvedServer{}, err
	}
	read, err := a.store.ResolveMCPServer(ctx, ref)
	if err != nil {
		return mcpServer.ResolvedServer{}, err
	}

	materialized, err := read.Resolved.MaterializeTrusted(
		ctx,
		a.secrets,
		a.environment,
	)
	if err != nil {
		return mcpServer.ResolvedServer{}, err
	}
	config, err := runtimeConfig(read.Resolved, materialized)
	if err != nil {
		return mcpServer.ResolvedServer{}, err
	}

	output := mcpServer.ResolvedServer{
		Server:  config.Server,
		Catalog: config.Catalog,
		Version: mcpServer.Digest(read.Resolved.Version),
		Config:  config,
	}
	if err := output.Validate(); err != nil {
		return mcpServer.ResolvedServer{}, err
	}
	return output, nil
}

// InspectRuntimeConfig converts an already-resolved installation without
// resolving its secret values. Management owns the resulting health view.
func (a *RuntimeAdapter) InspectRuntimeConfig(
	ctx context.Context,
	resolved serverMCPDomain.Resolved,
) (mcpServer.RuntimeConfig, error) {
	materialized, err := resolved.MaterializeForInspection(
		ctx,
		a.environment,
	)
	if err != nil {
		return mcpServer.RuntimeConfig{}, err
	}
	return runtimeConfig(resolved, materialized)
}

func (a *RuntimeAdapter) Status(
	ctx context.Context,
	server mcpServer.ServerID,
) (*mcpServer.MCPServerRuntimeSnapshot, error) {
	runtime, err := a.runtimeManager()
	if err != nil {
		return nil, err
	}
	return runtime.Status(ctx, server)
}

func (a *RuntimeAdapter) ListTools(
	ctx context.Context,
	server mcpServer.ServerID,
) ([]mcpServer.MCPToolCapability, error) {
	runtime, err := a.runtimeManager()
	if err != nil {
		return nil, err
	}
	return runtime.ListTools(ctx, server)
}

func (a *RuntimeAdapter) ListResources(
	ctx context.Context,
	server mcpServer.ServerID,
) ([]mcpServer.MCPResourceRef, error) {
	runtime, err := a.runtimeManager()
	if err != nil {
		return nil, err
	}
	return runtime.ListResources(ctx, server)
}

func (a *RuntimeAdapter) ListResourceTemplates(
	ctx context.Context,
	server mcpServer.ServerID,
) ([]mcpServer.MCPResourceTemplateRef, error) {
	runtime, err := a.runtimeManager()
	if err != nil {
		return nil, err
	}
	return runtime.ListResourceTemplates(ctx, server)
}

func (a *RuntimeAdapter) ListPrompts(
	ctx context.Context,
	server mcpServer.ServerID,
) ([]mcpServer.MCPPromptRef, error) {
	runtime, err := a.runtimeManager()
	if err != nil {
		return nil, err
	}
	return runtime.ListPrompts(ctx, server)
}

func (a *RuntimeAdapter) ReadResource(
	ctx context.Context,
	server mcpServer.ServerID,
	uri string,
) (*mcpServer.MCPReadResourceResponseBody, error) {
	runtime, err := a.runtimeManager()
	if err != nil {
		return nil, err
	}
	return runtime.ReadResource(ctx, server, uri)
}

func (a *RuntimeAdapter) GetPrompt(
	ctx context.Context,
	server mcpServer.ServerID,
	name string,
	arguments map[string]string,
) (*mcpServer.MCPGetPromptResponseBody, error) {
	runtime, err := a.runtimeManager()
	if err != nil {
		return nil, err
	}
	return runtime.GetPrompt(ctx, server, name, arguments)
}

func (a *RuntimeAdapter) runtimeManager() (
	*mcpConnection.MCPRuntimeManager,
	error,
) {
	if a.runtime == nil {
		return nil, spec.ErrClosed
	}
	return a.runtime, nil
}
