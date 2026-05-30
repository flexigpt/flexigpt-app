package auth

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	testScopeMCPTools = "mcp:tools"
	testScopeAdmin    = "admin"
	testScopeA        = "scope-a"
	testScopeB        = "scope-b"
)

var testWantScope = []string{testScopeMCPTools, testScopeAdmin}

func TestBearerChallengeValues(t *testing.T) {
	tests := []struct {
		name      string
		headers   []string
		wantErr   string
		wantScope []string
	}{
		{name: "bearer scope", headers: []string{`Bearer scope="mcp:tools admin"`}, wantScope: testWantScope},
		{
			name:      "bearer insufficient scope",
			headers:   []string{`Basic realm="ignored"`, `Bearer error="insufficient_scope", scope="mcp:tools admin"`},
			wantErr:   "insufficient_scope",
			wantScope: testWantScope,
		},
		{name: "no bearer challenge", headers: []string{`Basic realm="ignored"`}},
		{name: "invalid header", headers: []string{`Bearer foo="bar"`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr, gotScopes := bearerChallengeValues(tt.headers)
			if gotErr != tt.wantErr {
				t.Fatalf("challengeErr = %q, want %q", gotErr, tt.wantErr)
			}
			if len(gotScopes) != len(tt.wantScope) {
				t.Fatalf("scopes len = %d, want %d", len(gotScopes), len(tt.wantScope))
			}
			for i := range tt.wantScope {
				if gotScopes[i] != tt.wantScope[i] {
					t.Fatalf("scope[%d] = %q, want %q", i, gotScopes[i], tt.wantScope[i])
				}
			}
		})
	}
}

func TestAuthStatusFromHTTPFailure(t *testing.T) {
	base := spec.MCPAuthStatus{
		BundleID: testBundleID,
		ServerID: testHTTPServerID,
		AuthMode: spec.MCPHTTPAuthOAuth,
		State:    spec.MCPAuthStateRequired,
		Resource: testMCPResource,
	}

	tests := []struct {
		name       string
		statusCode int
		challenge  string
		wantState  spec.MCPAuthState
		wantScopes []string
	}{
		{
			name:       "401 required",
			statusCode: http.StatusUnauthorized,
			challenge:  `Bearer scope="mcp:tools admin"`,
			wantState:  spec.MCPAuthStateRequired,
			wantScopes: testWantScope,
		},
		{
			name:       "403 insufficient scope",
			statusCode: http.StatusForbidden,
			challenge:  `Bearer error="insufficient_scope", scope="mcp:tools admin"`,
			wantState:  spec.MCPAuthStateInsufficientScope,
			wantScopes: testWantScope,
		},
		{
			name:       "403 invalid token",
			statusCode: http.StatusForbidden,
			challenge:  `Bearer error="invalid_token", scope="mcp:tools"`,
			wantState:  spec.MCPAuthStateExpired,
			wantScopes: []string{testScopeMCPTools},
		},
		{name: "403 no challenge", statusCode: http.StatusForbidden, wantState: spec.MCPAuthStateError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.statusCode, Header: http.Header{}}
			if tt.challenge != "" {
				resp.Header.Set("WWW-Authenticate", tt.challenge)
			}

			st := authStatusFromHTTPFailure(base, resp, errors.New("request failed"))
			if st.State != tt.wantState {
				t.Fatalf("State = %q, want %q", st.State, tt.wantState)
			}
			if st.LastError != "request failed" {
				t.Fatalf("LastError = %q, want %q", st.LastError, "request failed")
			}
			if len(st.Scopes) != len(tt.wantScopes) {
				t.Fatalf("Scopes len = %d, want %d", len(st.Scopes), len(tt.wantScopes))
			}
			for i := range tt.wantScopes {
				if st.Scopes[i] != tt.wantScopes[i] {
					t.Fatalf("Scopes[%d] = %q, want %q", i, st.Scopes[i], tt.wantScopes[i])
				}
			}
		})
	}
}

