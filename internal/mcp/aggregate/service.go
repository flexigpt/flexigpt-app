package aggregate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
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
	Store     mcpConsumerAPI.ManagementStore
	Auth      AuthState
	Secrets   SecretStore
}

type Service struct {
	lifecycle *Lifecycle
	servers   *ArtifactServerResolver
	source    *RuntimeServerSource
	store     mcpConsumerAPI.ManagementStore
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
		case mcpv1.HTTPAuthModeOAuth, mcpv1.HTTPAuthModeClientCredentials:
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

func (s *Service) GetMCPEffectivePolicy(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpPolicy.Effective, error) {
	if err := s.ready(); err != nil {
		return mcpPolicy.Effective{}, err
	}
	return s.store.GetMCPEffectivePolicy(ctx, ref)
}

func (s *Service) CreateManagedMCP(
	ctx context.Context,
	request mcpConsumerAPI.ManagedMCPCreateRequest,
) (mcpConsumerAPI.ManagedMCPCreateResult, error) {
	if err := s.ready(); err != nil {
		return mcpConsumerAPI.ManagedMCPCreateResult{}, err
	}

	result, err := s.store.CreateManagedMCP(ctx, request)
	if err != nil {
		return mcpConsumerAPI.ManagedMCPCreateResult{}, err
	}
	if err := s.lifecycle.InvalidateServer(ctx, result.Artifact.Ref()); err != nil {
		return mcpConsumerAPI.ManagedMCPCreateResult{}, err
	}
	s.clearServerAuthStatus(result.Artifact.Ref())
	return result, nil
}

func (s *Service) ReplaceManagedMCP(
	ctx context.Context,
	request mcpConsumerAPI.ManagedMCPReplaceRequest,
) (mcpConsumerAPI.ManagedMCPReplaceResult, error) {
	if err := s.ready(); err != nil {
		return mcpConsumerAPI.ManagedMCPReplaceResult{}, err
	}
	if err := s.lifecycle.InvalidateServer(ctx, request.Artifact); err != nil {
		return mcpConsumerAPI.ManagedMCPReplaceResult{}, err
	}
	s.clearServerAuthStatus(request.Artifact)
	return s.store.ReplaceManagedMCP(ctx, request)
}

func (s *Service) PurgeManagedMCP(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := s.lifecycle.InvalidateServer(ctx, ref); err != nil {
		return err
	}
	s.clearServerAuthStatus(ref)
	return s.store.PurgeManagedMCP(ctx, ref, expectedRevision)
}

func (s *Service) UpsertManagedMCPPolicy(
	ctx context.Context,
	request mcpConsumerAPI.ManagedMCPPolicyUpsertRequest,
) (mcpConsumerAPI.ManagedMCPPolicyUpsertResult, error) {
	if err := s.ready(); err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}
	if err := request.Collection.Validate(); err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}
	if err := request.Name.Validate(); err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}

	affected, err := s.store.ListMCPServersReferencingPolicy(
		ctx,
		request.Collection.RootID,
		request.Name,
	)
	if err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}
	if err := s.lifecycle.InvalidateServers(ctx, affected); err != nil {
		return mcpConsumerAPI.ManagedMCPPolicyUpsertResult{}, err
	}
	return s.store.UpsertManagedMCPPolicy(ctx, request)
}

func (s *Service) PurgeManagedMCPPolicy(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if err := s.ready(); err != nil {
		return err
	}

	policy, err := s.store.GetMCPPolicy(ctx, ref)
	if err != nil {
		return err
	}
	affected, err := s.store.ListMCPServersReferencingPolicy(
		ctx,
		policy.Artifact.RootID,
		policy.Artifact.LogicalName,
	)
	if err != nil {
		return err
	}
	if err := s.lifecycle.InvalidateServers(ctx, affected); err != nil {
		return err
	}
	return s.store.PurgeManagedMCPPolicy(ctx, ref, expectedRevision)
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
