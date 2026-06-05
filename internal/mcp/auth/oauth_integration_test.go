package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	testOAuthRedirectURL = "http://127.0.0.1/oauth/callback"
	testClientID         = "dcr-client"
)

type testOAuthBroker struct {
	t *testing.T

	wantClientID    string
	wantRedirectURL string
	wantResourceURL string
	wantOffline     bool
}

func (b *testOAuthBroker) FetchAuthorizationCode(
	ctx context.Context,
	req OAuthAuthorizationRequest,
) (*OAuthAuthorizationResult, error) {
	if req.BundleID != testBundleID {
		b.t.Fatalf("bundleID = %q, want %q", req.BundleID, testBundleID)
	}
	if req.ServerID == "" {
		return nil, errors.New("missing serverID")
	}
	u, err := url.Parse(req.AuthorizationURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()

	if b.wantClientID != "" && q.Get("client_id") != b.wantClientID {
		b.t.Fatalf("client_id = %q, want %q", q.Get("client_id"), b.wantClientID)
	}
	if q.Get("redirect_uri") != b.wantRedirectURL {
		b.t.Fatalf("redirect_uri = %q, want %q", q.Get("redirect_uri"), b.wantRedirectURL)
	}
	if q.Get("resource") != b.wantResourceURL {
		b.t.Fatalf("resource = %q, want %q", q.Get("resource"), b.wantResourceURL)
	}
	if q.Get("state") == "" {
		b.t.Fatalf("missing state")
	}
	if q.Get("code_challenge") == "" {
		b.t.Fatalf("missing PKCE code_challenge")
	}
	if q.Get("code_challenge_method") != "S256" {
		b.t.Fatalf("code_challenge_method = %q, want S256", q.Get("code_challenge_method"))
	}

	scopes := strings.Fields(q.Get("scope"))
	if !containsString(scopes, testScopeMCPTools) {
		b.t.Fatalf("authorization scope %q does not contain %s", q.Get("scope"), testScopeMCPTools)
	}
	if b.wantOffline && !containsString(scopes, "offline_access") {
		b.t.Fatalf("authorization scope %q does not contain offline_access", q.Get("scope"))
	}

	return &OAuthAuthorizationResult{Code: "test-code", State: q.Get("state"), Iss: ""}, nil
}

func TestOAuthAuthorizationCodeFlows(t *testing.T) {
	ctx := t.Context()
	testCIMDURL := "https://client.example.com/flexigpt-mcp-client.json"
	tests := []struct {
		name            string
		resolver        StaticSecretResolver
		ref             string
		cimdURL         string
		cimdSupported   bool
		clients         map[string]string
		wantClientID    string
		wantRegisterCnt int
	}{
		{
			name:            "dcr flow",
			ref:             "",
			wantClientID:    testClientID,
			wantRegisterCnt: 1,
			clients:         map[string]string{testClientID: ""},
		},
		{
			name:            "preregistered public client",
			resolver:        StaticSecretResolver{"public-ref": testPublicRefRaw},
			ref:             "public-ref",
			wantClientID:    testPublicClientID,
			wantRegisterCnt: 0,
			clients:         map[string]string{testPublicClientID: ""},
		},
		{
			name: "preregistered confidential client",
			resolver: StaticSecretResolver{
				"confidential-ref": `{"clientID":"confidential-client","clientSecret":"top-secret"}`,
			},
			ref:             "confidential-ref",
			wantClientID:    testConfidentialClientID,
			wantRegisterCnt: 0,
			clients:         map[string]string{testConfidentialClientID: "top-secret"},
		},
		{
			name:            "client id metadata document supported",
			cimdURL:         testCIMDURL,
			cimdSupported:   true,
			wantClientID:    testCIMDURL,
			wantRegisterCnt: 0,
			clients:         map[string]string{testCIMDURL: ""},
		},
		{
			name:            "client id metadata document falls back to dcr",
			cimdURL:         testCIMDURL,
			cimdSupported:   false,
			wantClientID:    testClientID,
			wantRegisterCnt: 1,
			clients:         map[string]string{testClientID: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newOAuthHarness(t)
			h.cimdSupported = tt.cimdSupported
			maps.Copy(h.clients, tt.clients)

			mgr := NewAuthManager(
				tt.resolver,
				WithOAuthAuthorizationBroker(
					&testOAuthBroker{
						t:               t,
						wantClientID:    tt.wantClientID,
						wantRedirectURL: testOAuthRedirectURL,
						wantResourceURL: h.mcpURL(),
						wantOffline:     true,
					},
				),
				WithOAuthRedirectURL(testOAuthRedirectURL),
				WithAuthHTTPClient(h.client()),
			)

			cfg := streamableHTTPConfig(testBundleID, "oauth-server", h.mcpURL(), spec.MCPHTTPAuthOAuth, tt.ref)
			if tt.cimdURL != "" {
				cfg.StreamableHTTP.ClientIDMetadataDocumentURL = tt.cimdURL
			}

			resolved, err := mgr.PrepareTransportAuth(ctx, cfg)
			if err != nil {
				t.Fatalf("PrepareTransportAuth: %v", err)
			}
			if resolved.OAuthHandler == nil {
				t.Fatalf("OAuthHandler is nil")
			}

			mustAuthorize(t, ctx, h, resolved)

			ts, err := resolved.OAuthHandler.TokenSource(ctx)
			if err != nil {
				t.Fatalf("TokenSource: %v", err)
			}
			if ts == nil {
				t.Fatalf("TokenSource is nil")
			}

			tok, err := ts.Token()
			if err != nil {
				t.Fatalf("Token: %v", err)
			}
			if tok == nil || tok.AccessToken == "" {
				t.Fatalf("empty token: %#v", tok)
			}

			if got := h.registerCallCount(); got != tt.wantRegisterCnt {
				t.Fatalf("register calls = %d, want %d", got, tt.wantRegisterCnt)
			}
			if got := h.tokenCallCount(); got == 0 {
				t.Fatalf("token endpoint was not called")
			}

			st, ok := mgr.GetAuthStatus(testBundleID, cfg.ID)
			if !ok {
				t.Fatalf("missing auth status")
			}
			if st.BundleID != testBundleID || st.ServerID != cfg.ID {
				t.Fatalf("status identifiers = %#v, want bundle/server set", st)
			}
			if st.AuthMode != spec.MCPHTTPAuthOAuth {
				t.Fatalf("AuthMode = %q, want oauth", st.AuthMode)
			}
			if st.State != spec.MCPAuthStateAuthorized {
				t.Fatalf("State = %q, want authorized", st.State)
			}
			if st.Resource != h.mcpURL() {
				t.Fatalf("Resource = %q, want %q", st.Resource, h.mcpURL())
			}
		})
	}
}

func TestOAuthClientCredentialsGrantRefreshesToken(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := t.Context()
	h := newOAuthHarness(t)
	h.clients["service-client"] = "service-secret"
	h.expiresIn = 1

	mgr := NewAuthManager(
		StaticSecretResolver{"service-ref": `{"clientID":"service-client","clientSecret":"service-secret"}`},
		WithAuthHTTPClient(h.client()),
	)

	cfg := streamableHTTPConfig(
		testBundleID,
		"service-server",
		h.mcpURL(),
		spec.MCPHTTPAuthClientCredentials,
		"service-ref",
	)
	resolved, err := mgr.PrepareTransportAuth(ctx, cfg)
	if err != nil {
		t.Fatalf("PrepareTransportAuth: %v", err)
	}
	if resolved.OAuthHandler == nil {
		t.Fatalf("OAuthHandler is nil")
	}

	mustAuthorize(t, ctx, h, resolved)

	ts, err := resolved.OAuthHandler.TokenSource(ctx)
	if err != nil {
		t.Fatalf("TokenSource: %v", err)
	}

	tok1, err := ts.Token()
	if err != nil {
		t.Fatalf("Token #1: %v", err)
	}
	if tok1 == nil || tok1.AccessToken == "" {
		t.Fatalf("empty token #1: %#v", tok1)
	}

	before := h.tokenCallCount()
	time.Sleep(1100 * time.Millisecond)

	tok2, err := ts.Token()
	if err != nil {
		t.Fatalf("Token #2: %v", err)
	}
	if tok2 == nil || tok2.AccessToken == "" {
		t.Fatalf("empty token #2: %#v", tok2)
	}

	if got := h.tokenCallCount() - before; got < 1 {
		t.Fatalf("token refresh calls after expiry = %d, want at least 1", got)
	}

	st, ok := mgr.GetAuthStatus(testBundleID, cfg.ID)
	if !ok {
		t.Fatalf("missing auth status")
	}
	if st.State != spec.MCPAuthStateAuthorized {
		t.Fatalf("State = %q, want authorized", st.State)
	}
}

func TestOAuthClientCredentialsRefreshErrorRedactsStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := t.Context()
	h := newOAuthHarness(t)
	h.clients["service-client"] = "service-secret"
	h.expiresIn = 1
	h.failTokenAt = 2
	h.failTokenCode = errStrInvalidGrant
	h.failTokenDescription = "refresh token expired"

	mgr := NewAuthManager(
		StaticSecretResolver{"service-ref": `{"clientID":"service-client","clientSecret":"service-secret"}`},
		WithAuthHTTPClient(h.client()),
	)

	cfg := streamableHTTPConfig(
		testBundleID,
		"service-server",
		h.mcpURL(),
		spec.MCPHTTPAuthClientCredentials,
		"service-ref",
	)
	resolved, err := mgr.PrepareTransportAuth(ctx, cfg)
	if err != nil {
		t.Fatalf("PrepareTransportAuth: %v", err)
	}
	if resolved.OAuthHandler == nil {
		t.Fatalf("OAuthHandler is nil")
	}

	mustAuthorize(t, ctx, h, resolved)

	ts, err := resolved.OAuthHandler.TokenSource(ctx)
	if err != nil {
		t.Fatalf("TokenSource: %v", err)
	}

	if tok, err := ts.Token(); err == nil {
		if tok == nil || tok.AccessToken == "" {
			t.Fatalf("empty token #1: %#v", tok)
		}

		time.Sleep(1100 * time.Millisecond)

		if _, err := ts.Token(); err == nil {
			t.Fatalf("Token #2 succeeded, want invalid_grant")
		}
	}

	st, ok := mgr.GetAuthStatus(testBundleID, cfg.ID)
	if !ok {
		t.Fatalf("missing auth status")
	}
	if st.State != spec.MCPAuthStateError {
		t.Fatalf("State = %q, want error", st.State)
	}
	if strings.Contains(st.LastError, "top-secret") {
		t.Fatalf("LastError leaked secret: %q", st.LastError)
	}
}

