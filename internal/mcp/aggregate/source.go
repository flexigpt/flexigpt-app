package aggregate

import (
	"context"
	"errors"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
)

// RuntimeServerSource is the Store-to-Runtime anti-corruption adapter.
// Runtime receives only runtime/spec values and never Store values.
type RuntimeServerSource struct {
	servers     *ArtifactServerResolver
	secrets     mcpStoreServer.SecretResolver
	environment mcpStoreServer.EnvironmentResolver
}

func NewRuntimeServerSource(
	servers *ArtifactServerResolver,
	secrets mcpStoreServer.SecretResolver,
	environment mcpStoreServer.EnvironmentResolver,
) (*RuntimeServerSource, error) {
	if servers == nil {
		return nil, errors.New("MCP Artifact server resolver is required")
	}
	return &RuntimeServerSource{
		servers:     servers,
		secrets:     secrets,
		environment: environment,
	}, nil
}

func (s *RuntimeServerSource) ResolveServer(
	ctx context.Context,
	serverID mcpServer.ServerID,
) (mcpServer.ResolvedServer, error) {
	if s == nil || s.servers == nil {
		return mcpServer.ResolvedServer{}, mcpServer.ErrClosed
	}
	resolved, err := s.servers.ResolveMCPServer(ctx, serverID)
	if err != nil {
		return mcpServer.ResolvedServer{}, err
	}
	materialized, err := resolved.MaterializeTrusted(
		ctx,
		s.secrets,
		s.environment,
	)
	if err != nil {
		return mcpServer.ResolvedServer{}, err
	}
	config, err := runtimeConfig(resolved, materialized)
	if err != nil {
		return mcpServer.ResolvedServer{}, err
	}

	output := mcpServer.ResolvedServer{
		Server:  config.Server,
		Catalog: config.Catalog,
		Version: mcpServer.Digest(resolved.Version),
		Config:  config,
	}
	if err := output.Validate(); err != nil {
		return mcpServer.ResolvedServer{}, err
	}
	return output, nil
}

func (s *RuntimeServerSource) InspectRuntimeConfig(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpServer.RuntimeConfig, mcpStoreServer.Resolved, error) {
	if s == nil || s.servers == nil {
		return mcpServer.RuntimeConfig{},
			mcpStoreServer.Resolved{},
			mcpServer.ErrClosed
	}
	resolved, err := s.servers.InspectMCPServer(ctx, ref)
	if err != nil {
		return mcpServer.RuntimeConfig{}, mcpStoreServer.Resolved{}, err
	}
	materialized, err := resolved.MaterializeForInspection(
		ctx,
		s.environment,
	)
	if err != nil {
		return mcpServer.RuntimeConfig{}, resolved, err
	}
	config, err := runtimeConfig(resolved, materialized)
	if err != nil {
		return mcpServer.RuntimeConfig{}, resolved, err
	}
	return config, resolved, nil
}

func runtimeConfig(
	resolved mcpStoreServer.Resolved,
	input mcpStoreServer.MaterializedServer,
) (mcpServer.RuntimeConfig, error) {
	serverID, err := RuntimeServerIDForArtifact(resolved.Server)
	if err != nil {
		return mcpServer.RuntimeConfig{}, err
	}
	catalogID, err := RuntimeCatalogIDForCollection(resolved.Collection)
	if err != nil {
		return mcpServer.RuntimeConfig{}, err
	}

	output := mcpServer.RuntimeConfig{
		Server:                    serverID,
		Catalog:                   catalogID,
		LogicalName:               string(resolved.Document.LogicalName),
		DisplayName:               resolved.Document.DisplayName,
		OAuthClientSecretRequired: input.ClientCredentialSecretRequired,
		Policy: mcpPolicy.MCPPolicy{
			TrustLevel:    resolved.Policy.Body.TrustLevel,
			DefaultPolicy: resolved.Policy.Body.DefaultPolicy,
			ToolPolicies:  mcpPolicy.CloneToolPolicies(resolved.Policy.Body.ToolPolicies),
			AppsPolicy:    resolved.Policy.Body.AppsPolicy,
		},
		SensitiveValues: append(
			[]string(nil),
			input.SensitiveValues...,
		),
	}

	switch input.Core.Type {
	case mcpStoreServer.ServerTypeStdio:
		output.Transport = mcpServer.MCPTransportStdio
		output.Stdio = &mcpServer.MCPRuntimeStdioConfig{
			Command:          input.Core.Command,
			Args:             append([]string(nil), input.Core.Args...),
			Env:              maps.Clone(input.Core.Env),
			StartupTimeoutMS: input.TimeoutMS,
		}

	case mcpStoreServer.ServerTypeHTTP:
		authMode, err := runtimeHTTPAuthMode(input.Auth.Mode)
		if err != nil {
			return mcpServer.RuntimeConfig{}, err
		}
		output.Transport = mcpServer.MCPTransportStreamableHTTP
		output.StreamableHTTP = &mcpServer.MCPRuntimeStreamableHTTPConfig{
			URL:                         input.Core.URL,
			TimeoutMS:                   input.TimeoutMS,
			AuthMode:                    authMode,
			Headers:                     maps.Clone(input.Core.Headers),
			ClientCredentialRef:         input.ClientCredentialRef,
			ClientIDMetadataDocumentURL: input.Auth.ClientIDMetadataDocumentURL,
		}

	default:
		return mcpServer.RuntimeConfig{}, errors.New(
			"unsupported materialized MCP server transport",
		)
	}

	if err := output.Validate(); err != nil {
		return mcpServer.RuntimeConfig{}, err
	}
	return output, nil
}

func runtimeHTTPAuthMode(
	input mcpServer.MCPHTTPAuthMode,
) (mcpServer.MCPHTTPAuthMode, error) {
	switch input {
	case mcpServer.MCPHTTPAuthNone:
		return mcpServer.MCPHTTPAuthNone, nil
	case mcpServer.MCPHTTPAuthAPIKey:
		return mcpServer.MCPHTTPAuthAPIKey, nil
	case mcpServer.MCPHTTPAuthOAuth:
		return mcpServer.MCPHTTPAuthOAuth, nil
	case mcpServer.MCPHTTPAuthClientCredentials:
		return mcpServer.MCPHTTPAuthClientCredentials, nil
	default:
		return "", errors.New("unsupported materialized MCP authentication mode")
	}
}
