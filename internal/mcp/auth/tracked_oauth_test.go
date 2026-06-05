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
	"golang.org/x/oauth2"
)

type staticTokenSource struct {
	tok *oauth2.Token
	err error
}

func (s staticTokenSource) Token() (*oauth2.Token, error) {
	return s.tok, s.err
}

type fakeOAuthHandler struct {
	tokenSource      oauth2.TokenSource
	tokenSourceErr   error
	authorizeErr     error
	tokenSourceCalls int
	authorizeCalls   int
}

func (h *fakeOAuthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	h.tokenSourceCalls++
	if h.tokenSourceErr != nil {
		return nil, h.tokenSourceErr
	}
	return h.tokenSource, nil
}

func (h *fakeOAuthHandler) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	h.authorizeCalls++
	return h.authorizeErr
}

func TestOAuthStatusHelpers(t *testing.T) {
	t.Run("authStatusFromToken extracts expiry and scopes", func(t *testing.T) {
		expiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
		base := spec.MCPAuthStatus{
			BundleID: testBundleID,
			ServerID: "server",
			AuthMode: spec.MCPHTTPAuthOAuth,
			State:    spec.MCPAuthStateRequired,
			Resource: testMCPResource,
		}

		tok := tokenWithScopes("scope-a scope-b", expiry)
		got := authStatusFromToken(base, tok)

		if got.State != spec.MCPAuthStateAuthorized {
			t.Fatalf("State = %q, want %q", got.State, spec.MCPAuthStateAuthorized)
		}
		if got.LastError != "" {
			t.Fatalf("LastError = %q, want empty", got.LastError)
		}
		if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expiry) {
			t.Fatalf("ExpiresAt = %#v, want %v", got.ExpiresAt, expiry)
		}
		if got.Scopes == nil || len(got.Scopes) != 2 || got.Scopes[0] != "scope-a" || got.Scopes[1] != "scope-b" {
			t.Fatalf("Scopes = %#v, want [scope-a scope-b]", got.Scopes)
		}
	})

	t.Run("scopesFromOAuthToken accepts string, []string, and []any", func(t *testing.T) {
		tok1 := tokenWithScopes("alpha beta", time.Time{})
		got1 := scopesFromOAuthToken(tok1)
		if len(got1) != 2 || got1[0] != "alpha" || got1[1] != "beta" {
			t.Fatalf("scopesFromOAuthToken(string) = %#v", got1)
		}

		tok2 := tokenWithScopes([]string{"alpha", "beta"}, time.Time{})
		got2 := scopesFromOAuthToken(tok2)
		if len(got2) != 2 || got2[0] != "alpha" || got2[1] != "beta" {
			t.Fatalf("scopesFromOAuthToken([]string) = %#v", got2)
		}

		tok3 := tokenWithScopes([]any{"alpha", "", 123, "beta"}, time.Time{})
		got3 := scopesFromOAuthToken(tok3)
		if len(got3) != 2 || got3[0] != "alpha" || got3[1] != "beta" {
			t.Fatalf("scopesFromOAuthToken([]any) = %#v", got3)
		}
	})

	t.Run("authStatusFromTokenError marks invalid_grant expired", func(t *testing.T) {
		base := spec.MCPAuthStatus{
			BundleID: testBundleID,
			ServerID: "server",
			AuthMode: spec.MCPHTTPAuthOAuth,
			State:    spec.MCPAuthStateRequired,
		}

		got := authStatusFromTokenError(base, &oauth2.RetrieveError{ErrorCode: errStrInvalidGrant})
		if got.State != spec.MCPAuthStateExpired {
			t.Fatalf("State = %q, want %q", got.State, spec.MCPAuthStateExpired)
		}
	})
}

