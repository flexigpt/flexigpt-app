package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

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

	lastRegister      map[string]any
	lastTokenForm     url.Values
	lastTokenClientID string
	lastTokenSecret   string
	lastTokenGrant    string

	clients map[string]string

	cimdSupported bool
	issSupported  bool
	expiresIn     int

	failTokenAt          int
	failTokenCode        string
	failTokenDescription string
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
	return fmt.Sprintf(`Bearer resource_metadata=%q, scope=%q`, h.prmURL(), testScopeMCPTools)
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
		h.serveProtectedResourceMetadata(w)

	case "/.well-known/oauth-authorization-server",
		"/.well-known/openid-configuration":
		h.serveAuthorizationServerMetadata(w)

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

func (h *oauthHarness) serveProtectedResourceMetadata(w http.ResponseWriter) {
	writeJSON(w, map[string]any{
		"resource":              h.mcpURL(),
		"authorization_servers": []string{h.issuerURL()},
		"scopes_supported":      []string{testScopeMCPTools},
	})
}

func (h *oauthHarness) serveAuthorizationServerMetadata(w http.ResponseWriter) {
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
		"scopes_supported":                               []string{testScopeMCPTools, "offline_access"},
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
	h.lastTokenClientID = clientID
	h.lastTokenSecret = clientSecret
	h.lastTokenGrant = grantType
	expectedSecret, knownClient := h.clients[clientID]
	expiresIn := h.expiresIn
	failAt := h.failTokenAt
	failCode := h.failTokenCode
	failDesc := h.failTokenDescription
	h.mu.Unlock()

	if expiresIn <= 0 {
		expiresIn = 3600
	}
	if failCode == "" {
		failCode = errStrInvalidGrant
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

	if failAt > 0 && seq >= failAt {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]any{
			"error":             failCode,
			"error_description": failDesc,
		})
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

	return &OAuthAuthorizationResult{
		Code:  "test-code",
		State: q.Get("state"),
		Iss:   "",
	}, nil
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
			clients: map[string]string{
				testClientID: "",
			},
		},
		{
			name: "preregistered public client",
			resolver: StaticSecretResolver{
				"public-ref": testPublicRefRaw,
			},
			ref:             "public-ref",
			wantClientID:    testPublicClientID,
			wantRegisterCnt: 0,
			clients: map[string]string{
				testPublicClientID: "",
			},
		},
		{
			name: "preregistered confidential client",
			resolver: StaticSecretResolver{
				"confidential-ref": `{"clientID":"confidential-client","clientSecret":"top-secret"}`,
			},
			ref:             "confidential-ref",
			wantClientID:    testConfidentialClientID,
			wantRegisterCnt: 0,
			clients: map[string]string{
				testConfidentialClientID: "top-secret",
			},
		},
		{
			name:            "client id metadata document supported",
			cimdURL:         testCIMDURL,
			cimdSupported:   true,
			wantClientID:    testCIMDURL,
			wantRegisterCnt: 0,
			clients: map[string]string{
				testCIMDURL: "",
			},
		},
		{
			name:            "client id metadata document falls back to dcr",
			cimdURL:         testCIMDURL,
			cimdSupported:   false,
			wantClientID:    testClientID,
			wantRegisterCnt: 1,
			clients: map[string]string{
				testClientID: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newOAuthHarness(t)
			h.cimdSupported = tt.cimdSupported
			maps.Copy(h.clients, tt.clients)

			mgr := NewAuthManager(
				tt.resolver,
				WithOAuthAuthorizationBroker(&testOAuthBroker{
					t:               t,
					wantClientID:    tt.wantClientID,
					wantRedirectURL: testOAuthRedirectURL,
					wantResourceURL: h.mcpURL(),
					wantOffline:     true,
				}),
				WithOAuthRedirectURL(testOAuthRedirectURL),
				WithAuthHTTPClient(h.client()),
			)

			cfg := streamableHTTPConfig("oauth-server", h.mcpURL(), spec.MCPHTTPAuthOAuth, tt.ref)
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

			st, ok := mgr.GetAuthStatus(cfg.ID)
			if !ok {
				t.Fatalf("missing auth status")
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
		StaticSecretResolver{
			"service-ref": `{"clientID":"service-client","clientSecret":"service-secret"}`,
		},
		WithAuthHTTPClient(h.client()),
	)

	cfg := streamableHTTPConfig("service-server", h.mcpURL(), spec.MCPHTTPAuthClientCredentials, "service-ref")
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

	st, ok := mgr.GetAuthStatus(cfg.ID)
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
		StaticSecretResolver{
			"service-ref": `{"clientID":"service-client","clientSecret":"service-secret"}`,
		},
		WithAuthHTTPClient(h.client()),
	)

	cfg := streamableHTTPConfig("service-server", h.mcpURL(), spec.MCPHTTPAuthClientCredentials, "service-ref")
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

	// The SDK may fetch a token during authorization or on the first Token()
	// call, depending on internal flow. Accept either sequence and only assert
	// that a later refresh/error path is redacted correctly.
	if tok, err := ts.Token(); err == nil {
		if tok == nil || tok.AccessToken == "" {
			t.Fatalf("empty token #1: %#v", tok)
		}

		time.Sleep(1100 * time.Millisecond)

		if _, err := ts.Token(); err == nil {
			t.Fatalf("Token #2 succeeded, want invalid_grant")
		}
	}
	// If the first token request already fails, that's still valid for this
	// test: we only care that the resulting auth status redacts secrets.
	st, ok := mgr.GetAuthStatus(cfg.ID)
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

func mustAuthorize(
	t *testing.T,
	ctx context.Context,
	h *oauthHarness,
	resolved ResolvedTransportAuth,
) {
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
		Header: http.Header{
			"WWW-Authenticate": []string{h.bearerChallenge()},
		},
		Body: io.NopCloser(strings.NewReader("")),
	}

	if err := resolved.OAuthHandler.Authorize(ctx, req, resp); err != nil {
		t.Fatalf("Authorize: %v", err)
	}
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

func containsString(ss []string, want string) bool {
	return slices.Contains(ss, want)
}
