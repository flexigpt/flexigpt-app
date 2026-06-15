package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestPrepareTransportAuthAPIKeyAndSecretHeaderValidation(t *testing.T) {
	//nolint:gosec // Test.
	const apiKeyRef = "api-key-ref"

	tests := []struct {
		name            string
		mgr             *AuthManager
		cfg             spec.MCPServerConfig
		wantStatus      spec.MCPAuthStatus
		wantHeaders     map[string]string
		wantSensitive   []string
		wantErrIs       error
		wantErrContains string
	}{
		{
			name: "api key success",
			mgr:  NewAuthManager(StaticSecretResolver{apiKeyRef: "api-secret"}),
			cfg: spec.MCPServerConfig{
				BundleID:  testBundleID,
				ID:        testHTTPServerID,
				Transport: spec.MCPTransportStreamableHTTP,
				StreamableHTTP: &spec.MCPStreamableHTTPConfig{
					URL:      " https://example.test/mcp ",
					AuthMode: spec.MCPHTTPAuthAPIKey,
					Headers: map[string]string{
						"X-Trace": "trace",
					},
					SecretHeaderRefs: map[string]string{
						"X-API-Key": apiKeyRef,
					},
				},
			},
			wantStatus: spec.MCPAuthStatus{
				BundleID: testBundleID,
				ServerID: testHTTPServerID,
				AuthMode: spec.MCPHTTPAuthAPIKey,
				State:    spec.MCPAuthStateAuthorized,
				Resource: testMCPResource,
			},
			wantHeaders: map[string]string{
				"X-Trace":   "trace",
				"X-API-Key": "api-secret",
			},
			wantSensitive: []string{"api-secret"},
		},
		{
			name: "api key requires secret header refs",
			mgr:  NewAuthManager(nil),
			cfg: spec.MCPServerConfig{
				BundleID:  testBundleID,
				ID:        "api-key-missing-refs",
				Transport: spec.MCPTransportStreamableHTTP,
				StreamableHTTP: &spec.MCPStreamableHTTPConfig{
					URL:      testMCPResource,
					AuthMode: spec.MCPHTTPAuthAPIKey,
				},
			},
			wantStatus: spec.MCPAuthStatus{
				BundleID: testBundleID,
				ServerID: "api-key-missing-refs",
				AuthMode: spec.MCPHTTPAuthAPIKey,
				State:    spec.MCPAuthStateRequired,
				Resource: testMCPResource,
			},
			wantErrIs:       spec.ErrMCPAuthRequired,
			wantErrContains: errStrAPIKeyHeaderRequired,
		},
		{
			name: "api key empty secret is rejected",
			mgr:  NewAuthManager(StaticSecretResolver{"empty-ref": ""}),
			cfg: spec.MCPServerConfig{
				BundleID:  testBundleID,
				ID:        "api-key-empty-secret",
				Transport: spec.MCPTransportStreamableHTTP,
				StreamableHTTP: &spec.MCPStreamableHTTPConfig{
					URL:      testMCPResource,
					AuthMode: spec.MCPHTTPAuthAPIKey,
					SecretHeaderRefs: map[string]string{
						"X-API-Key": "empty-ref",
					},
				},
			},
			wantStatus: spec.MCPAuthStatus{
				BundleID: testBundleID,
				ServerID: "api-key-empty-secret",
				AuthMode: spec.MCPHTTPAuthAPIKey,
				State:    spec.MCPAuthStateRequired,
				Resource: testMCPResource,
			},
			wantErrIs:       spec.ErrMCPAuthRequired,
			wantErrContains: "secret HTTP header X-API-Key is empty",
		},
		{
			name: "api key secret must not contain newlines",
			mgr:  NewAuthManager(StaticSecretResolver{"bad-ref": "bad\nvalue"}),
			cfg: spec.MCPServerConfig{
				BundleID:  testBundleID,
				ID:        "api-key-newline-secret",
				Transport: spec.MCPTransportStreamableHTTP,
				StreamableHTTP: &spec.MCPStreamableHTTPConfig{
					URL:      testMCPResource,
					AuthMode: spec.MCPHTTPAuthAPIKey,
					SecretHeaderRefs: map[string]string{
						"X-API-Key": "bad-ref",
					},
				},
			},
			wantStatus: spec.MCPAuthStatus{
				BundleID: testBundleID,
				ServerID: "api-key-newline-secret",
				AuthMode: spec.MCPHTTPAuthAPIKey,
				State:    spec.MCPAuthStateError,
				Resource: testMCPResource,
			},
			wantErrIs:       spec.ErrMCPInvalidRequest,
			wantErrContains: "must not contain CR, LF, or NUL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.mgr.PrepareTransportAuth(t.Context(), tt.cfg)
			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("PrepareTransportAuth succeeded, want error containing %q", tt.wantErrContains)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("errors.Is(err, %v) = false, err=%v", tt.wantErrIs, err)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrContains)
				}
			} else if err != nil {
				t.Fatalf("PrepareTransportAuth: %v", err)
			}

			assertAuthStatusCore(t, got.Status, tt.wantStatus)
			if len(tt.wantSensitive) != len(got.SensitiveValues) {
				t.Fatalf("SensitiveValues len = %d, want %d", len(got.SensitiveValues), len(tt.wantSensitive))
			}
			for i := range tt.wantSensitive {
				if got.SensitiveValues[i] != tt.wantSensitive[i] {
					t.Fatalf("SensitiveValues[%d] = %q, want %q", i, got.SensitiveValues[i], tt.wantSensitive[i])
				}
			}
			for k, want := range tt.wantHeaders {
				if got.Headers[k] != want {
					t.Fatalf("Headers[%q] = %q, want %q", k, got.Headers[k], want)
				}
			}
			if tt.wantHeaders != nil && len(got.Headers) != len(tt.wantHeaders) {
				t.Fatalf("Headers len = %d, want %d", len(got.Headers), len(tt.wantHeaders))
			}
			if got.OAuthHandler != nil {
				t.Fatalf("OAuthHandler = %#v, want nil", got.OAuthHandler)
			}
		})
	}
}