func TestTrackingTokenSourcePublishesStatus(t *testing.T) {
	sink := NewAuthManager(nil)
	base := spec.MCPAuthStatus{
		BundleID: testBundleID,
		ServerID: "server",
		AuthMode: spec.MCPHTTPAuthOAuth,
		State:    spec.MCPAuthStateRequired,
		Resource: testMCPResource,
	}

	t.Run("success publishes authorized status", func(t *testing.T) {
		expiry := time.Now().UTC().Add(30 * time.Minute).Truncate(time.Second)
		src := &trackingTokenSource{
			source:          staticTokenSource{tok: tokenWithScopes("mcp:tools admin", expiry)},
			sink:            sink,
			status:          base,
			sensitiveValues: []string{"secret-value"},
		}

		tok, err := src.Token()
		if err != nil {
			t.Fatalf("Token: %v", err)
		}
		if tok == nil || tok.AccessToken == "" {
			t.Fatalf("token = %#v", tok)
		}

		st, ok := sink.GetAuthStatus(testBundleID, base.ServerID)
		if !ok {
			t.Fatalf("missing auth status")
		}
		if st.BundleID != testBundleID {
			t.Fatalf("BundleID = %q, want %q", st.BundleID, testBundleID)
		}
		if st.State != spec.MCPAuthStateAuthorized {
			t.Fatalf("State = %q, want %q", st.State, spec.MCPAuthStateAuthorized)
		}
		if st.ExpiresAt == nil || !st.ExpiresAt.Equal(expiry) {
			t.Fatalf("ExpiresAt = %#v, want %v", st.ExpiresAt, expiry)
		}
		if len(st.Scopes) != 2 || st.Scopes[0] != "mcp:tools" || st.Scopes[1] != "admin" {
			t.Fatalf("Scopes = %#v", st.Scopes)
		}
		if strings.Contains(st.LastError, "secret-value") {
			t.Fatalf("LastError leaked secret: %q", st.LastError)
		}
	})

	t.Run("nil token publishes error", func(t *testing.T) {
		src := &trackingTokenSource{source: staticTokenSource{}, sink: sink, status: base}

		tok, err := src.Token()
		if err == nil || tok != nil {
			t.Fatalf("Token = %#v, err=%v, want nil token error", tok, err)
		}

		st, ok := sink.GetAuthStatus(testBundleID, base.ServerID)
		if !ok {
			t.Fatalf("missing auth status")
		}
		if st.State != spec.MCPAuthStateError {
			t.Fatalf("State = %q, want %q", st.State, spec.MCPAuthStateError)
		}
		if !strings.Contains(st.LastError, "nil token") {
			t.Fatalf("LastError = %q, want nil token error", st.LastError)
		}
	})

	t.Run("invalid_grant publishes expired status", func(t *testing.T) {
		src := &trackingTokenSource{
			source: staticTokenSource{err: &oauth2.RetrieveError{ErrorCode: errStrInvalidGrant}},
			sink:   sink,
			status: base,
		}

		tok, err := src.Token()
		if err == nil || tok != nil {
			t.Fatalf("Token = %#v, err=%v, want error", tok, err)
		}

		st, ok := sink.GetAuthStatus(testBundleID, base.ServerID)
		if !ok {
			t.Fatalf("missing auth status")
		}
		if st.State != spec.MCPAuthStateExpired {
			t.Fatalf("State = %q, want %q", st.State, spec.MCPAuthStateExpired)
		}
	})
}

func TestTrackedOAuthHandlerTokenSourceAndAuthorize(t *testing.T) {
	t.Run("TokenSource error is published", func(t *testing.T) {
		sink := NewAuthManager(nil)
		base := spec.MCPAuthStatus{
			BundleID: testBundleID,
			ServerID: "server-a",
			AuthMode: spec.MCPHTTPAuthOAuth,
			State:    spec.MCPAuthStateRequired,
		}
		h := &trackedOAuthHandler{
			inner:           &fakeOAuthHandler{tokenSourceErr: errors.New("token source failed top-secret")},
			sink:            sink,
			status:          base,
			sensitiveValues: []string{"top-secret"},
		}

		ts, err := h.TokenSource(t.Context())
		if err == nil || ts != nil {
			t.Fatalf("TokenSource = %#v, err=%v, want error", ts, err)
		}

		st, ok := sink.GetAuthStatus(testBundleID, base.ServerID)
		if !ok {
			t.Fatalf("missing auth status")
		}
		if st.State != spec.MCPAuthStateError {
			t.Fatalf("State = %q, want %q", st.State, spec.MCPAuthStateError)
		}
		if strings.Contains(st.LastError, "top-secret") {
			t.Fatalf("LastError leaked secret: %q", st.LastError)
		}
	})

	t.Run("Authorize success publishes authorized status", func(t *testing.T) {
		sink := NewAuthManager(nil)
		base := spec.MCPAuthStatus{
			BundleID: testBundleID,
			ServerID: "server-b",
			AuthMode: spec.MCPHTTPAuthOAuth,
			State:    spec.MCPAuthStateRequired,
			Resource: testMCPResource,
		}
		expiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
		tok := tokenWithScopes("scope-a scope-b", expiry)
		inner := &fakeOAuthHandler{tokenSource: staticTokenSource{tok: tok}}
		h := &trackedOAuthHandler{inner: inner, sink: sink, status: base, sensitiveValues: []string{"top-secret"}}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testMCPResource, http.NoBody)
		resp := &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{"WWW-Authenticate": []string{`Bearer scope="scope-a scope-b"`}},
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

		st, ok := sink.GetAuthStatus(testBundleID, base.ServerID)
		if !ok {
			t.Fatalf("missing auth status")
		}
		if st.State != spec.MCPAuthStateAuthorized {
			t.Fatalf("State = %q, want %q", st.State, spec.MCPAuthStateAuthorized)
		}
		if st.ExpiresAt == nil || !st.ExpiresAt.Equal(expiry) {
			t.Fatalf("ExpiresAt = %#v, want %v", st.ExpiresAt, expiry)
		}
		if len(st.Scopes) != 2 || st.Scopes[0] != "scope-a" || st.Scopes[1] != "scope-b" {
			t.Fatalf("Scopes = %#v, want [scope-a scope-b]", st.Scopes)
		}
	})
}

func tokenWithScopes(scopes any, expiry time.Time) *oauth2.Token {
	tok := &oauth2.Token{
		AccessToken: "access-token",
		TokenType:   "Bearer",
		Expiry:      expiry,
	}
	return tok.WithExtra(map[string]any{"scope": scopes})
}
