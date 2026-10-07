package mcpruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	mcpAuth "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/auth"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/secret"
	"golang.org/x/oauth2"
)

// LoadOAuthToken translates the opaque runtime identity before using the
// application's Artifact-scoped secret persistence capability.
func (a *RuntimeAdapter) LoadOAuthToken(
	ctx context.Context,
	status mcpAuth.MCPAuthStatus,
) (*oauth2.Token, error) {
	ref, err := oauthTokenSecretRef(status.Server)
	if err != nil {
		return nil, err
	}
	raw, err := a.secrets.ResolveSecret(ctx, ref)
	if err != nil {
		if errors.Is(err, secret.ErrNotFound) {
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

func (a *RuntimeAdapter) SaveOAuthToken(
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

	//nolint:gosec // Access token is written only through secret persistence.
	raw, err := json.Marshal(token)
	if err != nil {
		return err
	}
	_, _, err = a.oauthTokens.SetMCPSecret(ctx, ref, string(raw))
	return err
}

func (a *RuntimeAdapter) DeleteOAuthToken(
	ctx context.Context,
	status mcpAuth.MCPAuthStatus,
) error {
	ref, err := oauthTokenSecretRef(status.Server)
	if err != nil {
		return err
	}
	return a.oauthTokens.DeleteSecret(ctx, ref)
}

func oauthTokenSecretRef(serverID mcpServer.ServerID) (string, error) {
	ref, err := ArtifactRefForServerID(serverID)
	if err != nil {
		return "", err
	}
	return secret.NewMCPSecretRefString(
		ref,
		secret.MCPSecretKindOAuthToken,
		"token",
	)
}
