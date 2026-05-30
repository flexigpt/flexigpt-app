package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type oauthAuthorizationResult struct {
	res *OAuthAuthorizationResult
	err error
}

func TestAuthorizationStateAndErrorDescription(t *testing.T) {
	t.Run("state from URL", func(t *testing.T) {
		got, err := authorizationState("https://issuer.test/authorize?client_id=abc&state=state-123")
		if err != nil {
			t.Fatalf("authorizationState: %v", err)
		}
		if got != "state-123" {
			t.Fatalf("state = %q, want %q", got, "state-123")
		}
	})

	t.Run("missing state", func(t *testing.T) {
		_, err := authorizationState("https://issuer.test/authorize?client_id=abc")
		if err == nil || !strings.Contains(err.Error(), "missing state") {
			t.Fatalf("err = %v, want substring %q", err, "missing state")
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		_, err := authorizationState("://bad-url")
		if err == nil || !strings.Contains(err.Error(), "invalid OAuth authorization URL") {
			t.Fatalf("err = %v, want invalid OAuth authorization URL", err)
		}
	})

	t.Run("format description", func(t *testing.T) {
		if got := formatOAuthErrorDescription("  access denied  "); got != ": access denied" {
			t.Fatalf("formatOAuthErrorDescription = %q, want %q", got, ": access denied")
		}
		if got := formatOAuthErrorDescription("   "); got != "" {
			t.Fatalf("formatOAuthErrorDescription blank = %q, want empty", got)
		}
	})
}

func TestOAuthLoopbackBrokerLifecycle(t *testing.T) {
	t.Run("success callback completes fetch", func(t *testing.T) {
		ctx := t.Context()
		b, err := NewOAuthLoopbackBroker(ctx, &OAuthLoopbackBrokerOptions{
			TTL:          time.Minute,
			CallbackPath: "/mcp/oauth/callback",
		})
		if err != nil {
			t.Fatalf("NewOAuthLoopbackBroker: %v", err)
		}
		t.Cleanup(func() { _ = b.Close() })

		serverID := spec.MCPServerID("server-a")
		state := "state-a"
		authURL := "https://issuer.test/authorize?state=" + state + "&client_id=client-a"

		resultCh := make(chan oauthAuthorizationResult, 1)
		go func() {
			res, err := b.FetchAuthorizationCode(ctx, OAuthAuthorizationRequest{
				ServerID:         serverID,
				AuthorizationURL: authURL,
			})
			resultCh <- oauthAuthorizationResult{res: res, err: err}
		}()

		waitForPendingCount(t, b, 1)

		pending := b.Pending()
		if len(pending) != 1 {
			t.Fatalf("Pending len = %d, want 1", len(pending))
		}
		if pending[0].ServerID != serverID {
			t.Fatalf("Pending[0].ServerID = %q, want %q", pending[0].ServerID, serverID)
		}
		if pending[0].AuthorizationURL != authURL {
			t.Fatalf("Pending[0].AuthorizationURL = %q, want %q", pending[0].AuthorizationURL, authURL)
		}
		if pending[0].ExpiresAt == "" {
			t.Fatalf("Pending[0].ExpiresAt is empty")
		}

		req := httptest.NewRequestWithContext(t.Context(),
			http.MethodGet,
			b.RedirectURL()+"?state="+state+"&code=test-code&iss=https%3A%2F%2Fissuer.test",
			http.NoBody,
		)
		req.Host = b.redirectHost

		rr := httptest.NewRecorder()
		b.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("ServeHTTP status = %d, want %d; body=%q", rr.Code, http.StatusOK, rr.Body.String())
		}

		select {
		case got := <-resultCh:
			if got.err != nil {
				t.Fatalf("FetchAuthorizationCode: %v", got.err)
			}
			if got.res == nil {
				t.Fatalf("result is nil")
			}
			if got.res.Code != "test-code" {
				t.Fatalf("Code = %q, want %q", got.res.Code, "test-code")
			}
			if got.res.State != state {
				t.Fatalf("State = %q, want %q", got.res.State, state)
			}
			if got.res.Iss != "https://issuer.test" {
				t.Fatalf("Iss = %q, want %q", got.res.Iss, "https://issuer.test")
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for authorization result")
		}

		if got := b.Pending(); len(got) != 0 {
			t.Fatalf("Pending len after success = %d, want 0", len(got))
		}
	})

	t.Run("authorization server error completes fetch with error", func(t *testing.T) {
		ctx := t.Context()
		b, err := NewOAuthLoopbackBroker(ctx, &OAuthLoopbackBrokerOptions{
			TTL:          time.Minute,
			CallbackPath: "/mcp/oauth/callback",
		})
		if err != nil {
			t.Fatalf("NewOAuthLoopbackBroker: %v", err)
		}
		t.Cleanup(func() { _ = b.Close() })

		serverID := spec.MCPServerID("server-b")
		state := "state-b"
		authURL := "https://issuer.test/authorize?state=" + state

		resultCh := make(chan oauthAuthorizationResult, 1)
		go func() {
			res, err := b.FetchAuthorizationCode(ctx, OAuthAuthorizationRequest{
				ServerID:         serverID,
				AuthorizationURL: authURL,
			})
			resultCh <- oauthAuthorizationResult{res: res, err: err}
		}()

		waitForPendingCount(t, b, 1)

		req := httptest.NewRequestWithContext(t.Context(),
			http.MethodGet,
			b.RedirectURL()+"?state="+state+"&error=access_denied&error_description=denied",
			http.NoBody,
		)
		req.Host = b.redirectHost

		rr := httptest.NewRecorder()
		b.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("ServeHTTP status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "FlexiGPT authorization was not completed") {
			t.Fatalf("unexpected response body: %q", rr.Body.String())
		}

		select {
		case got := <-resultCh:
			if got.res != nil {
				t.Fatalf("result = %#v, want nil", got.res)
			}
			if got.err == nil || !strings.Contains(got.err.Error(), "access_denied") {
				t.Fatalf("err = %v, want substring %q", got.err, "access_denied")
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for authorization error")
		}

		if got := b.Pending(); len(got) != 0 {
			t.Fatalf("Pending len after error = %d, want 0", len(got))
		}
	})

	t.Run("pending list is sorted by serverID", func(t *testing.T) {
		ctx := t.Context()
		b, err := NewOAuthLoopbackBroker(ctx, &OAuthLoopbackBrokerOptions{
			TTL:          time.Minute,
			CallbackPath: "/mcp/oauth/callback",
		})
		if err != nil {
			t.Fatalf("NewOAuthLoopbackBroker: %v", err)
		}
		t.Cleanup(func() { _ = b.Close() })

		type fetchResult struct {
			serverID spec.MCPServerID
			err      error
		}
		resultCh := make(chan fetchResult, 2)

		startFetch := func(serverID spec.MCPServerID, state string) {
			go func() {
				_, err := b.FetchAuthorizationCode(ctx, OAuthAuthorizationRequest{
					ServerID:         serverID,
					AuthorizationURL: "https://issuer.test/authorize?state=" + state,
				})
				resultCh <- fetchResult{serverID: serverID, err: err}
			}()
		}

		startFetch("z-server", "state-z")
		startFetch("a-server", "state-a")

		waitForPendingCount(t, b, 2)

		pending := b.Pending()
		if len(pending) != 2 {
			t.Fatalf("Pending len = %d, want 2", len(pending))
		}
		if pending[0].ServerID != "a-server" || pending[1].ServerID != "z-server" {
			t.Fatalf("Pending order = %#v, want a-server then z-server", pending)
		}

		if !b.Cancel("a-server") {
			t.Fatalf("Cancel(a-server) = false, want true")
		}
		if !b.Cancel("z-server") {
			t.Fatalf("Cancel(z-server) = false, want true")
		}

		for range 2 {
			select {
			case got := <-resultCh:
				if got.err == nil || !strings.Contains(got.err.Error(), "cancelled") {
					t.Fatalf("fetch result for %s = %v, want cancelled error", got.serverID, got.err)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("timed out waiting for cancelled fetch result")
			}
		}
	})

	t.Run("expiry returns error", func(t *testing.T) {
		ctx := t.Context()
		b, err := NewOAuthLoopbackBroker(ctx, &OAuthLoopbackBrokerOptions{
			TTL:          20 * time.Millisecond,
			CallbackPath: "/mcp/oauth/callback",
		})
		if err != nil {
			t.Fatalf("NewOAuthLoopbackBroker: %v", err)
		}
		t.Cleanup(func() { _ = b.Close() })

		resultCh := make(chan oauthAuthorizationResult, 1)
		go func() {
			res, err := b.FetchAuthorizationCode(ctx, OAuthAuthorizationRequest{
				ServerID:         "expiry-server",
				AuthorizationURL: "https://issuer.test/authorize?state=expiry-state",
			})
			resultCh <- oauthAuthorizationResult{res: res, err: err}
		}()

		select {
		case got := <-resultCh:
			if got.res != nil {
				t.Fatalf("result = %#v, want nil", got.res)
			}
			if got.err == nil || !strings.Contains(got.err.Error(), "expired") {
				t.Fatalf("err = %v, want substring %q", got.err, "expired")
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for expiry")
		}

		if got := b.Pending(); len(got) != 0 {
			t.Fatalf("Pending len after expiry = %d, want 0", len(got))
		}
	})
}

func TestOAuthLoopbackBrokerServeHTTPValidation(t *testing.T) {
	ctx := t.Context()
	b, err := NewOAuthLoopbackBroker(ctx, &OAuthLoopbackBrokerOptions{
		TTL:          time.Minute,
		CallbackPath: "/mcp/oauth/callback",
	})
	if err != nil {
		t.Fatalf("NewOAuthLoopbackBroker: %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })

	t.Run("invalid callback host", func(t *testing.T) {
		req := httptest.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			b.RedirectURL()+"?state=x&code=y",
			http.NoBody,
		)
		req.Host = "evil.test"

		rr := httptest.NewRecorder()
		b.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "invalid OAuth callback host") {
			t.Fatalf("body = %q, want invalid host error", rr.Body.String())
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, b.RedirectURL(), http.NoBody)
		req.Host = b.redirectHost

		rr := httptest.NewRecorder()
		b.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("missing state", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, b.RedirectURL(), http.NoBody)
		req.Host = b.redirectHost

		rr := httptest.NewRecorder()
		b.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "missing OAuth state") {
			t.Fatalf("body = %q, want missing state error", rr.Body.String())
		}
	})

	t.Run("unknown state", func(t *testing.T) {
		req := httptest.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			b.RedirectURL()+"?state=unknown&code=test-code",
			http.NoBody,
		)
		req.Host = b.redirectHost

		rr := httptest.NewRecorder()
		b.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
		if !strings.Contains(rr.Body.String(), "unknown or expired OAuth state") {
			t.Fatalf("body = %q, want unknown/expired state error", rr.Body.String())
		}
	})

	t.Run("wrong path", func(t *testing.T) {
		req := httptest.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			"http://"+b.redirectHost+"/wrong?state=x",
			http.NoBody,
		)
		req.Host = b.redirectHost

		rr := httptest.NewRecorder()
		b.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})
}

func waitForPendingCount(t *testing.T, b *OAuthLoopbackBroker, want int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := len(b.Pending()); got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for pending count %d; got %d", want, len(b.Pending()))
}
