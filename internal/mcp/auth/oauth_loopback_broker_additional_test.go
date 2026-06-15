package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestOAuthLoopbackBrokerFetchAuthorizationCodeValidation(t *testing.T) {
	ctx := t.Context()
	b, err := NewOAuthLoopbackBroker(ctx, &OAuthLoopbackBrokerOptions{
		TTL:          time.Minute,
		CallbackPath: "/mcp/oauth/callback",
	})
	if err != nil {
		t.Fatalf("NewOAuthLoopbackBroker: %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })

	validReq := OAuthAuthorizationRequest{
		BundleID:         testBundleID,
		ServerID:         "server-a",
		AuthorizationURL: "https://issuer.test/authorize?state=state-a",
	}

	tests := []struct {
		name            string
		broker          *OAuthLoopbackBroker
		req             OAuthAuthorizationRequest
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:            "nil broker",
			broker:          nil,
			req:             validReq,
			wantErrIs:       spec.ErrMCPAuthRequired,
			wantErrContains: "OAuth loopback broker is not configured",
		},
		{
			name:   "missing bundleID",
			broker: b,
			req: OAuthAuthorizationRequest{
				ServerID:         validReq.ServerID,
				AuthorizationURL: validReq.AuthorizationURL,
			},
			wantErrIs:       spec.ErrMCPInvalidRequest,
			wantErrContains: "OAuth bundleID required",
		},
		{
			name:   "missing serverID",
			broker: b,
			req: OAuthAuthorizationRequest{
				BundleID:         validReq.BundleID,
				AuthorizationURL: validReq.AuthorizationURL,
			},
			wantErrIs:       spec.ErrMCPInvalidRequest,
			wantErrContains: "OAuth serverID required",
		},
		{
			name:   "missing authorization URL",
			broker: b,
			req: OAuthAuthorizationRequest{
				BundleID: validReq.BundleID,
				ServerID: validReq.ServerID,
			},
			wantErrIs:       spec.ErrMCPAuthRequired,
			wantErrContains: "OAuth authorization URL required",
		},
		{
			name:   "authorization URL missing state",
			broker: b,
			req: OAuthAuthorizationRequest{
				BundleID:         validReq.BundleID,
				ServerID:         validReq.ServerID,
				AuthorizationURL: "https://issuer.test/authorize",
			},
			wantErrIs:       spec.ErrMCPAuthRequired,
			wantErrContains: "missing state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.broker.FetchAuthorizationCode(ctx, tt.req)
			if err == nil {
				t.Fatalf("FetchAuthorizationCode succeeded, want error")
			}
			if !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("errors.Is(err, %v) = false, err=%v", tt.wantErrIs, err)
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestOAuthLoopbackBrokerHostHelpers(t *testing.T) {
	t.Run("splitOAuthHostPort", func(t *testing.T) {
		tests := []struct {
			name     string
			hostport string
			wantHost string
			wantPort string
			wantOK   bool
		}{
			{name: "host and port", hostport: "localhost:8080", wantHost: "localhost", wantPort: "8080", wantOK: true},
			{name: "host only", hostport: "localhost", wantHost: "localhost", wantPort: "", wantOK: true},
			{name: "ipv6 loopback", hostport: "[::1]:8080", wantHost: "::1", wantPort: "8080", wantOK: true},
			{name: "invalid multi-colon", hostport: "bad:port:extra", wantOK: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				host, port, ok := splitOAuthHostPort(tt.hostport)
				if host != tt.wantHost || port != tt.wantPort || ok != tt.wantOK {
					t.Fatalf(
						"splitOAuthHostPort(%q) = (%q, %q, %v), want (%q, %q, %v)",
						tt.hostport,
						host,
						port,
						ok,
						tt.wantHost,
						tt.wantPort,
						tt.wantOK,
					)
				}
			})
		}
	})

	t.Run("sameOAuthLoopbackEndpoint", func(t *testing.T) {
		tests := []struct {
			name         string
			callbackHost string
			redirectHost string
			want         bool
		}{
			{name: "same host and port", callbackHost: "127.0.0.1:37033", redirectHost: "127.0.0.1:37033", want: true},
			{name: "localhost alias", callbackHost: "localhost:37033", redirectHost: "127.0.0.1:37033", want: true},
			{name: "different port", callbackHost: "localhost:37034", redirectHost: "127.0.0.1:37033", want: false},
			{
				name:         "non loopback host",
				callbackHost: "example.com:37033",
				redirectHost: "127.0.0.1:37033",
				want:         false,
			},
			{name: "missing port", callbackHost: "localhost", redirectHost: "127.0.0.1:37033", want: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := sameOAuthLoopbackEndpoint(tt.callbackHost, tt.redirectHost); got != tt.want {
					t.Fatalf(
						"sameOAuthLoopbackEndpoint(%q, %q) = %v, want %v",
						tt.callbackHost,
						tt.redirectHost,
						got,
						tt.want,
					)
				}
			})
		}
	})
}

func TestOAuthLoopbackBrokerNilReceiverAndServeHTTPValidation(t *testing.T) {
	var b *OAuthLoopbackBroker
	if _, err := b.FetchAuthorizationCode(
		t.Context(),
		OAuthAuthorizationRequest{
			BundleID:         testBundleID,
			ServerID:         "server",
			AuthorizationURL: "https://issuer.test/authorize?state=s",
		},
	); err == nil ||
		!strings.Contains(err.Error(), "not configured") {
		t.Fatalf("nil broker fetch error = %v, want broker not configured", err)
	}

	ctx := t.Context()
	broker, err := NewOAuthLoopbackBroker(ctx, &OAuthLoopbackBrokerOptions{
		TTL:          time.Minute,
		CallbackPath: "/mcp/oauth/callback",
	})
	if err != nil {
		t.Fatalf("NewOAuthLoopbackBroker: %v", err)
	}
	t.Cleanup(func() { _ = broker.Close() })

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, broker.RedirectURL(), http.NoBody)
	req.Host = broker.redirectHost

	rr := httptest.NewRecorder()
	broker.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
