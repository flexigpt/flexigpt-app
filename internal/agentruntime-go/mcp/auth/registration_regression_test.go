package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	"golang.org/x/oauth2"
)

func TestRegistrationErrorPreservesCause(t *testing.T) {
	cause := errors.New("no configured client registration method is supported")
	err := explainOAuthRegistrationError(fmt.Errorf("authorize: %w", cause))
	if !errors.Is(err, cause) || !errors.Is(err, ErrMCPAuthRequired) {
		t.Fatalf("registration error lost its cause: %v", err)
	}
	if !strings.Contains(err.Error(), "preregistered OAuth client") {
		t.Fatalf("registration error is not actionable: %v", err)
	}

	ordinary := errors.New("connection refused")
	if got := explainOAuthRegistrationError(ordinary); !errors.Is(got, ordinary) {
		t.Fatalf("unrelated error changed: %v", got)
	}
	if got := explainOAuthRegistrationError(nil); got != nil {
		t.Fatalf("nil error changed: %v", got)
	}
}

func TestAuthHealthKeepsRegistrationFailure(t *testing.T) {
	ctx := t.Context()
	manager := NewAuthManager(nil, WithOAuthTokenStore(registrationTestTokenStore{}))
	config := mcpServer.RuntimeConfig{
		Server: "github-test",
		StreamableHTTP: &mcpServer.MCPRuntimeStreamableHTTPConfig{
			URL:      "https://example.com/mcp",
			AuthMode: mcpServer.MCPHTTPAuthOAuth,
		},
	}
	if health := manager.BuildAuthHealth(ctx, config); health.State != MCPAuthHealthStateAuthorized {
		t.Fatalf("expected persisted-token health before an attempt: %+v", health)
	}

	status := defaultAuthStatus(config)
	status.State = MCPAuthStateRequired
	status.LastError = "no configured client registration method is supported"
	if err := manager.SaveAuthStatus(ctx, status); err != nil {
		t.Fatal(err)
	}
	health := manager.BuildAuthHealth(ctx, config)
	if health.LastError != status.LastError ||
		health.State != MCPAuthHealthStateAuthorizationNeeded {
		t.Fatalf("persisted token hid the current auth failure: %+v", health)
	}
}

type registrationTestTokenStore struct{}

func (registrationTestTokenStore) LoadOAuthToken(
	context.Context,
	MCPAuthStatus,
) (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "test-access-token", TokenType: "Bearer"}, nil
}

func (registrationTestTokenStore) SaveOAuthToken(
	context.Context,
	MCPAuthStatus,
	*oauth2.Token,
) error {
	return nil
}

func (registrationTestTokenStore) DeleteOAuthToken(context.Context, MCPAuthStatus) error {
	return nil
}
