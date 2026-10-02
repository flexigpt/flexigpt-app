package aggregate

import (
	"context"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpDomainSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/secret"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

func (s *Service) SetMCPServerSecret(
	ctx context.Context,
	ref artifact.ArtifactRef,
	input string,
	value string,
) (MCPServerDetails, error) {
	if err := s.ready(); err != nil {
		return MCPServerDetails{}, err
	}

	settings, err := s.store.GetServerSettings(ctx, ref)
	if err != nil {
		return MCPServerDetails{}, err
	}
	resolved, kind, slot, err := s.serverSecretTarget(ctx, ref, input)
	if err != nil {
		return MCPServerDetails{}, err
	}

	if kind == mcpDomainSecret.MCPSecretKindOAuthClientCredentials {
		if err := mcpAuth.ValidateOAuthClientCredentialsSecret(
			value,
			resolved.Document.OAuthClientSecretRequired(),
		); err != nil {
			return MCPServerDetails{}, err
		}
	}
	if kind == mcpDomainSecret.MCPSecretKindHTTPHeader &&
		(strings.TrimSpace(value) == "" ||
			strings.ContainsAny(value, "\r\n\x00")) {
		return MCPServerDetails{}, fmt.Errorf(
			"%w: invalid HTTP header secret value",
			mcpAuth.ErrMCPInvalidAuthRequest,
		)
	}

	secretRef, err := mcpDomainSecret.NewMCPSecretRefString(
		ref,
		kind,
		slot,
	)
	if err != nil {
		return MCPServerDetails{}, err
	}
	if _, _, err := s.secrets.SetMCPSecret(ctx, secretRef, value); err != nil {
		return MCPServerDetails{}, err
	}

	data := resolved.Installation.Clone()
	if data.Inputs == nil {
		data.Inputs = map[string]mcpDomainServer.InputBinding{}
	}
	data.Inputs[input] = mcpDomainServer.InputBinding{
		SecretRef: secretRef,
	}
	return s.saveMCPServerSettings(
		ctx,
		ref,
		settings.InstallationRevision,
		data,
	)
}

func (s *Service) ClearMCPServerSecret(
	ctx context.Context,
	ref artifact.ArtifactRef,
	input string,
) (MCPServerDetails, error) {
	if err := s.ready(); err != nil {
		return MCPServerDetails{}, err
	}

	settings, err := s.store.GetServerSettings(ctx, ref)
	if err != nil {
		return MCPServerDetails{}, err
	}
	resolved, _, _, err := s.serverSecretTarget(ctx, ref, input)
	if err != nil {
		return MCPServerDetails{}, err
	}

	data := resolved.Installation.Clone()
	binding, found := data.Inputs[input]
	if !found || binding.SecretRef == "" {
		return s.GetMCPServer(ctx, ref)
	}
	delete(data.Inputs, input)

	return s.saveMCPServerSettings(
		ctx,
		ref,
		settings.InstallationRevision,
		data,
	)
}

func (s *Service) serverSecretTarget(
	ctx context.Context,
	ref artifact.ArtifactRef,
	input string,
) (
	mcpDomainServer.Resolved,
	mcpDomainSecret.MCPSecretKind,
	string,
	error,
) {
	resolved, err := s.servers.InspectMCPServer(ctx, ref)
	if err != nil {
		return mcpDomainServer.Resolved{}, "", "", err
	}

	declaration, found := resolved.Document.Configuration.Install.Inputs[input]
	if !found {
		return mcpDomainServer.Resolved{}, "", "", fmt.Errorf(
			"%w: MCP secret input %q is not declared",
			mcpAuth.ErrMCPInvalidAuthRequest,
			input,
		)
	}

	switch declaration.Kind {
	case mcpDomainServer.InputOAuthClientCredentials:
		if resolved.Document.Configuration.Auth.ClientCredentialsInput != input {
			return mcpDomainServer.Resolved{}, "", "", fmt.Errorf(
				"%w: MCP secret input %q is not an OAuth client credential input",
				mcpAuth.ErrMCPInvalidAuthRequest,
				input,
			)
		}
		return resolved,
			mcpDomainSecret.MCPSecretKindOAuthClientCredentials,
			"clientCredentials",
			nil

	case mcpDomainServer.InputSecret:
		targets, err := resolved.Document.SecretInputTargets()
		if err != nil {
			return mcpDomainServer.Resolved{}, "", "", err
		}
		target, found := targets[input]
		if !found {
			return mcpDomainServer.Resolved{}, "", "", fmt.Errorf(
				"%w: MCP secret input %q has no target",
				mcpAuth.ErrMCPInvalidAuthRequest,
				input,
			)
		}
		if target.Kind == mcpDomainServer.SecretInputTargetHTTPHeader {
			return resolved,
				mcpDomainSecret.MCPSecretKindHTTPHeader,
				target.Slot,
				nil
		}
		return resolved,
			mcpDomainSecret.MCPSecretKindStdioEnv,
			target.Slot,
			nil

	default:
		return mcpDomainServer.Resolved{}, "", "", fmt.Errorf(
			"%w: MCP input %q does not accept a secret",
			mcpAuth.ErrMCPInvalidAuthRequest,
			input,
		)
	}
}
