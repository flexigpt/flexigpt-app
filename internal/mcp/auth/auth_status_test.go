package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	mcpAuth "github.com/modelcontextprotocol/go-sdk/auth"
	"golang.org/x/oauth2"
)

const (
	testServerID       = "server"
	testBearerHeader   = "Bearer"
	testSensitiveValue = "top-secret"
)

func TestAuthStatusFromHTTPFailureStatusTransitions(t *testing.T) {
	base := spec.MCPAuthStatus{
		ServerID: testServerID,
		AuthMode: spec.MCPHTTPAuthOAuth,
		State:    spec.MCPAuthStateRequired,
		Resource: "https://example.test/mcp",
	}

	tests := []struct {
		name   string
		status int
		header string
		want   spec.MCPAuthState
	}{
		{
			name:   "401 means required",
			status: http.StatusUnauthorized,
			header: testBearerHeader,
			want:   spec.MCPAuthStateRequired,
		},
		{
			name:   "403 without insufficient_scope remains error",
			status: http.StatusForbidden,
			header: testBearerHeader,
			want:   spec.MCPAuthStateError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.status,
				Header: http.Header{
					"WWW-Authenticate": []string{tt.header},
				},
			}
			st := authStatusFromHTTPFailure(base, resp, errors.New("request failed"))
			if st.State != tt.want {
				p, _ := json.MarshalIndent(resp, "", "")
				t.Fatalf("state = %q, want %q http %s", st.State, tt.want, string(p))
			}
			if tt.want == spec.MCPAuthStateInsufficientScope {
				if got := strings.Join(st.Scopes, " "); got != "mcp:tools admin" {
					t.Fatalf("scopes = %q, want %q", got, "mcp:tools admin")
				}
			}
		})
	}
}

func TestAuthStatusFromTokenErrorInvalidGrantIsExpired(t *testing.T) {
	base := spec.MCPAuthStatus{
		ServerID: testServerID,
		AuthMode: spec.MCPHTTPAuthOAuth,
		State:    spec.MCPAuthStateAuthorized,
	}

	st := authStatusFromTokenError(base, &oauth2.RetrieveError{
		ErrorCode:        "invalid_grant",
		ErrorDescription: "refresh token expired",
	})
	if st.State != spec.MCPAuthStateExpired {
		t.Fatalf("state = %q, want expired", st.State)
	}
}

func TestTrackedOAuthHandlerRedactsAuthorizeErrorsInStatus(t *testing.T) {
	sink := &captureAuthStatusSink{}
	inner := &fakeOAuthHandler{
		authorizeErr: errors.New("token endpoint rejected client secret top-secret"),
	}

	h := &trackedOAuthHandler{
		inner: inner,
		sink:  sink,
		status: spec.MCPAuthStatus{
			ServerID: testServerID,
			AuthMode: spec.MCPHTTPAuthOAuth,
			State:    spec.MCPAuthStateRequired,
		},
		sensitiveValues: []string{testSensitiveValue},
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.test/mcp", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}

	if err := h.Authorize(t.Context(), req, resp); err == nil {
		t.Fatalf("Authorize returned nil error")
	}

	st, ok := sink.last()
	if !ok {
		t.Fatalf("missing saved status")
	}
	if strings.Contains(st.LastError, testSensitiveValue) {
		t.Fatalf("LastError leaked secret: %q", st.LastError)
	}
	if !strings.Contains(st.LastError, "[REDACTED]") {
		t.Fatalf("LastError was not redacted: %q", st.LastError)
	}
}

func TestTrackedOAuthHandlerRedactsTokenSourceErrorsInStatus(t *testing.T) {
	sink := &captureAuthStatusSink{}
	inner := &fakeOAuthHandler{
		tokenSource: errorTokenSource{
			err: errors.New("refresh failed for top-secret"),
		},
	}

	h := &trackedOAuthHandler{
		inner: inner,
		sink:  sink,
		status: spec.MCPAuthStatus{
			ServerID: testServerID,
			AuthMode: spec.MCPHTTPAuthOAuth,
			State:    spec.MCPAuthStateAuthorized,
		},
		sensitiveValues: []string{testSensitiveValue},
	}

	ts, err := h.TokenSource(t.Context())
	if err != nil {
		t.Fatalf("TokenSource: %v", err)
	}
	if ts == nil {
		t.Fatalf("TokenSource returned nil")
	}

	if _, err := ts.Token(); err == nil {
		t.Fatalf("Token returned nil error")
	}

	st, ok := sink.last()
	if !ok {
		t.Fatalf("missing saved status")
	}
	if strings.Contains(st.LastError, testSensitiveValue) {
		t.Fatalf("LastError leaked secret: %q", st.LastError)
	}
	if !strings.Contains(st.LastError, "[REDACTED]") {
		t.Fatalf("LastError was not redacted: %q", st.LastError)
	}
}

type captureAuthStatusSink struct {
	mu       sync.Mutex
	statuses []spec.MCPAuthStatus
}

func (s *captureAuthStatusSink) SaveAuthStatus(ctx context.Context, st spec.MCPAuthStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses = append(s.statuses, cloneAuthStatus(st))
	return nil
}

func (s *captureAuthStatusSink) last() (spec.MCPAuthStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.statuses) == 0 {
		return spec.MCPAuthStatus{}, false
	}
	return cloneAuthStatus(s.statuses[len(s.statuses)-1]), true
}

type fakeOAuthHandler struct {
	tokenSource  oauth2.TokenSource
	tokenErr     error
	authorizeErr error
}

func (h *fakeOAuthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	if h.tokenErr != nil {
		return nil, h.tokenErr
	}
	return h.tokenSource, nil
}

func (h *fakeOAuthHandler) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
	return h.authorizeErr
}

var _ mcpAuth.OAuthHandler = (*fakeOAuthHandler)(nil)

type errorTokenSource struct {
	err error
}

func (s errorTokenSource) Token() (*oauth2.Token, error) {
	return nil, s.err
}
