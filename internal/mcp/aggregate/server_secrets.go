package aggregate

import (
	"context"
	"fmt"
	"strings"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/secret"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
)

func (s *Service) SetMCPServerSecret(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
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

	if kind == secret.MCPSecretKindOAuthClientCredentials {
		if err := mcpAuth.ValidateOAuthClientCredentialsSecret(
			value,
			resolved.Document.OAuthClientSecretRequired(),
		); err != nil {
			return MCPServerDetails{}, err
		}
	}
	if kind == secret.MCPSecretKindHTTPHeader &&
		(strings.TrimSpace(value) == "" ||
			strings.ContainsAny(value, "\r\n\x00")) {
		return MCPServerDetails{}, fmt.Errorf(
			"%w: invalid HTTP header secret value",
			mcpAuth.ErrMCPInvalidAuthRequest,
		)
	}

	secretRef, err := secret.NewMCPSecretRefString(
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
		data.Inputs = map[string]serverMCPDomain.InputBinding{}
	}
	data.Inputs[input] = serverMCPDomain.InputBinding{
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
	ref artifactModel.ArtifactRef,
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
	ref artifactModel.ArtifactRef,
	input string,
) (
	serverMCPDomain.Resolved,
	secret.MCPSecretKind,
	string,
	error,
) {
	resolved, err := s.servers.InspectMCPServer(ctx, ref)
	if err != nil {
		return serverMCPDomain.Resolved{}, "", "", err
	}

	declaration, found := resolved.Document.Configuration.Install.Inputs[input]
	if !found {
		return serverMCPDomain.Resolved{}, "", "", fmt.Errorf(
			"%w: MCP secret input %q is not declared",
			mcpAuth.ErrMCPInvalidAuthRequest,
			input,
		)
	}

	switch declaration.Kind {
	case serverMCPDomain.InputOAuthClientCredentials:
		if resolved.Document.Configuration.Auth.ClientCredentialsInput != input {
			return serverMCPDomain.Resolved{}, "", "", fmt.Errorf(
				"%w: MCP secret input %q is not an OAuth client credential input",
				mcpAuth.ErrMCPInvalidAuthRequest,
				input,
			)
		}
		return resolved,
			secret.MCPSecretKindOAuthClientCredentials,
			"clientCredentials",
			nil

	case serverMCPDomain.InputSecret:
		targets, err := resolved.Document.SecretInputTargets()
		if err != nil {
			return serverMCPDomain.Resolved{}, "", "", err
		}
		target, found := targets[input]
		if !found {
			return serverMCPDomain.Resolved{}, "", "", fmt.Errorf(
				"%w: MCP secret input %q has no target",
				mcpAuth.ErrMCPInvalidAuthRequest,
				input,
			)
		}
		if target.Kind == serverMCPDomain.SecretInputTargetHTTPHeader {
			return resolved,
				secret.MCPSecretKindHTTPHeader,
				target.Slot,
				nil
		}
		return resolved,
			secret.MCPSecretKindStdioEnv,
			target.Slot,
			nil

	default:
		return serverMCPDomain.Resolved{}, "", "", fmt.Errorf(
			"%w: MCP input %q does not accept a secret",
			mcpAuth.ErrMCPInvalidAuthRequest,
			input,
		)
	}
}