func TestRedactAuthStatus(t *testing.T) {
	st := spec.MCPAuthStatus{
		BundleID:  testBundleID,
		ServerID:  testHTTPServerID,
		AuthMode:  spec.MCPHTTPAuthOAuth,
		State:     spec.MCPAuthStateError,
		LastError: "token endpoint rejected top-secret",
		Resource:  testMCPResource,
	}

	got := redactAuthStatus(st, []string{"top-secret", "", "   "})

	if strings.Contains(got.LastError, "top-secret") {
		t.Fatalf("LastError leaked secret: %q", got.LastError)
	}
	if !strings.Contains(got.LastError, "[REDACTED]") {
		t.Fatalf("LastError was not redacted: %q", got.LastError)
	}
	if got.Resource != st.Resource {
		t.Fatalf("Resource changed: got %q want %q", got.Resource, st.Resource)
	}
}

func TestAuthManagerStatusLifecycle(t *testing.T) {
	mgr := NewAuthManager(nil)

	expiresAt := time.Now().UTC().Add(time.Hour)
	wantExpiresAt := expiresAt

	in := spec.MCPAuthStatus{
		BundleID:  testBundleID,
		ServerID:  testHTTPServerID,
		AuthMode:  spec.MCPHTTPAuthOAuth,
		State:     spec.MCPAuthStateAuthorized,
		Scopes:    []string{testScopeA, testScopeB},
		ExpiresAt: &expiresAt,
	}

	if err := mgr.SaveAuthStatus(t.Context(), in); err != nil {
		t.Fatalf("SaveAuthStatus: %v", err)
	}

	in.Scopes[0] = "mutated"
	expiresAt = expiresAt.Add(2 * time.Hour)

	got, ok := mgr.GetAuthStatus(testBundleID, testHTTPServerID)
	if !ok {
		t.Fatalf("missing status")
	}
	if got.Scopes[0] != testScopeA || got.Scopes[1] != testScopeB {
		t.Fatalf("Scopes were not cloned: %#v", got.Scopes)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(wantExpiresAt) {
		t.Fatalf("ExpiresAt = %#v, want %v", got.ExpiresAt, wantExpiresAt)
	}

	mgr.ClearAuthStatus(testBundleID, testHTTPServerID)
	if _, ok := mgr.GetAuthStatus(testBundleID, testHTTPServerID); ok {
		t.Fatalf("status was not cleared")
	}

	if err := mgr.SaveAuthStatus(t.Context(), in); err != nil {
		t.Fatalf("SaveAuthStatus #2: %v", err)
	}
	mgr.ClearAuthStatuses()
	if _, ok := mgr.GetAuthStatus(testBundleID, testHTTPServerID); ok {
		t.Fatalf("statuses were not cleared")
	}
}

func TestDefaultMCPAuthStatusFromConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       spec.MCPServerConfig
		wantMode  spec.MCPHTTPAuthMode
		wantState spec.MCPAuthState
		wantRes   string
	}{
		{
			name:      "no streamable http config",
			cfg:       spec.MCPServerConfig{BundleID: testBundleID, ID: testHTTPServerID},
			wantMode:  spec.MCPHTTPAuthNone,
			wantState: spec.MCPAuthStateNotRequired,
		},
		{
			name: "oauth",
			cfg: spec.MCPServerConfig{
				BundleID: testBundleID,
				ID:       testHTTPServerID,
				StreamableHTTP: &spec.MCPStreamableHTTPConfig{
					URL:      " https://example.test/mcp ",
					AuthMode: spec.MCPHTTPAuthOAuth,
				},
			},
			wantMode:  spec.MCPHTTPAuthOAuth,
			wantState: spec.MCPAuthStateRequired,
			wantRes:   testMCPResource,
		},
		{
			name: "client credentials",
			cfg: spec.MCPServerConfig{
				BundleID: testBundleID,
				ID:       testHTTPServerID,
				StreamableHTTP: &spec.MCPStreamableHTTPConfig{
					URL:      testMCPResource,
					AuthMode: spec.MCPHTTPAuthClientCredentials,
				},
			},
			wantMode:  spec.MCPHTTPAuthClientCredentials,
			wantState: spec.MCPAuthStateRequired,
			wantRes:   testMCPResource,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultMCPAuthStatusFromConfig(tt.cfg)
			if got.BundleID != tt.cfg.BundleID {
				t.Fatalf("BundleID = %q, want %q", got.BundleID, tt.cfg.BundleID)
			}
			if got.ServerID != tt.cfg.ID {
				t.Fatalf("ServerID = %q, want %q", got.ServerID, tt.cfg.ID)
			}
			if got.AuthMode != tt.wantMode {
				t.Fatalf("AuthMode = %q, want %q", got.AuthMode, tt.wantMode)
			}
			if got.State != tt.wantState {
				t.Fatalf("State = %q, want %q", got.State, tt.wantState)
			}
			if got.Resource != tt.wantRes {
				t.Fatalf("Resource = %q, want %q", got.Resource, tt.wantRes)
			}
		})
	}
}

