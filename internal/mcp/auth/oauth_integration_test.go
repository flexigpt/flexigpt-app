package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	testOAuthRedirectURL = "http://127.0.0.1/oauth/callback"
	testClientID         = "dcr-client"
)

type oauthHarness struct {
	t *testing.T

	server *httptest.Server

	mu sync.Mutex

	registerCalls int
	tokenCalls    int

	lastRegister  map[string]any
	lastTokenForm url.Values
	lastTokenAuth string

	clients map[string]string

	cimdSupported bool
	issSupported  bool
	expiresIn     int
}

func newOAuthHarness(t *testing.T) *oauthHarness {
	t.Helper()

	h := &oauthHarness{
		t:       t,
		clients: map[string]string{},
	}
	h.server = httptest.NewServer(http.HandlerFunc(h.serveHTTP))
	t.Cleanup(h.server.Close)

	return h
}

func (h *oauthHarness) mcpURL() string {
	return h.server.URL + "/mcp"
}

func (h *oauthHarness) prmURL() string {
	return h.server.URL + "/.well-known/oauth-protected-resource/mcp"
}

func (h *oauthHarness) issuerURL() string {
	return h.server.URL
}

func (h *oauthHarness) client() *http.Client {
	return h.server.Client()
}

func (h *oauthHarness) bearerChallenge() string {
	return fmt.Sprintf(`Bearer resource_metadata="%q", scope="mcp:tools"`, h.prmURL())
}

func (h *oauthHarness) registerCallCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.registerCalls
}

func (h *oauthHarness) tokenCallCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.tokenCalls
}

func (h *oauthHarness) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/.well-known/oauth-protected-resource/mcp",
		"/.well-known/oauth-protected-resource":
		h.serveProtectedResourceMetadata(w, r)

	case "/.well-known/oauth-authorization-server",
		"/.well-known/openid-configuration":
		h.serveAuthorizationServerMetadata(w, r)

	case "/register":
		h.serveRegister(w, r)

	case "/token":
		h.serveToken(w, r)

	case "/authorize":
		http.Error(w, "authorization endpoint is handled by test broker", http.StatusBadRequest)

	default:
		http.NotFound(w, r)
	}
}

func (h *oauthHarness) serveProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"resource":              h.mcpURL(),
		"authorization_servers": []string{h.issuerURL()},
		"scopes_supported":      []string{"mcp:tools"},
	})
}

func (h *oauthHarness) serveAuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"issuer":                   h.issuerURL(),
		"authorization_endpoint":   h.issuerURL() + "/authorize",
		"token_endpoint":           h.issuerURL() + "/token",
		"registration_endpoint":    h.issuerURL() + "/register",
		"response_types_supported": []string{"code"},
		"grant_types_supported": []string{
			string(spec.GrantTypeAuthorizationCode),
			string(spec.GrantTypeRefreshToken),
			"client_credentials",
		},
		"token_endpoint_auth_methods_supported":          []string{"client_secret_post", "client_secret_basic", "none"},
		"code_challenge_methods_supported":               []string{"S256"},
		"scopes_supported":                               []string{"mcp:tools", "offline_access"},
		"client_id_metadata_document_supported":          h.cimdSupported,
		"authorization_response_iss_parameter_supported": h.issSupported,
	})
}

func (h *oauthHarness) serveRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	h.registerCalls++
	h.lastRegister = body
	h.clients[testClientID] = ""
	h.mu.Unlock()

	writeJSON(w, map[string]any{
		"client_id":                  testClientID,
		"redirect_uris":              body["redirect_uris"],
		"token_endpoint_auth_method": "none",
		"grant_types":                body["grant_types"],
		"response_types":             body["response_types"],
		"client_name":                body["client_name"],
	})
}

