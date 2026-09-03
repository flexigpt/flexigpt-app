package aggregate

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpStore "github.com/flexigpt/flexigpt-app/internal/mcp/store"
	mcpSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/secret"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
)

type BundleServerStore interface {
	ListServers(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Artifact, error)

	GetServerInstallation(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (mcpStore.ServerInstallationView, error)
}

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
	Bundles   BundleServerStore
	Auth      AuthState
	Secrets   SecretStore
}

type Service struct {
	lifecycle *Lifecycle
	servers   *ArtifactServerResolver
	source    *RuntimeServerSource
	bundles   BundleServerStore
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
		dependencies.Bundles == nil ||
		dependencies.Auth == nil ||
		dependencies.Secrets == nil {
		return nil, errors.New("MCP aggregate dependencies are incomplete")
	}

	return &Service{
		lifecycle: dependencies.Lifecycle,
		servers:   dependencies.Servers,
		source:    dependencies.Source,
		bundles:   dependencies.Bundles,
		auth:      dependencies.Auth,
		secrets:   dependencies.Secrets,
	}, nil
}

func (s *Service) InspectRuntimeConfig(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpServer.RuntimeConfig, mcpStoreServer.Resolved, error) {
	if err := s.ready(); err != nil {
		return mcpServer.RuntimeConfig{}, mcpStoreServer.Resolved{}, err
	}
	return s.source.InspectRuntimeConfig(ctx, ref)
}

func (s *Service) ReplaceDocument(
	ctx context.Context,
	request mcpStore.ReplaceDocumentRequest,
) (mcpStore.Bundle, error) {
	ids, err := s.serverIDsForBundle(ctx, request.Bundle)
	if err != nil {
		return mcpStore.Bundle{}, err
	}
	value, err := s.lifecycle.ReplaceDocument(ctx, request)
	if err != nil {
		return mcpStore.Bundle{}, err
	}
	s.clearAuthStatuses(ids)
	return value, nil
}

func (s *Service) RefreshBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	allowProtected bool,
) (mcpStore.Bundle, error) {
	ids, err := s.serverIDsForBundle(ctx, ref)
	if err != nil {
		return mcpStore.Bundle{}, err
	}
	value, err := s.lifecycle.RefreshBundle(ctx, ref, allowProtected)
	if err != nil {
		return mcpStore.Bundle{}, err
	}
	s.clearAuthStatuses(ids)
	return value, nil
}

func (s *Service) UpdateBundleEnabled(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
	enabled bool,
) (mcpStore.Bundle, error) {
	ids, err := s.serverIDsForBundle(ctx, ref)
	if err != nil {
		return mcpStore.Bundle{}, err
	}
	value, err := s.lifecycle.UpdateBundleEnabled(ctx, ref, expectedRevision, enabled)
	if err != nil {
		return mcpStore.Bundle{}, err
	}
	s.clearAuthStatuses(ids)
	return value, nil
}

func (s *Service) RetireBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	ids, err := s.serverIDsForBundle(ctx, ref)
	if err != nil {
		return collection.Collection{}, err
	}
	value, err := s.lifecycle.RetireBundle(ctx, ref, expectedRevision)
	if err != nil {
		return collection.Collection{}, err
	}
	s.clearAuthStatuses(ids)
	return value, nil
}

func (s *Service) PurgeBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	if err := s.ready(); err != nil {
		return err
	}
	return s.lifecycle.PurgeBundle(ctx, ref, expectedRevision)
}

func (s *Service) UpdateProtectedBundleInstallation(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
) error {
	ids, err := s.serverIDsForBundle(ctx, ref)
	if err != nil {
		return err
	}
	if err := s.lifecycle.UpdateProtectedBundleInstallation(
		ctx,
		ref,
		expectedOverlayRevision,
		runtimeEnabled,
	); err != nil {
		return err
	}
	s.clearAuthStatuses(ids)
	return nil
}

