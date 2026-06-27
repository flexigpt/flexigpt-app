package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

type observingSecretResolver struct {
	called      bool
	sawCanceled bool
	lastRef     string
	err         error
}

func (r *observingSecretResolver) ResolveSecret(ctx context.Context, ref string) (string, error) {
	r.called = true
	r.lastRef = ref
	if ctx != nil {
		r.sawCanceled = ctx.Err() != nil
	}
	if r.err != nil {
		return "", r.err
	}
	return "resolved-secret", nil
}

func TestStaticSecretResolverResolveSecret(t *testing.T) {
	resolver := StaticSecretResolver{
		"ref-a": "secret-a",
	}

	got, err := resolver.ResolveSecret(t.Context(), "ref-a")
	if err != nil {
		t.Fatalf("ResolveSecret(existing): %v", err)
	}
	if got != "secret-a" {
		t.Fatalf("ResolveSecret(existing) = %q, want %q", got, "secret-a")
	}

	_, err = resolver.ResolveSecret(t.Context(), "missing")
	if err == nil {
		t.Fatalf("ResolveSecret(missing) succeeded, want error")
	}
	if !strings.Contains(err.Error(), "secret ref not found: missing") {
		t.Fatalf("ResolveSecret(missing) error = %q, want secret ref not found", err.Error())
	}
}

func TestSecretRedactorEdgeCases(t *testing.T) {
	t.Run("nil receiver and blank values are no-op", func(t *testing.T) {
		var redactor *SecretRedactor
		if got := redactor.Redact("keep me"); got != "keep me" {
			t.Fatalf("nil redactor changed string: %q", got)
		}

		got := NewSecretRedactor(ResolvedTransportAuth{
			SensitiveValues: []string{"", "   "},
		}).Redact("keep me")
		if got != "keep me" {
			t.Fatalf("blank-only redactor changed string: %q", got)
		}
	})

	t.Run("deduplicates and redacts exact values", func(t *testing.T) {
		redactor := NewSecretRedactor(ResolvedTransportAuth{
			SensitiveValues: []string{
				"alpha",
				"alpha",
				"  spaced-secret  ",
				"",
				"   ",
			},
		})

		got := redactor.Redact("alpha /  spaced-secret  / alpha")
		want := "[REDACTED] /[REDACTED]/ [REDACTED]"
		if got != want {
			t.Fatalf("Redact = %q, want %q", got, want)
		}
	})
}

func TestAuthManagerPrepareTransportAuthPersistsErrorStatusWithCanceledContext(t *testing.T) {
	resolver := &observingSecretResolver{
		err: errors.New("secret ref not found: missing-ref"),
	}

	mgr := NewAuthManager(resolver)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	cfg := spec.MCPServerConfig{
		BundleID:  testBundleID,
		ID:        testStdIOServerID,
		Transport: spec.MCPTransportStdio,
		Stdio: &spec.MCPStdioConfig{
			Command: "server-binary",
			SecretEnvRefs: map[string]string{
				"TOKEN": "missing-ref",
			},
		},
	}

	_, err := mgr.PrepareTransportAuth(ctx, cfg)
	if err == nil {
		t.Fatalf("PrepareTransportAuth succeeded, want error")
	}
	if !strings.Contains(err.Error(), "secret ref not found: missing-ref") {
		t.Fatalf("error = %q, want secret ref not found", err.Error())
	}

	if !resolver.called {
		t.Fatalf("secret resolver was not called")
	}
	if resolver.lastRef != "missing-ref" {
		t.Fatalf("resolver.lastRef = %q, want %q", resolver.lastRef, "missing-ref")
	}
	if !resolver.sawCanceled {
		t.Fatalf("resolver did not observe canceled context")
	}

	st, ok := mgr.GetAuthStatus(testBundleID, testStdIOServerID)
	if !ok {
		t.Fatalf("missing saved auth status")
	}

	wantStatus := spec.MCPAuthStatus{
		BundleID: testBundleID,
		ServerID: testStdIOServerID,
		AuthMode: spec.MCPHTTPAuthNone,
		State:    spec.MCPAuthStateError,
	}
	assertAuthStatusCore(t, st, wantStatus)

	if st.LastError != "secret ref not found: missing-ref" {
		t.Fatalf("LastError = %q, want %q", st.LastError, "secret ref not found: missing-ref")
	}
}

