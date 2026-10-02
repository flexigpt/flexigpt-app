package aggregate

import (
	"context"
	"encoding/base64"
	"errors"
	"maps"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

// RuntimeServerSource is the Store-to-Runtime anti-corruption adapter.
// Runtime receives only runtime/spec values and never Store values.
type RuntimeServerSource struct {
	servers     *ArtifactServerResolver
	secrets     mcpDomainServer.SecretResolver
	environment mcpDomainServer.EnvironmentResolver
}

func NewRuntimeServerSource(
	servers *ArtifactServerResolver,
	secrets mcpDomainServer.SecretResolver,
	environment mcpDomainServer.EnvironmentResolver,
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
	resolved mcpDomainServer.Resolved,
) (mcpServer.RuntimeConfig, error) {
	if s == nil {
		return mcpServer.RuntimeConfig{}, mcpServer.ErrClosed
	}
	materialized, err := resolved.MaterializeForInspection(
		ctx,
		s.environment,
	)
	if err != nil {
		return mcpServer.RuntimeConfig{}, err
	}
	return runtimeConfig(resolved, materialized)
}

func runtimeConfig(
	resolved mcpDomainServer.Resolved,
	input mcpDomainServer.MaterializedServer,
) (mcpServer.RuntimeConfig, error) {
	serverID, err := runtimeServerIDForArtifact(resolved.Server)
	if err != nil {
		return mcpServer.RuntimeConfig{}, err
	}
	catalogID, err := runtimeCatalogIDForRoot(resolved.Server.RootID)
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
			ToolPolicies:  mcpPolicy.Clone(resolved.Policy.Body).ToolPolicies,
			AppsPolicy:    resolved.Policy.Body.AppsPolicy,
		},
		SensitiveValues: append(
			[]string(nil),
			input.SensitiveValues...,
		),
		Include: runtimeInclude(resolved.Document.Include),
	}

	switch input.Core.Type {
	case mcpDomainServer.ServerTypeStdio:
		output.Transport = mcpServer.MCPTransportStdio
		output.Stdio = &mcpServer.MCPRuntimeStdioConfig{
			Command:          input.Core.Command,
			Args:             append([]string(nil), input.Core.Args...),
			Env:              maps.Clone(input.Core.Env),
			StartupTimeoutMS: input.TimeoutMS,
		}

	case mcpDomainServer.ServerTypeHTTP:
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

func runtimeCatalogIDForRoot(
	rootID root.RootID,
) (mcpServer.CatalogID, error) {
	if err := rootID.Validate(); err != nil {
		return "", err
	}
	return mcpServer.CatalogID(
		artifactCatalogIDPrefix +
			base64.RawURLEncoding.EncodeToString([]byte(rootID)),
	), nil
}

func runtimeInclude(
	input *mcpDomainServer.Include,
) *mcpServer.MCPInclude {
	if input == nil {
		return nil
	}
	return &mcpServer.MCPInclude{
		Tools:     slices.Clone(input.Tools),
		Resources: slices.Clone(input.Resources),
		Prompts:   slices.Clone(input.Prompts),
	}
}

func runtimeHTTPAuthMode(
	input mcpv1.HTTPAuthMode,
) (mcpServer.MCPHTTPAuthMode, error) {
	switch input {
	case mcpv1.HTTPAuthModeNone:
		return mcpServer.MCPHTTPAuthNone, nil
	case mcpv1.HTTPAuthModeAPIKey:
		return mcpServer.MCPHTTPAuthAPIKey, nil
	case mcpv1.HTTPAuthModeOAuth:
		return mcpServer.MCPHTTPAuthOAuth, nil
	case mcpv1.HTTPAuthModeClientCredentials:
		return mcpServer.MCPHTTPAuthClientCredentials, nil
	default:
		return "", errors.New("unsupported materialized MCP authentication mode")
	}
}
