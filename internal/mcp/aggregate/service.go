package aggregate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpDomainSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/secret"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

type AuthState interface {
	ClearAuthStatus(server mcpServer.ServerID)

	BuildAuthHealth(
		ctx context.Context,
		config mcpServer.RuntimeConfig,
	) mcpAuth.MCPAuthHealth
}

// SecretStore is deliberately narrow. Aggregate owns the mapping from runtime
// identity to artifact-scoped secret identity; the app supplies persistence.
type SecretStore interface {
	ResolveSecret(ctx context.Context, ref string) (string, error)

	SetMCPSecret(
		ctx context.Context,
		ref string,
		value string,
	) (hash string, nonEmpty bool, err error)

	DeleteSecret(ctx context.Context, ref string) error
}

type Dependencies struct {
	Lifecycle *Lifecycle
	Servers   *ArtifactServerResolver
	Source    *RuntimeServerSource
	Store     mcpConsumerAPI.ServerStore
	Auth      AuthState
	Secrets   SecretStore
}

type Service struct {
	lifecycle *Lifecycle
	servers   *ArtifactServerResolver
	source    *RuntimeServerSource
	store     mcpConsumerAPI.ServerStore
	auth      AuthState
	secrets   SecretStore
}

type SecretWriteResult struct {
	SecretRef string `json:"secretRef"`
	SHA256    string `json:"sha256,omitempty"`
	NonEmpty  bool   `json:"nonEmpty"`
}

func NewService(dependencies Dependencies) (*Service, error) {
	if dependencies.Lifecycle == nil ||
		dependencies.Servers == nil ||
		dependencies.Source == nil ||
		dependencies.Store == nil ||
		dependencies.Auth == nil ||
		dependencies.Secrets == nil {
		return nil, errors.New("MCP aggregate dependencies are incomplete")
	}

	return &Service{
		lifecycle: dependencies.Lifecycle,
		servers:   dependencies.Servers,
		source:    dependencies.Source,
		store:     dependencies.Store,
		auth:      dependencies.Auth,
		secrets:   dependencies.Secrets,
	}, nil
}

func (s *Service) InspectRuntimeConfig(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpServer.RuntimeConfig, mcpDomainServer.Resolved, error) {
	if err := s.ready(); err != nil {
		return mcpServer.RuntimeConfig{}, mcpDomainServer.Resolved{}, err
	}
	return s.source.InspectRuntimeConfig(ctx, ref)
}

func (s *Service) UpdateServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpDomainServer.ServerData,
) (artifact.Artifact, error) {
	if err := s.ready(); err != nil {
		return artifact.Artifact{}, err
	}
	value, err := s.lifecycle.UpdateServerInstallation(
		ctx,
		ref,
		expectedArtifactRevision,
		data,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	s.clearServerAuthStatus(ref)
	return value, nil
}

func (s *Service) UpdateProtectedServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	data mcpDomainServer.ServerData,
) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := s.lifecycle.UpdateProtectedServerInstallation(
		ctx,
		ref,
		expectedOverlayRevision,
		data,
	); err != nil {
		return err
	}
	s.clearServerAuthStatus(ref)
	return nil
}