func TestResolveOAuthClientCredentialsTrimsRefAndReturnsRedactionValues(t *testing.T) {
	const raw = `{"clientID":"service-client","clientSecret":"service-secret"}`

	creds, sensitive, err := resolveOAuthClientCredentials(
		t.Context(),
		StaticSecretResolver{"service-ref": raw},
		"  service-ref  ",
		true,
	)
	if err != nil {
		t.Fatalf("resolveOAuthClientCredentials: %v", err)
	}
	if creds == nil {
		t.Fatalf("creds is nil")
	}
	if creds.ClientID != "service-client" {
		t.Fatalf("ClientID = %q, want %q", creds.ClientID, "service-client")
	}
	if creds.ClientSecretAuth == nil || creds.ClientSecretAuth.ClientSecret != "service-secret" {
		t.Fatalf("ClientSecretAuth = %#v, want secret", creds.ClientSecretAuth)
	}
	if len(sensitive) != 2 {
		t.Fatalf("SensitiveValues len = %d, want 2", len(sensitive))
	}
	if sensitive[0] != raw || sensitive[1] != "service-secret" {
		t.Fatalf("SensitiveValues = %#v, want [raw, secret]", sensitive)
	}
}

func TestParseOAuthClientCredentialsSecretRejectsWhitespaceOnlyOptionalSecret(t *testing.T) {
	_, _, err := parseOAuthClientCredentialsSecret(`{"clientID":"public-client","clientSecret":"   "}`, false)
	if err == nil {
		t.Fatalf("parseOAuthClientCredentialsSecret succeeded, want error")
	}
	if !strings.Contains(err.Error(), "clientSecret must not be only whitespace") {
		t.Fatalf("err = %q, want whitespace secret error", err.Error())
	}
}

