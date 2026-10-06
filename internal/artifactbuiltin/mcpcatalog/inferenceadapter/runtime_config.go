package inferenceadapter

import (
	"errors"
	"maps"
	"slices"

	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/policy"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	mcpv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/contract/v1"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
)

func runtimeConfig(
	resolved serverMCPDomain.Resolved,
	input serverMCPDomain.MaterializedServer,
) (mcpServer.RuntimeConfig, error) {
	serverID, err := ServerIDForArtifact(resolved.Server)
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
		SensitiveValues: slices.Clone(input.SensitiveValues),
		Include:         runtimeInclude(resolved.Document.Include),
	}

	switch input.Core.Type {
	case serverMCPDomain.ServerTypeStdio:
		output.Transport = mcpServer.MCPTransportStdio
		output.Stdio = &mcpServer.MCPRuntimeStdioConfig{
			Command:          input.Core.Command,
			Args:             slices.Clone(input.Core.Args),
			Env:              maps.Clone(input.Core.Env),
			StartupTimeoutMS: input.TimeoutMS,
		}

	case serverMCPDomain.ServerTypeHTTP:
		authMode, err := HTTPAuthMode(input.Auth.Mode)
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

func runtimeInclude(
	input *serverMCPDomain.Include,
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

// HTTPAuthMode converts the declaration enum without materializing an
// installation. Management also uses it for incomplete-installation health.
func HTTPAuthMode(
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