func (s *Service) UpdateServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpStoreServer.ServerData,
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
	runtimeEnabled bool,
	data mcpStoreServer.ServerData,
) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := s.lifecycle.UpdateProtectedServerInstallation(
		ctx,
		ref,
		expectedOverlayRevision,
		runtimeEnabled,
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
	kind mcpSecret.MCPSecretKind,
	slot string,
	value string,
) (SecretWriteResult, error) {
	if err := s.ready(); err != nil {
		return SecretWriteResult{}, err
	}
	if kind == mcpSecret.MCPSecretKindOAuthToken {
		return SecretWriteResult{}, fmt.Errorf(
			"%w: OAuth token secrets are runtime-managed",
			mcpAuth.ErrMCPInvalidAuthRequest,
		)
	}

	installation, err := s.bundles.GetServerInstallation(ctx, ref)
	if err != nil {
		return SecretWriteResult{}, err
	}
	if err := validateSecretTarget(installation.Document, kind, slot); err != nil {
		return SecretWriteResult{}, err
	}

	if kind == mcpSecret.MCPSecretKindOAuthClientCredentials {
		switch installation.Document.Extension.Auth.Mode {
		case mcpServer.MCPHTTPAuthNone, mcpServer.MCPHTTPAuthClientCredentials:
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

	if kind == mcpSecret.MCPSecretKindHTTPHeader &&
		(strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n\x00")) {
		return SecretWriteResult{}, fmt.Errorf(
			"%w: invalid HTTP header secret value",
			mcpAuth.ErrMCPInvalidAuthRequest,
		)
	}

	if err := s.lifecycle.InvalidateServer(ctx, ref); err != nil {
		return SecretWriteResult{}, err
	}
	secretRef, err := mcpSecret.NewMCPSecretRefString(ref, kind, slot)
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
	kind mcpSecret.MCPSecretKind,
	slot string,
) error {
	if err := s.ready(); err != nil {
		return err
	}
	if kind == mcpSecret.MCPSecretKindOAuthToken {
		return fmt.Errorf(
			"%w: OAuth token secrets are runtime-managed",
			mcpAuth.ErrMCPInvalidAuthRequest,
		)
	}
	if _, err := s.bundles.GetServerInstallation(ctx, ref); err != nil {
		return err
	}
	if err := s.lifecycle.InvalidateServer(ctx, ref); err != nil {
		return err
	}
	secretRef, err := mcpSecret.NewMCPSecretRefString(ref, kind, slot)
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
	return mcpAuth.MCPAuthHealth{
		Server:     serverID,
		AuthMode:   resolved.Document.Extension.Auth.Mode,
		State:      mcpAuth.MCPAuthHealthStateNotConfigured,
		Configured: false,
		LastError:  "required MCP installation input is not configured",
	}, nil
}

func (s *Service) serverIDsForBundle(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]mcpServer.ServerID, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	records, err := s.bundles.ListServers(ctx, ref)
	if err != nil {
		return nil, err
	}
	seen := make(map[mcpServer.ServerID]struct{}, len(records))
	for _, record := range records {
		serverID, err := RuntimeServerIDForArtifact(record.Ref())
		if err != nil {
			return nil, err
		}
		seen[serverID] = struct{}{}
	}

	output := make([]mcpServer.ServerID, 0, len(seen))
	for serverID := range seen {
		output = append(output, serverID)
	}
	slices.Sort(output)
	return output, nil
}

func (s *Service) clearAuthStatuses(ids []mcpServer.ServerID) {
	for _, id := range ids {
		s.auth.ClearAuthStatus(id)
	}
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
		s.bundles == nil ||
		s.auth == nil ||
		s.secrets == nil {
		return mcpServer.ErrClosed
	}
	return nil
}

func validateSecretTarget(
	document mcpStoreServer.ServerDocument,
	kind mcpSecret.MCPSecretKind,
	slot string,
) error {
	switch kind {
	case mcpSecret.MCPSecretKindOAuthClientCredentials:
		input := document.Extension.Auth.ClientCredentialsInput
		declaration, found := document.Extension.Install.Inputs[input]
		if input == "" ||
			!found ||
			declaration.Kind != mcpStoreServer.InputOAuthClientCredentials ||
			!strings.EqualFold(strings.TrimSpace(slot), "clientCredentials") {
			return fmt.Errorf(
				"%w: invalid OAuth client credentials secret target",
				mcpAuth.ErrMCPInvalidAuthRequest,
			)
		}
		return nil

	case mcpSecret.MCPSecretKindStdioEnv, mcpSecret.MCPSecretKindHTTPHeader:
		targets, err := mcpStoreServer.SecretInputTargets(document)
		if err != nil {
			return err
		}
		for _, target := range targets {
			expectedKind := mcpSecret.MCPSecretKindStdioEnv
			if target.Kind == mcpStoreServer.SecretInputTargetHTTPHeader {
				expectedKind = mcpSecret.MCPSecretKindHTTPHeader
			}
			if kind == expectedKind &&
				strings.EqualFold(target.Slot, strings.TrimSpace(slot)) {
				return nil
			}
		}
		return fmt.Errorf(
			"%w: secret target is not declared by the MCP server",
			mcpAuth.ErrMCPInvalidAuthRequest,
		)

	default:
		return fmt.Errorf(
			"%w: unsupported MCP secret kind %q",
			mcpAuth.ErrMCPInvalidAuthRequest,
			kind,
		)
	}
}
