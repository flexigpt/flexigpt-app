package aggregate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
	mcpSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/secret"
	"golang.org/x/oauth2"
)

// OAuthTokenStore translates runtime-owned opaque ServerID values at the
// Aggregate boundary before using artifact-scoped secret references.
type OAuthTokenStore struct {
	secrets SecretStore
}

func NewOAuthTokenStore(secrets SecretStore) (*OAuthTokenStore, error) {
	if secrets == nil {
		return nil, errors.New("MCP OAuth token secret store is required")
	}
	return &OAuthTokenStore{secrets: secrets}, nil
}

func (s *OAuthTokenStore) LoadOAuthToken(
	ctx context.Context,
	status mcpAuth.MCPAuthStatus,
) (*oauth2.Token, error) {
	ref, err := oauthTokenSecretRef(status.Server)
	if err != nil {
		return nil, err
	}
	raw, err := s.secrets.ResolveSecret(ctx, ref)
	if err != nil {
		if errors.Is(err, mcpSecret.ErrNotFound) {
			return nil, mcpAuth.ErrOAuthTokenNotFound
		}
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(raw), &token); err != nil {
		return nil, fmt.Errorf("decode persisted MCP OAuth token: %w", err)
	}
	return &token, nil
}

func (s *OAuthTokenStore) SaveOAuthToken(
	ctx context.Context,
	status mcpAuth.MCPAuthStatus,
	token *oauth2.Token,
) error {
	if token == nil || !token.Valid() {
		return nil
	}
	ref, err := oauthTokenSecretRef(status.Server)
	if err != nil {
		return err
	}
	//nolint:gosec // Access token.
	raw, err := json.Marshal(token)
	if err != nil {
		return err
	}
	_, _, err = s.secrets.SetMCPSecret(ctx, ref, string(raw))
	return err
}

func (s *OAuthTokenStore) DeleteOAuthToken(
	ctx context.Context,
	status mcpAuth.MCPAuthStatus,
) error {
	ref, err := oauthTokenSecretRef(status.Server)
	if err != nil {
		return err
	}
	return s.secrets.DeleteSecret(ctx, ref)
}

func oauthTokenSecretRef(serverID mcpSpec.ServerID) (string, error) {
	ref, err := ArtifactRefForRuntimeServerID(serverID)
	if err != nil {
		return "", err
	}
	return mcpSecret.NewMCPSecretRefString(
		ref,
		mcpSecret.MCPSecretKindOAuthToken,
		"token",
	)
}

var _ mcpAuth.OAuthTokenStore = (*OAuthTokenStore)(nil)