func (h *oauthHarness) serveToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	clientID, clientSecret := tokenClientAuth(r)
	grantType := r.Form.Get("grant_type")

	h.mu.Lock()
	h.tokenCalls++
	seq := h.tokenCalls
	h.lastTokenForm = cloneValues(r.Form)
	h.lastTokenAuth = clientID
	expectedSecret, knownClient := h.clients[clientID]
	expiresIn := h.expiresIn
	h.mu.Unlock()

	if expiresIn <= 0 {
		expiresIn = 3600
	}

	if !knownClient {
		http.Error(w, "unknown client", http.StatusUnauthorized)
		return
	}
	if expectedSecret != "" && clientSecret != expectedSecret {
		http.Error(w, "bad client secret", http.StatusUnauthorized)
		return
	}

	switch grantType {
	case "authorization_code":
		if r.Form.Get("code") != "test-code" {
			http.Error(w, "bad code", http.StatusBadRequest)
			return
		}
		if r.Form.Get("code_verifier") == "" {
			http.Error(w, "missing code verifier", http.StatusBadRequest)
			return
		}
		if r.Form.Get("resource") != h.mcpURL() {
			http.Error(w, "bad resource", http.StatusBadRequest)
			return
		}

	case "refresh_token":
		if r.Form.Get("refresh_token") == "" {
			http.Error(w, "missing refresh token", http.StatusBadRequest)
			return
		}

	case "client_credentials":
		if expectedSecret == "" {
			http.Error(w, "client_credentials requires confidential client", http.StatusUnauthorized)
			return
		}

	default:
		http.Error(w, "unsupported grant type", http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]any{
		"access_token":  fmt.Sprintf("access-%d", seq),
		"refresh_token": fmt.Sprintf("refresh-%d", seq),
		"token_type":    "Bearer",
		"expires_in":    expiresIn,
		"scope":         "mcp:tools offline_access",
	})
}

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
	if req.ServerID == "" {
		return nil, errors.New("missing serverID")
	}
	u, err := url.Parse(req.AuthorizationURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()

	if q.Get("client_id") != b.wantClientID {
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
	if !slices.Contains(scopes, "mcp:tools") {
		b.t.Fatalf("authorization scope %q does not contain mcp:tools", q.Get("scope"))
	}
	if b.wantOffline && !slices.Contains(scopes, "offline_access") {
		b.t.Fatalf("authorization scope %q does not contain offline_access", q.Get("scope"))
	}

	return &OAuthAuthorizationResult{
		Code:  "test-code",
		State: q.Get("state"),
		Iss:   "",
	}, nil
}

func TestOAuthAuthorizationCodeDCRFlow(t *testing.T) {
	ctx := t.Context()
	h := newOAuthHarness(t)

	mgr := NewAuthManager(
		StaticSecretResolver{},
		WithOAuthAuthorizationBroker(&testOAuthBroker{
			t:               t,
			wantClientID:    testClientID,
			wantRedirectURL: testOAuthRedirectURL,
			wantResourceURL: h.mcpURL(),
			wantOffline:     true,
		}),
		WithOAuthRedirectURL(testOAuthRedirectURL),
		WithAuthHTTPClient(h.client()),
	)

	cfg := streamableHTTPConfig("dcr", h.mcpURL(), spec.MCPHTTPAuthOAuth, "")
	resolved, err := mgr.PrepareTransportAuth(ctx, cfg)
	if err != nil {
		t.Fatalf("PrepareTransportAuth: %v", err)
	}
	authorizeWithChallenge(t, h, resolved)

	if got := h.registerCallCount(); got != 1 {
		t.Fatalf("register calls = %d, want 1", got)
	}

	st, ok := mgr.GetAuthStatus(cfg.ID)
	if !ok {
		t.Fatalf("missing auth status")
	}
	if st.State != spec.MCPAuthStateAuthorized {
		t.Fatalf("state = %q, want authorized, lastError=%q", st.State, st.LastError)
	}
}

func TestOAuthAuthorizationCodePreregisteredPublicClient(t *testing.T) {
	ctx := t.Context()
	h := newOAuthHarness(t)
	h.clients["public-client"] = ""

	const ref = "public-ref"

	mgr := NewAuthManager(
		StaticSecretResolver{
			ref: `{"clientID":"public-client"}`,
		},
		WithOAuthAuthorizationBroker(&testOAuthBroker{
			t:               t,
			wantClientID:    "public-client",
			wantRedirectURL: testOAuthRedirectURL,
			wantResourceURL: h.mcpURL(),
			wantOffline:     true,
		}),
		WithOAuthRedirectURL(testOAuthRedirectURL),
		WithAuthHTTPClient(h.client()),
	)

	cfg := streamableHTTPConfig("public", h.mcpURL(), spec.MCPHTTPAuthOAuth, ref)
	resolved, err := mgr.PrepareTransportAuth(ctx, cfg)
	if err != nil {
		t.Fatalf("PrepareTransportAuth: %v", err)
	}
	authorizeWithChallenge(t, h, resolved)

	if got := h.registerCallCount(); got != 0 {
		t.Fatalf("register calls = %d, want 0", got)
	}
}

func TestOAuthAuthorizationCodePreregisteredConfidentialClient(t *testing.T) {
	ctx := t.Context()
	h := newOAuthHarness(t)
	h.clients["confidential-client"] = "top-secret"

	const ref = "confidential-ref"

	mgr := NewAuthManager(
		StaticSecretResolver{
			ref: `{"clientID":"confidential-client","clientSecret":"top-secret"}`,
		},
		WithOAuthAuthorizationBroker(&testOAuthBroker{
			t:               t,
			wantClientID:    "confidential-client",
			wantRedirectURL: testOAuthRedirectURL,
			wantResourceURL: h.mcpURL(),
			wantOffline:     true,
		}),
		WithOAuthRedirectURL(testOAuthRedirectURL),
		WithAuthHTTPClient(h.client()),
	)

	cfg := streamableHTTPConfig("confidential", h.mcpURL(), spec.MCPHTTPAuthOAuth, ref)
	resolved, err := mgr.PrepareTransportAuth(ctx, cfg)
	if err != nil {
		t.Fatalf("PrepareTransportAuth: %v", err)
	}
	authorizeWithChallenge(t, h, resolved)

	h.mu.Lock()
	lastForm := cloneValues(h.lastTokenForm)
	h.mu.Unlock()

	if got := lastForm.Get("client_secret"); got != "top-secret" {
		t.Fatalf("client_secret form value = %q, want top-secret", got)
	}
}

func TestOAuthClientIDMetadataDocumentSupported(t *testing.T) {
	ctx := t.Context()
	h := newOAuthHarness(t)
	h.cimdSupported = true

	const clientIDURL = "https://client.example.com/flexigpt-mcp-client.json"
	h.clients[clientIDURL] = ""

	mgr := NewAuthManager(
		StaticSecretResolver{},
		WithOAuthAuthorizationBroker(&testOAuthBroker{
			t:               t,
			wantClientID:    clientIDURL,
			wantRedirectURL: testOAuthRedirectURL,
			wantResourceURL: h.mcpURL(),
			wantOffline:     true,
		}),
		WithOAuthRedirectURL(testOAuthRedirectURL),
		WithAuthHTTPClient(h.client()),
	)

	cfg := streamableHTTPConfig("cimd", h.mcpURL(), spec.MCPHTTPAuthOAuth, "")
	cfg.StreamableHTTP.ClientIDMetadataDocumentURL = clientIDURL

	resolved, err := mgr.PrepareTransportAuth(ctx, cfg)
	if err != nil {
		t.Fatalf("PrepareTransportAuth: %v", err)
	}
	authorizeWithChallenge(t, h, resolved)

	if got := h.registerCallCount(); got != 0 {
		t.Fatalf("register calls = %d, want 0", got)
	}
}

func TestOAuthClientIDMetadataDocumentFallsBackToDCR(t *testing.T) {
	ctx := t.Context()
	h := newOAuthHarness(t)
	h.cimdSupported = false

	const clientIDURL = "https://client.example.com/flexigpt-mcp-client.json"

	mgr := NewAuthManager(
		StaticSecretResolver{},
		WithOAuthAuthorizationBroker(&testOAuthBroker{
			t:               t,
			wantClientID:    testClientID,
			wantRedirectURL: testOAuthRedirectURL,
			wantResourceURL: h.mcpURL(),
			wantOffline:     true,
		}),
		WithOAuthRedirectURL(testOAuthRedirectURL),
		WithAuthHTTPClient(h.client()),
	)

	cfg := streamableHTTPConfig("cimd-fallback", h.mcpURL(), spec.MCPHTTPAuthOAuth, "")
	cfg.StreamableHTTP.ClientIDMetadataDocumentURL = clientIDURL

	resolved, err := mgr.PrepareTransportAuth(ctx, cfg)
	if err != nil {
		t.Fatalf("PrepareTransportAuth: %v", err)
	}
	authorizeWithChallenge(t, h, resolved)

	if got := h.registerCallCount(); got != 1 {
		t.Fatalf("register calls = %d, want 1", got)
	}
}

func TestClientCredentialsGrantRefreshesToken(t *testing.T) {
	ctx := t.Context()
	h := newOAuthHarness(t)
	h.clients["service-client"] = "service-secret"
	h.expiresIn = 1

	const ref = "service-ref"

	mgr := NewAuthManager(
		StaticSecretResolver{
			ref: `{"clientID":"service-client","clientSecret":"service-secret"}`,
		},
		WithAuthHTTPClient(h.client()),
	)

	cfg := streamableHTTPConfig("service", h.mcpURL(), spec.MCPHTTPAuthClientCredentials, ref)
	resolved, err := mgr.PrepareTransportAuth(ctx, cfg)
	if err != nil {
		t.Fatalf("PrepareTransportAuth: %v", err)
	}

	req, resp := challengeRequestResponse(t, h, http.StatusUnauthorized)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err := resolved.OAuthHandler.Authorize(ctx, req, resp); err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	ts, err := resolved.OAuthHandler.TokenSource(ctx)
	if err != nil {
		t.Fatalf("TokenSource: %v", err)
	}
	if ts == nil {
		t.Fatalf("TokenSource returned nil")
	}

	before := h.tokenCallCount()

	tok1, err := ts.Token()
	if err != nil {
		t.Fatalf("Token #1: %v", err)
	}
	tok2, err := ts.Token()
	if err != nil {
		t.Fatalf("Token #2: %v", err)
	}

	if tok1.AccessToken == "" || tok2.AccessToken == "" {
		t.Fatalf("empty access token(s): %q %q", tok1.AccessToken, tok2.AccessToken)
	}
	if got := h.tokenCallCount() - before; got < 2 {
		t.Fatalf("token refresh calls after TokenSource = %d, want at least 2", got)
	}

	st, ok := mgr.GetAuthStatus(cfg.ID)
	if !ok {
		t.Fatalf("missing auth status")
	}
	if st.State != spec.MCPAuthStateAuthorized {
		t.Fatalf("state = %q, want authorized, lastError=%q", st.State, st.LastError)
	}
}

func authorizeWithChallenge(
	t *testing.T,

	h *oauthHarness,
	resolved ResolvedTransportAuth,
) {
	t.Helper()

	if resolved.OAuthHandler == nil {
		t.Fatalf("resolved OAuthHandler is nil")
	}

	req, resp := challengeRequestResponse(t, h, http.StatusUnauthorized)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err := resolved.OAuthHandler.Authorize(t.Context(), req, resp); err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	ts, err := resolved.OAuthHandler.TokenSource(t.Context())
	if err != nil {
		t.Fatalf("TokenSource: %v", err)
	}
	if ts == nil {
		t.Fatalf("TokenSource returned nil")
	}

	tok, err := ts.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok == nil || tok.AccessToken == "" {
		t.Fatalf("empty token: %#v", tok)
	}
}

func challengeRequestResponse(
	t *testing.T,
	h *oauthHarness,
	status int,
) (*http.Request, *http.Response) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, h.mcpURL(), http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp := &http.Response{
		StatusCode: status,
		Header: http.Header{
			"WWW-Authenticate": []string{h.bearerChallenge()},
		},
		Body: io.NopCloser(strings.NewReader("")),
	}

	return req, resp
}

func streamableHTTPConfig(
	id string,
	urlStr string,
	mode spec.MCPHTTPAuthMode,
	ref string,
) spec.MCPServerConfig {
	return spec.MCPServerConfig{
		ID:        spec.MCPServerID(id),
		Enabled:   true,
		Transport: spec.MCPTransportStreamableHTTP,
		StreamableHTTP: &spec.MCPStreamableHTTPConfig{
			URL:                 urlStr,
			AuthMode:            mode,
			ClientCredentialRef: ref,
		},
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