func TestAuthStatusHelperBranches(t *testing.T) {
	base := trackedBaseStatus(testHTTPServerID)

	t.Run("authStatusFromToken handles nil token", func(t *testing.T) {
		got := authStatusFromToken(base, nil)
		if got.State != spec.MCPAuthStateError {
			t.Fatalf("State = %q, want %q", got.State, spec.MCPAuthStateError)
		}
		if got.LastError != "oauth token source returned nil token" {
			t.Fatalf("LastError = %q, want empty", got.LastError)
		}
		if got.ExpiresAt != nil {
			t.Fatalf("ExpiresAt = %#v, want nil", got.ExpiresAt)
		}
		if len(got.Scopes) != 0 {
			t.Fatalf("Scopes = %#v, want empty", got.Scopes)
		}
	})

	t.Run("authStatusFromTokenError covers generic and nil errors", func(t *testing.T) {
		got := authStatusFromTokenError(base, errors.New("boom"))
		if got.State != spec.MCPAuthStateError {
			t.Fatalf("State = %q, want %q", got.State, spec.MCPAuthStateError)
		}
		if got.LastError != "boom" {
			t.Fatalf("LastError = %q, want %q", got.LastError, "boom")
		}

		gotNil := authStatusFromTokenError(base, nil)
		if gotNil.State != spec.MCPAuthStateError {
			t.Fatalf("nil error State = %q, want %q", gotNil.State, spec.MCPAuthStateError)
		}
		if gotNil.LastError != "" {
			t.Fatalf("nil error LastError = %q, want empty", gotNil.LastError)
		}
	})

	t.Run("authStatusFromHTTPFailure handles nil response and 401 without challenge", func(t *testing.T) {
		gotNil := authStatusFromHTTPFailure(base, nil, errors.New("request failed"))
		if gotNil.State != spec.MCPAuthStateError {
			t.Fatalf("nil response State = %q, want %q", gotNil.State, spec.MCPAuthStateError)
		}
		if gotNil.LastError != "request failed" {
			t.Fatalf("nil response LastError = %q, want %q", gotNil.LastError, "request failed")
		}

		resp := &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{},
		}
		got401 := authStatusFromHTTPFailure(base, resp, errors.New("unauthorized"))
		if got401.State != spec.MCPAuthStateRequired {
			t.Fatalf("401 State = %q, want %q", got401.State, spec.MCPAuthStateRequired)
		}
		if got401.LastError != "unauthorized" {
			t.Fatalf("401 LastError = %q, want %q", got401.LastError, "unauthorized")
		}
	})

	t.Run("scopesFromOAuthToken handles nil token", func(t *testing.T) {
		if got := scopesFromOAuthToken(nil); got != nil {
			t.Fatalf("scopesFromOAuthToken(nil) = %#v, want nil", got)
		}
	})
}

func TestTrackedOAuthHandlerAdditionalBranches(t *testing.T) {
	t.Run("TokenSource returns nil source without publishing status", func(t *testing.T) {
		serverID := spec.MCPServerID("tracked-nil-source")
		sink := NewAuthManager(nil)
		h := &trackedOAuthHandler{
			inner: &fakeOAuthHandler{
				tokenSource: nil,
			},
			sink:   sink,
			status: trackedBaseStatus(serverID),
		}

		ts, err := h.TokenSource(t.Context())
		if err != nil {
			t.Fatalf("TokenSource: %v", err)
		}
		if ts != nil {
			t.Fatalf("TokenSource = %#v, want nil", ts)
		}

		if _, ok := sink.GetAuthStatus(testBundleID, serverID); ok {
			t.Fatalf("auth status was published unexpectedly")
		}
	})

	t.Run("Authorize with no token source does not publish status", func(t *testing.T) {
		serverID := spec.MCPServerID("tracked-authorize-no-token-source")
		sink := NewAuthManager(nil)
		inner := &fakeOAuthHandler{
			tokenSource: nil,
		}
		h := &trackedOAuthHandler{
			inner:  inner,
			sink:   sink,
			status: trackedBaseStatus(serverID),
		}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testMCPResource, http.NoBody)
		resp := &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{},
		}

		if err := h.Authorize(t.Context(), req, resp); err != nil {
			t.Fatalf("Authorize: %v", err)
		}

		if inner.authorizeCalls != 1 {
			t.Fatalf("Authorize calls = %d, want 1", inner.authorizeCalls)
		}
		if inner.tokenSourceCalls != 1 {
			t.Fatalf("TokenSource calls = %d, want 1", inner.tokenSourceCalls)
		}

		if _, ok := sink.GetAuthStatus(testBundleID, serverID); ok {
			t.Fatalf("auth status was published unexpectedly")
		}
	})
}

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

func trackedBaseStatus(serverID spec.MCPServerID) spec.MCPAuthStatus {
	return spec.MCPAuthStatus{
		BundleID: testBundleID,
		ServerID: serverID,
		AuthMode: spec.MCPHTTPAuthOAuth,
		State:    spec.MCPAuthStateRequired,
		Resource: testMCPResource,
	}
}