func TestSaveAuthStatusValidationAndNilManager(t *testing.T) {
	var nilMgr *AuthManager
	if err := nilMgr.SaveAuthStatus(
		t.Context(),
		spec.MCPAuthStatus{BundleID: testBundleID, ServerID: testHTTPServerID},
	); err != nil {
		t.Fatalf("nil manager SaveAuthStatus: %v", err)
	}
	if _, ok := nilMgr.GetAuthStatus(testBundleID, testHTTPServerID); ok {
		t.Fatalf("nil manager unexpectedly returned a status")
	}
	nilMgr.ClearAuthStatus(testBundleID, testHTTPServerID)
	nilMgr.ClearAuthStatuses()

	mgr := NewAuthManager(nil)
	tests := []struct {
		name            string
		status          spec.MCPAuthStatus
		wantErrContains string
	}{
		{
			name:            "missing bundleID",
			status:          spec.MCPAuthStatus{ServerID: testHTTPServerID},
			wantErrContains: "bundleID required",
		},
		{
			name:            "missing serverID",
			status:          spec.MCPAuthStatus{BundleID: testBundleID},
			wantErrContains: "serverID required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mgr.SaveAuthStatus(
				t.Context(),
				tt.status,
			); err == nil ||
				!strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("SaveAuthStatus error = %v, want substring %q", err, tt.wantErrContains)
			}
		})
	}
}

func TestDefaultMCPAuthStatusFromConfigAPIKey(t *testing.T) {
	cfg := spec.MCPServerConfig{
		BundleID: testBundleID,
		ID:       testHTTPServerID,
		StreamableHTTP: &spec.MCPStreamableHTTPConfig{
			URL:      " https://example.test/mcp ",
			AuthMode: spec.MCPHTTPAuthAPIKey,
		},
	}

	got := DefaultMCPAuthStatusFromConfig(cfg)
	want := spec.MCPAuthStatus{
		BundleID: testBundleID,
		ServerID: testHTTPServerID,
		AuthMode: spec.MCPHTTPAuthAPIKey,
		State:    spec.MCPAuthStateRequired,
		Resource: testMCPResource,
	}
	assertAuthStatusCore(t, got, want)
}

func TestMergeMCPAuthStatusResourceMismatchReturnsDefault(t *testing.T) {
	cfg := spec.MCPServerConfig{
		BundleID: testBundleID,
		ID:       testHTTPServerID,
		StreamableHTTP: &spec.MCPStreamableHTTPConfig{
			URL:      testMCPResource,
			AuthMode: spec.MCPHTTPAuthOAuth,
		},
	}
	want := DefaultMCPAuthStatusFromConfig(cfg)
	got := MergeMCPAuthStatus(spec.MCPAuthStatus{
		BundleID: testBundleID,
		ServerID: testHTTPServerID,
		AuthMode: spec.MCPHTTPAuthOAuth,
		State:    spec.MCPAuthStateAuthorized,
		Resource: "https://example.test/other",
	}, cfg)

	assertAuthStatusCore(t, got, want)
}

func TestTrackedOAuthHandlerAuthorizeErrorPublishesRedactedStatus(t *testing.T) {
	sink := NewAuthManager(nil)
	inner := &fakeOAuthHandler{
		authorizeErr: errors.New("authorize failed top-secret"),
	}
	h := &trackedOAuthHandler{
		inner:           inner,
		sink:            sink,
		status:          trackedBaseStatus(testHTTPServerID),
		sensitiveValues: []string{"top-secret"},
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testMCPResource, http.NoBody)
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header: http.Header{
			"WWW-Authenticate": []string{`Bearer error="insufficient_scope", scope="scope-a scope-b"`},
		},
	}

	if err := h.Authorize(t.Context(), req, resp); err == nil {
		t.Fatalf("Authorize succeeded, want error")
	}
	if inner.authorizeCalls != 1 {
		t.Fatalf("Authorize calls = %d, want 1", inner.authorizeCalls)
	}
	if inner.tokenSourceCalls != 0 {
		t.Fatalf("TokenSource calls = %d, want 0", inner.tokenSourceCalls)
	}

	st, ok := sink.GetAuthStatus(testBundleID, testHTTPServerID)
	if !ok {
		t.Fatalf("missing auth status")
	}
	if st.State != spec.MCPAuthStateError {
		t.Fatalf("State = %q, want %q", st.State, spec.MCPAuthStateError)
	}
}
