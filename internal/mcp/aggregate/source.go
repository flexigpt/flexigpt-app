package aggregate

import (
	"context"
	"errors"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
	mcpStore "github.com/flexigpt/flexigpt-app/internal/mcp/store"
	mcpArtifact "github.com/flexigpt/flexigpt-app/internal/mcp/store/artifact"
)

type ServerSource struct {
	store       *mcpStore.API
	secrets     mcpArtifact.SecretResolver
	environment mcpArtifact.EnvironmentResolver
}

func NewServerSource(
	store *mcpStore.API,
	secrets mcpArtifact.SecretResolver,
	environment mcpArtifact.EnvironmentResolver,
) (*ServerSource, error) {
	if store == nil {
		return nil, errors.New("MCP store is required")
	}
	return &ServerSource{
		store:       store,
		secrets:     secrets,
		environment: environment,
	}, nil
}

func (s *ServerSource) ResolveServer(
	ctx context.Context,
	serverID mcpSpec.ServerID,
) (mcpSpec.ResolvedServer, error) {
	if s == nil || s.store == nil {
		return mcpSpec.ResolvedServer{}, mcpSpec.ErrClosed
	}
	ref, err := ArtifactRefForServerID(serverID)
	if err != nil {
		return mcpSpec.ResolvedServer{}, err
	}
	resolved, err := s.store.ResolveMCPServer(ctx, ref)
	if err != nil {
		return mcpSpec.ResolvedServer{}, err
	}
	config, err := resolved.MaterializeTrusted(
		ctx,
		s.secrets,
		s.environment,
	)
	if err != nil {
		return mcpSpec.ResolvedServer{}, err
	}
	runtimeConfig, err := runtimeConfig(config)
	if err != nil {
		return mcpSpec.ResolvedServer{}, err
	}
	output := mcpSpec.ResolvedServer{
		Server:  runtimeConfig.Server,
		Catalog: runtimeConfig.Catalog,
		Version: mcpSpec.Digest(resolved.Version),
		Config:  runtimeConfig,
	}
	if err := output.Validate(); err != nil {
		return mcpSpec.ResolvedServer{}, err
	}
	return output, nil
}

func (s *ServerSource) InspectRuntimeConfig(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpSpec.RuntimeConfig, mcpArtifact.Resolved, error) {
	if s == nil || s.store == nil {
		return mcpSpec.RuntimeConfig{},
			mcpArtifact.Resolved{},
			mcpSpec.ErrClosed
	}
	resolved, err := s.store.InspectMCPServer(ctx, ref)
	if err != nil {
		return mcpSpec.RuntimeConfig{}, mcpArtifact.Resolved{}, err
	}
	config, err := resolved.MaterializeForInspection(
		ctx,
		s.environment,
	)
	if err != nil {
		return mcpSpec.RuntimeConfig{}, resolved, err
	}
	runtimeConfig, err := runtimeConfig(config)
	if err != nil {
		return mcpSpec.RuntimeConfig{}, resolved, err
	}
	return runtimeConfig, resolved, nil
}

func runtimeConfig(
	input mcpArtifact.RuntimeConfig,
) (mcpSpec.RuntimeConfig, error) {
	serverID, err := ServerIDForArtifact(input.Server)
	if err != nil {
		return mcpSpec.RuntimeConfig{}, err
	}
	catalogID, err := CatalogIDForCollection(input.Collection)
	if err != nil {
		return mcpSpec.RuntimeConfig{}, err
	}

	output := mcpSpec.RuntimeConfig{
		Server:                    serverID,
		Catalog:                   catalogID,
		LogicalName:               input.LogicalName,
		DisplayName:               input.DisplayName,
		Transport:                 mcpSpec.MCPTransportType(input.Transport),
		OAuthClientSecretRequired: input.OAuthClientSecretRequired,
		TrustLevel:                input.TrustLevel,
		DefaultPolicy:             runtimeServerPolicy(input.DefaultPolicy),
		ToolPolicies:              runtimeToolPolicies(input.ToolPolicies),
		AppsPolicy:                runtimeAppsPolicy(input.AppsPolicy),
		SensitiveValues: append(
			[]string(nil),
			input.SensitiveValues...,
		),
	}
	if input.Stdio != nil {
		output.Stdio = &mcpSpec.MCPRuntimeStdioConfig{
			Command:          input.Stdio.Command,
			Args:             append([]string(nil), input.Stdio.Args...),
			Env:              maps.Clone(input.Stdio.Env),
			StartupTimeoutMS: input.Stdio.StartupTimeoutMS,
		}
	}
	if input.StreamableHTTP != nil {
		output.StreamableHTTP = &mcpSpec.MCPRuntimeStreamableHTTPConfig{
			URL:                         input.StreamableHTTP.URL,
			TimeoutMS:                   input.StreamableHTTP.TimeoutMS,
			AuthMode:                    input.StreamableHTTP.AuthMode,
			Headers:                     maps.Clone(input.StreamableHTTP.Headers),
			ClientCredentialRef:         input.StreamableHTTP.ClientCredentialRef,
			ClientIDMetadataDocumentURL: input.StreamableHTTP.ClientIDMetadataDocumentURL,
		}
	}
	if err := output.Validate(); err != nil {
		return mcpSpec.RuntimeConfig{}, err
	}
	return output, nil
}

func runtimeServerPolicy(
	input mcpSpec.MCPServerPolicy,
) mcpSpec.MCPServerPolicy {
	return mcpSpec.MCPServerPolicy{
		DefaultApprovalRule: input.DefaultApprovalRule,

		DefaultExecutionMode:          input.DefaultExecutionMode,
		RequireApprovalForUnknownRisk: input.RequireApprovalForUnknownRisk,
		RequireApprovalForWrite:       input.RequireApprovalForWrite,
		RequireApprovalForDestructive: input.RequireApprovalForDestructive,
	}
}

func runtimeAppsPolicy(
	input mcpSpec.MCPAppsPolicy,
) mcpSpec.MCPAppsPolicy {
	return mcpSpec.MCPAppsPolicy{
		Enabled:                          input.Enabled,
		AllowAppInitiatedToolCalls:       input.AllowAppInitiatedToolCalls,
		RequireApprovalForOpenLink:       input.RequireApprovalForOpenLink,
		RequireApprovalForContextUpdates: input.RequireApprovalForContextUpdates,
	}
}

func runtimeToolPolicies(
	input map[string]mcpSpec.MCPToolPolicyOverride,
) map[string]mcpSpec.MCPToolPolicyOverride {
	if input == nil {
		return nil
	}
	output := make(
		map[string]mcpSpec.MCPToolPolicyOverride,
		len(input),
	)
	for name, value := range input {
		override := mcpSpec.MCPToolPolicyOverride{
			ToolName:         value.ToolName,
			AllowStaleDigest: value.AllowStaleDigest,
			ExpectedDigest:   value.ExpectedDigest,
		}
		if value.ApprovalRule != nil {
			rule := *value.ApprovalRule
			override.ApprovalRule = &rule
		}
		if value.ExecutionMode != nil {
			mode := *value.ExecutionMode
			override.ExecutionMode = &mode
		}
		output[name] = override
	}
	return output
}