func TestMergeMCPAuthStatus(t *testing.T) {
	cfgOAuth := spec.MCPServerConfig{
		BundleID: testBundleID,
		ID:       testHTTPServerID,
		StreamableHTTP: &spec.MCPStreamableHTTPConfig{
			URL:      testMCPResource,
			AuthMode: spec.MCPHTTPAuthOAuth,
		},
	}
	defOAuth := DefaultMCPAuthStatusFromConfig(cfgOAuth)

	tests := []struct {
		name string
		st   spec.MCPAuthStatus
		cfg  spec.MCPServerConfig
		want spec.MCPAuthStatus
	}{
		{
			name: "fills defaults from config",
			st: spec.MCPAuthStatus{
				BundleID: testBundleID,
				ServerID: testHTTPServerID,
				State:    spec.MCPAuthStateAuthorized,
				Scopes:   []string{testScopeA},
			},
			cfg: cfgOAuth,
			want: spec.MCPAuthStatus{
				BundleID: testBundleID,
				ServerID: testHTTPServerID,
				AuthMode: spec.MCPHTTPAuthOAuth,
				State:    spec.MCPAuthStateAuthorized,
				Resource: testMCPResource,
				Scopes:   []string{testScopeA},
			},
		},
		{
			name: "mismatch auth mode resets to default",
			st: spec.MCPAuthStatus{
				BundleID:  testBundleID,
				ServerID:  testHTTPServerID,
				AuthMode:  spec.MCPHTTPAuthClientCredentials,
				State:     spec.MCPAuthStateError,
				LastError: "boom",
				Resource:  testMCPResource,
			},
			cfg:  cfgOAuth,
			want: defOAuth,
		},
		{
			name: "none auth clears auth-specific fields",
			st: spec.MCPAuthStatus{
				BundleID:            testBundleID,
				ServerID:            testHTTPServerID,
				AuthMode:            spec.MCPHTTPAuthNone,
				State:               spec.MCPAuthStateError,
				Scopes:              []string{testScopeA},
				LastError:           "boom",
				AuthorizationServer: "https://issuer.test",
			},
			cfg: spec.MCPServerConfig{BundleID: testBundleID, ID: testHTTPServerID},
			want: spec.MCPAuthStatus{
				BundleID: testBundleID,
				ServerID: testHTTPServerID,
				AuthMode: spec.MCPHTTPAuthNone,
				State:    spec.MCPAuthStateNotRequired,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeMCPAuthStatus(tt.st, tt.cfg)

			if got.BundleID != tt.want.BundleID {
				t.Fatalf("BundleID = %q, want %q", got.BundleID, tt.want.BundleID)
			}
			if got.ServerID != tt.want.ServerID {
				t.Fatalf("ServerID = %q, want %q", got.ServerID, tt.want.ServerID)
			}
			if got.AuthMode != tt.want.AuthMode {
				t.Fatalf("AuthMode = %q, want %q", got.AuthMode, tt.want.AuthMode)
			}
			if got.State != tt.want.State {
				t.Fatalf("State = %q, want %q", got.State, tt.want.State)
			}
			if got.Resource != tt.want.Resource {
				t.Fatalf("Resource = %q, want %q", got.Resource, tt.want.Resource)
			}
			if len(got.Scopes) != len(tt.want.Scopes) {
				t.Fatalf("Scopes len = %d, want %d", len(got.Scopes), len(tt.want.Scopes))
			}
			for i := range tt.want.Scopes {
				if got.Scopes[i] != tt.want.Scopes[i] {
					t.Fatalf("Scopes[%d] = %q, want %q", i, got.Scopes[i], tt.want.Scopes[i])
				}
			}
			if tt.want.LastError == "" && got.LastError != "" {
				t.Fatalf("LastError = %q, want empty", got.LastError)
			}
			if tt.want.AuthorizationServer == "" && got.AuthorizationServer != "" {
				t.Fatalf("AuthorizationServer = %q, want empty", got.AuthorizationServer)
			}
		})
	}
}