func mustAuthorize(t *testing.T, ctx context.Context, h *oauthHarness, resolved ResolvedTransportAuth) {
	t.Helper()

	if resolved.OAuthHandler == nil {
		t.Fatalf("OAuthHandler is nil")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.mcpURL(), http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Header:     http.Header{"WWW-Authenticate": []string{h.bearerChallenge()}},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	if err := resolved.OAuthHandler.Authorize(ctx, req, resp); err != nil {
		t.Fatalf("Authorize: %v", err)
	}
}

func streamableHTTPConfig(
	bundleID bundleitemutils.BundleID,
	id, urlStr string,
	mode spec.MCPHTTPAuthMode,
	ref string,
) spec.MCPServerConfig {
	return spec.MCPServerConfig{
		BundleID:       bundleID,
		ID:             spec.MCPServerID(id),
		Enabled:        true,
		Transport:      spec.MCPTransportStreamableHTTP,
		StreamableHTTP: &spec.MCPStreamableHTTPConfig{URL: urlStr, AuthMode: mode, ClientCredentialRef: ref},
	}
}

func tokenClientAuth(r *http.Request) (clientID, clientSecret string) {
	if id, secret, ok := r.BasicAuth(); ok {
		return id, secret
	}
	return r.Form.Get("client_id"), r.Form.Get("client_secret")
}

func cloneValues(in url.Values) url.Values {
	out := make(url.Values, len(in))
	for k, values := range in {
		out[k] = append([]string(nil), values...)
	}
	return out
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func containsString(ss []string, want string) bool {
	return slices.Contains(ss, want)
}
