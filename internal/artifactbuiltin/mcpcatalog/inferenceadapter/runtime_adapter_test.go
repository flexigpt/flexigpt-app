package inferenceadapter

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	mcpAuth "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/auth"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/secret"
	"golang.org/x/oauth2"
)

func testArtifactRef() artifactModel.ArtifactRef {
	return artifactModel.ArtifactRef{
		RootID:     "0192c4c0-0000-7000-8000-000000000002",
		ArtifactID: "0192c4c0-0002-7000-8000-000000000001",
	}
}

func TestServerIdentityPreservesFormat(t *testing.T) {
	ref := testArtifactRef()
	id, err := ServerIDForArtifact(ref)
	if err != nil {
		t.Fatal(err)
	}

	expected := mcpServer.ServerID(
		"artifact-server:v1:" +
			base64.RawURLEncoding.EncodeToString(
				[]byte(string(ref.RootID)+"\x00"+string(ref.ArtifactID)),
			),
	)
	if id != expected {
		t.Fatalf("runtime identity changed: got %q, want %q", id, expected)
	}

	decoded, err := ArtifactRefForServerID(id)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != ref {
		t.Fatalf("identity round trip: got %#v, want %#v", decoded, ref)
	}

	catalog, err := runtimeCatalogIDForRoot(ref.RootID)
	if err != nil {
		t.Fatal(err)
	}
	expectedCatalog := mcpServer.CatalogID(
		"artifact-root:v1:" +
			base64.RawURLEncoding.EncodeToString([]byte(ref.RootID)),
	)
	if catalog != expectedCatalog {
		t.Fatalf("catalog identity changed: got %q, want %q", catalog, expectedCatalog)
	}
}

func TestServerIdentityRejectsMalformedValues(t *testing.T) {
	values := []mcpServer.ServerID{
		"",
		"other-server:v1:value",
		"artifact-server:v1:!",
		mcpServer.ServerID(
			"artifact-server:v1:" +
				base64.RawURLEncoding.EncodeToString([]byte("missing-separator")),
		),
		mcpServer.ServerID(
			"artifact-server:v1:" +
				base64.RawURLEncoding.EncodeToString([]byte("\x00")),
		),
	}
	for _, value := range values {
		if _, err := ArtifactRefForServerID(value); err == nil {
			t.Errorf("accepted malformed runtime identity %q", value)
		}
	}

	if _, err := ServerIDForArtifact(artifactModel.ArtifactRef{}); err == nil {
		t.Fatal("accepted an empty Artifact reference")
	}
}

func TestOAuthTokenPersistenceUsesArtifactScopedReference(t *testing.T) {
	ctx := t.Context()
	ref := testArtifactRef()
	server, err := ServerIDForArtifact(ref)
	if err != nil {
		t.Fatal(err)
	}
	expectedSecretRef, err := secret.NewMCPSecretRefString(
		ref,
		secret.MCPSecretKindOAuthToken,
		"token",
	)
	if err != nil {
		t.Fatal(err)
	}

	secrets := &memoryMCPSecrets{values: map[string]string{}}
	adapter := &RuntimeAdapter{secrets: secrets}
	status := mcpAuth.MCPAuthStatus{Server: server}

	if _, err := adapter.LoadOAuthToken(ctx, status); !errors.Is(
		err,
		mcpAuth.ErrOAuthTokenNotFound,
	) {
		t.Fatalf("missing token error: %v", err)
	}

	if err := adapter.SaveOAuthToken(ctx, status, nil); err != nil {
		t.Fatal(err)
	}
	if err := adapter.SaveOAuthToken(ctx, status, &oauth2.Token{
		AccessToken: "expired-test-token",
		Expiry:      time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if len(secrets.values) != 0 {
		t.Fatal("nil or invalid token was persisted")
	}

	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(time.Hour).UTC(),
	}
	if err := adapter.SaveOAuthToken(ctx, status, token); err != nil {
		t.Fatal(err)
	}
	if _, found := secrets.values[expectedSecretRef]; !found {
		t.Fatal("OAuth token was not stored at the existing Artifact-scoped reference")
	}

	loaded, err := adapter.LoadOAuthToken(ctx, status)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AccessToken != token.AccessToken ||
		loaded.TokenType != token.TokenType ||
		loaded.RefreshToken != token.RefreshToken ||
		!loaded.Expiry.Equal(token.Expiry) {
		t.Fatal("OAuth token did not round trip")
	}

	if err := adapter.DeleteOAuthToken(ctx, status); err != nil {
		t.Fatal(err)
	}
	if err := adapter.DeleteOAuthToken(ctx, status); err != nil {
		t.Fatalf("repeated token deletion: %v", err)
	}
	if _, err := adapter.LoadOAuthToken(ctx, status); !errors.Is(
		err,
		mcpAuth.ErrOAuthTokenNotFound,
	) {
		t.Fatalf("deleted token error: %v", err)
	}
}

func TestOAuthTokenLoadPreservesPersistenceError(t *testing.T) {
	server, err := ServerIDForArtifact(testArtifactRef())
	if err != nil {
		t.Fatal(err)
	}

	cause := errors.New("secret persistence unavailable")
	adapter := &RuntimeAdapter{
		secrets: &memoryMCPSecrets{readErr: cause},
	}
	_, err = adapter.LoadOAuthToken(
		t.Context(),
		mcpAuth.MCPAuthStatus{Server: server},
	)
	if !errors.Is(err, cause) {
		t.Fatalf("persistence error was lost: %v", err)
	}
	if errors.Is(err, mcpAuth.ErrOAuthTokenNotFound) {
		t.Fatal("persistence failure was incorrectly reported as a missing token")
	}
}

type memoryMCPSecrets struct {
	values  map[string]string
	readErr error
}

func (s *memoryMCPSecrets) ResolveSecret(
	_ context.Context,
	ref string,
) (string, error) {
	if s.readErr != nil {
		return "", s.readErr
	}
	value, found := s.values[ref]
	if !found {
		return "", secret.ErrNotFound
	}
	return value, nil
}

func (s *memoryMCPSecrets) SetMCPSecret(
	_ context.Context,
	ref string,
	value string,
) (a string, v bool, err error) {
	s.values[ref] = value
	return "", value != "", nil
}

func (s *memoryMCPSecrets) DeleteSecret(
	_ context.Context,
	ref string,
) error {
	delete(s.values, ref)
	return nil
}