func (s *Service) PutServerSecret(
	ctx context.Context,
	ref artifact.ArtifactRef,
	kind mcpDomainSecret.MCPSecretKind,
	slot string,
	value string,
) (SecretWriteResult, error) {
	if err := s.ready(); err != nil {
		return SecretWriteResult{}, err
	}
	if kind == mcpDomainSecret.MCPSecretKindOAuthToken {
		return SecretWriteResult{}, fmt.Errorf(
			"%w: OAuth token secrets are runtime-managed",
			mcpAuth.ErrMCPInvalidAuthRequest,
		)
	}

	installation, err := s.store.GetServerInstallation(ctx, ref)
	if err != nil {
		return SecretWriteResult{}, err
	}
	if err := installation.Document.AcceptsSecretTarget(
		kind,
		slot,
	); err != nil {
		return SecretWriteResult{}, err
	}

	if kind == mcpDomainSecret.MCPSecretKindOAuthClientCredentials {
		switch installation.Document.Configuration.Auth.Mode {
		case mcpv1.HTTPAuthModeNone, mcpv1.HTTPAuthModeClientCredentials:
		default:
			return SecretWriteResult{}, fmt.Errorf(
				"%w: MCP server does not declare OAuth client credentials",
				mcpAuth.ErrMCPInvalidAuthRequest,
			)
		}
		if err := mcpAuth.ValidateOAuthClientCredentialsSecret(
			value,
			installation.Document.OAuthClientSecretRequired(),
		); err != nil {
			return SecretWriteResult{}, err
		}
	}

	if kind == mcpDomainSecret.MCPSecretKindHTTPHeader &&
		(strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n\x00")) {
		return SecretWriteResult{}, fmt.Errorf(
			"%w: invalid HTTP header secret value",
			mcpAuth.ErrMCPInvalidAuthRequest,
		)
	}

	if err := s.lifecycle.InvalidateServer(ctx, ref); err != nil {
		return SecretWriteResult{}, err
	}
	secretRef, err := mcpDomainSecret.NewMCPSecretRefString(ref, kind, slot)
	if err != nil {
		return SecretWriteResult{}, err
	}
	hash, nonEmpty, err := s.secrets.SetMCPSecret(ctx, secretRef, value)
	if err != nil {
		return SecretWriteResult{}, err
	}
	s.clearServerAuthStatus(ref)
	return SecretWriteResult{
		SecretRef: secretRef,
		SHA256:    hash,
		NonEmpty:  nonEmpty,
	}, nil
}

func (s *Service) DeleteServerSecret(
	ctx context.Context,
	ref artifact.ArtifactRef,
	kind mcpDomainSecret.MCPSecretKind,
	slot string,
) error {
	if err := s.ready(); err != nil {
		return err
	}
	if kind == mcpDomainSecret.MCPSecretKindOAuthToken {
		return fmt.Errorf(
			"%w: OAuth token secrets are runtime-managed",
			mcpAuth.ErrMCPInvalidAuthRequest,
		)
	}
	installation, err := s.store.GetServerInstallation(ctx, ref)
	if err != nil {
		return err
	}
	if err := installation.Document.AcceptsSecretTarget(kind, slot); err != nil {
		return err
	}
	if err := s.lifecycle.InvalidateServer(ctx, ref); err != nil {
		return err
	}
	secretRef, err := mcpDomainSecret.NewMCPSecretRefString(ref, kind, slot)
	if err != nil {
		return err
	}
	if err := s.secrets.DeleteSecret(ctx, secretRef); err != nil {
		return err
	}
	s.clearServerAuthStatus(ref)
	return nil
}

func (s *Service) GetServerAuthHealth(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpAuth.MCPAuthHealth, error) {
	if err := s.ready(); err != nil {
		return mcpAuth.MCPAuthHealth{}, err
	}

	config, resolved, err := s.source.InspectRuntimeConfig(ctx, ref)
	if err == nil {
		return s.auth.BuildAuthHealth(ctx, config), nil
	}
	if resolved.Server != ref {
		return mcpAuth.MCPAuthHealth{}, err
	}

	serverID, idErr := RuntimeServerIDForArtifact(ref)
	if idErr != nil {
		return mcpAuth.MCPAuthHealth{}, idErr
	}
	m, err := runtimeHTTPAuthMode(resolved.Document.Configuration.Auth.Mode)
	if err != nil {
		return mcpAuth.MCPAuthHealth{}, err
	}
	return mcpAuth.MCPAuthHealth{
		Server:     serverID,
		AuthMode:   m,
		State:      mcpAuth.MCPAuthHealthStateNotConfigured,
		Configured: false,
		LastError:  "required MCP installation input is not configured",
	}, nil
}

func (s *Service) clearServerAuthStatus(ref artifact.ArtifactRef) {
	serverID, err := RuntimeServerIDForArtifact(ref)
	if err == nil {
		s.auth.ClearAuthStatus(serverID)
	}
}

func (s *Service) ready() error {
	if s == nil ||
		s.lifecycle == nil ||
		s.servers == nil ||
		s.source == nil ||
		s.store == nil ||
		s.auth == nil ||
		s.secrets == nil {
		return mcpServer.ErrClosed
	}
	return nil
}
