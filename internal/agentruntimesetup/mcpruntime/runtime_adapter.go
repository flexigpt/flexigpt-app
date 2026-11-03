// Package mcpruntime adapts source-backed MCP installations to the
// application MCP runtime and the inference consumer's narrow capability.
// It supports both built-in and user-authored MCP installations.
package mcpruntime

import (
	"context"
	"fmt"

	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
)

// OAuthTokenStore is the narrow mutation capability needed for app-managed
// OAuth tokens. Installation-secret plaintext is supplied separately through
// serverMCPDomain.SecretResolver.
type OAuthTokenStore interface {
	SetMCPSecret(ctx context.Context, ref, value string) (hash string, nonEmpty bool, err error)
	DeleteSecret(ctx context.Context, ref string) error
}

// SourceAdapter translates MCP Artifact configuration into the opaque source
// capability consumed by MCP Runtime Manager. It owns no runtime manager and
// has no late binding phase.
type SourceAdapter struct {
	store       mcpAPI.ServerStore
	secrets     serverMCPDomain.SecretResolver
	oauthTokens OAuthTokenStore
	environment serverMCPDomain.EnvironmentResolver
}

func NewSourceAdapter(
	store mcpAPI.ServerStore,
	secrets serverMCPDomain.SecretResolver,
	oauthTokens OAuthTokenStore,
	environment serverMCPDomain.EnvironmentResolver,
) (*SourceAdapter, error) {
	if store == nil ||
		secrets == nil ||
		oauthTokens == nil {
		return nil, fmt.Errorf(
			"%w: MCP runtime adapter dependencies are incomplete",
			storeSpec.ErrInvalid,
		)
	}

	return &SourceAdapter{
		store:       store,
		secrets:     secrets,
		oauthTokens: oauthTokens,
		environment: environment,
	}, nil
}

// ResolveServer implements the MCP runtime's source capability. Artifact
// identities and family documents do not escape into the runtime manager.
func (a *SourceAdapter) ResolveServer(
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
func (a *SourceAdapter) InspectRuntimeConfig(
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
