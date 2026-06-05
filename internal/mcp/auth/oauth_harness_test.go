package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
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

	h := &oauthHarness{t: t, clients: map[string]string{}}
	h.server = httptest.NewServer(http.HandlerFunc(h.serveHTTP))
	t.Cleanup(h.server.Close)

	return h
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
	case "/.well-known/oauth-protected-resource/mcp", "/.well-known/oauth-protected-resource":
		h.serveProtectedResourceMetadata(w)
	case "/.well-known/oauth-authorization-server", "/.well-known/openid-configuration":
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
	writeJSON(
		w,
		map[string]any{
			"resource":              h.mcpURL(),
			"authorization_servers": []string{h.issuerURL()},
			"scopes_supported":      []string{testScopeMCPTools},
		},
	)
}

func (h *oauthHarness) serveAuthorizationServerMetadata(w http.ResponseWriter) {
	writeJSON(w, map[string]any{
		"issuer":                   h.issuerURL(),
		"authorization_endpoint":   h.issuerURL() + "/authorize",
		"token_endpoint":           h.issuerURL() + "/token",
		"registration_endpoint":    h.issuerURL() + "/register",
		"response_types_supported": []string{testResponseType},
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

	writeJSON(
		w,
		map[string]any{
			"client_id":                  testClientID,
			"redirect_uris":              body["redirect_uris"],
			"token_endpoint_auth_method": "none",
			"grant_types":                body["grant_types"],
			"response_types":             body["response_types"],
			"client_name":                body["client_name"],
		},
	)
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
		if r.Form.Get(testResponseType) != "test-code" {
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
		writeJSON(w, map[string]any{"error": failCode, "error_description": failDesc})
		return
	}

	writeJSON(
		w,
		map[string]any{
			"access_token":  fmt.Sprintf("access-%d", seq),
			"refresh_token": fmt.Sprintf("refresh-%d", seq),
			"token_type":    "Bearer",
			"expires_in":    expiresIn,
			"scope":         "mcp:tools offline_access",
		},
	)
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
