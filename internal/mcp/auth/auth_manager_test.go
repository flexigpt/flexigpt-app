package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	testMCPResource          = "https://example.test/mcp"
	testPublicRefRaw         = `{"clientID":"public-client"}`
	testPublicClientID       = "public-client"
	testConfidentialClientID = "confidential-client"
	testSecretValue          = "secret-value"
	testResolvedSecret       = "resolved-secret"
	testStdIOServerID        = "stdio-server"
	testHTTPServerID         = "http-server"
)

func TestNormalizeHTTPAuthMode(t *testing.T) {
	tests := []struct {
		name string
		in   spec.MCPHTTPAuthMode
		want spec.MCPHTTPAuthMode
	}{
		{
			name: "blank becomes none",
			in:   "",
			want: spec.MCPHTTPAuthNone,
		},
		{
			name: "trim oauth",
			in:   "  oauth  ",
			want: spec.MCPHTTPAuthOAuth,
		},
		{
			name: "trim client credentials",
			in:   " clientCredentials ",
			want: spec.MCPHTTPAuthClientCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeHTTPAuthMode(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeHTTPAuthMode(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseAndValidateOAuthClientCredentialsSecret(t *testing.T) {
	tests := []struct {
		name            string
		raw             string
		requireSecret   bool
		wantClientID    string
		wantSecret      string
		wantSensitive   []string
		wantErrContains string
	}{
		{
			name:          testPublicClientID,
			raw:           testPublicRefRaw,
			requireSecret: false,
			wantClientID:  testPublicClientID,
			wantSecret:    "",
			wantSensitive: []string{},
		},
		{
			name:          "public client with optional secret",
			raw:           `{"clientID":"public-client","clientSecret":"secret-value"}`,
			requireSecret: false,
			wantClientID:  testPublicClientID,
			wantSecret:    testSecretValue,
			wantSensitive: []string{testSecretValue},
		},
		{
			name:          "confidential client",
			raw:           `{"clientID":"confidential-client","clientSecret":"secret-value"}`,
			requireSecret: true,
			wantClientID:  testConfidentialClientID,
			wantSecret:    testSecretValue,
			wantSensitive: []string{testSecretValue},
		},
		{
			name:            "missing clientID",
			raw:             `{"clientSecret":"secret-value"}`,
			requireSecret:   false,
			wantErrContains: "requires clientID",
		},
		{
			name:            "clientID whitespace",
			raw:             `{"clientID":" public-client "}`,
			requireSecret:   false,
			wantErrContains: "clientID must not have leading/trailing whitespace",
		},
		{
			name:            "missing clientSecret when required",
			raw:             `{"clientID":"confidential-client"}`,
			requireSecret:   true,
			wantErrContains: "requires clientSecret",
		},
		{
			name:            "clientSecret whitespace only",
			raw:             `{"clientID":"confidential-client","clientSecret":"   "}`,
			requireSecret:   true,
			wantErrContains: "clientSecret must not be only whitespace",
		},
		{
			name:            "unknown field",
			raw:             `{"clientID":"public-client","extra":true}`,
			requireSecret:   false,
			wantErrContains: "JSON object",
		},
		{
			name:            "multiple objects",
			raw:             `{"clientID":"public-client"}{"clientID":"other"}`,
			requireSecret:   false,
			wantErrContains: "single JSON object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, sensitive, err := parseOAuthClientCredentialsSecret(tt.raw, tt.requireSecret)
			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf(
						"parseOAuthClientCredentialsSecret succeeded, want error containing %q",
						tt.wantErrContains,
					)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				if err := ValidateOAuthClientCredentialsSecret(tt.raw, tt.requireSecret); err == nil ||
					!strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf(
						"ValidateOAuthClientCredentialsSecret error = %v, want substring %q",
						err,
						tt.wantErrContains,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseOAuthClientCredentialsSecret: %v", err)
			}
			if creds == nil {
				t.Fatalf("creds is nil")
			}
			if creds.ClientID != tt.wantClientID {
				t.Fatalf("ClientID = %q, want %q", creds.ClientID, tt.wantClientID)
			}
			if tt.wantSecret == "" {
				if creds.ClientSecretAuth != nil {
					t.Fatalf("ClientSecretAuth = %#v, want nil", creds.ClientSecretAuth)
				}
			} else {
				if creds.ClientSecretAuth == nil {
					t.Fatalf("ClientSecretAuth is nil, want secret")
				}
				if creds.ClientSecretAuth.ClientSecret != tt.wantSecret {
					t.Fatalf("ClientSecret = %q, want %q", creds.ClientSecretAuth.ClientSecret, tt.wantSecret)
				}
			}
			if len(sensitive) != len(tt.wantSensitive) {
				t.Fatalf("SensitiveValues len = %d, want %d", len(sensitive), len(tt.wantSensitive))
			}
			for i := range tt.wantSensitive {
				if sensitive[i] != tt.wantSensitive[i] {
					t.Fatalf("SensitiveValues[%d] = %q, want %q", i, sensitive[i], tt.wantSensitive[i])
				}
			}
			if err := ValidateOAuthClientCredentialsSecret(tt.raw, tt.requireSecret); err != nil {
				t.Fatalf("ValidateOAuthClientCredentialsSecret: %v", err)
			}
		})
	}
}

func TestResolveOAuthClientCredentials(t *testing.T) {
	const (
		publicRef       = "public-ref"
		confidentialRef = "confidential-ref"
		missingRef      = "missing-ref"
	)

	publicRaw := testPublicRefRaw
	confidentialRaw := `{"clientID":"confidential-client","clientSecret":"secret-value"}`

	tests := []struct {
		name            string
		resolver        SecretResolver
		ref             string
		requireSecret   bool
		wantClientID    string
		wantSecret      string
		wantSensitive   []string
		wantErrContains string
	}{
		{
			name:          testPublicClientID,
			resolver:      StaticSecretResolver{publicRef: publicRaw},
			ref:           publicRef,
			requireSecret: false,
			wantClientID:  testPublicClientID,
			wantSensitive: []string{publicRaw},
		},
		{
			name:          "confidential client",
			resolver:      StaticSecretResolver{confidentialRef: confidentialRaw},
			ref:           confidentialRef,
			requireSecret: true,
			wantClientID:  testConfidentialClientID,
			wantSecret:    testSecretValue,
			wantSensitive: []string{confidentialRaw, testSecretValue},
		},
		{
			name:            "empty ref",
			resolver:        StaticSecretResolver{publicRef: publicRaw},
			ref:             "",
			requireSecret:   false,
			wantErrContains: "clientCredentialRef is required",
		},
		{
			name:            "nil resolver",
			resolver:        nil,
			ref:             publicRef,
			requireSecret:   false,
			wantErrContains: "secret resolver is not configured",
		},
		{
			name:            "missing ref",
			resolver:        StaticSecretResolver{publicRef: publicRaw},
			ref:             missingRef,
			requireSecret:   false,
			wantErrContains: "secret ref not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, sensitive, err := resolveOAuthClientCredentials(
				t.Context(),
				tt.resolver,
				tt.ref,
				tt.requireSecret,
			)
			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("resolveOAuthClientCredentials succeeded, want error containing %q", tt.wantErrContains)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveOAuthClientCredentials: %v", err)
			}
			if creds == nil {
				t.Fatalf("creds is nil")
			}
			if creds.ClientID != tt.wantClientID {
				t.Fatalf("ClientID = %q, want %q", creds.ClientID, tt.wantClientID)
			}
			if tt.wantSecret == "" {
				if creds.ClientSecretAuth != nil {
					t.Fatalf("ClientSecretAuth = %#v, want nil", creds.ClientSecretAuth)
				}
			} else {
				if creds.ClientSecretAuth == nil {
					t.Fatalf("ClientSecretAuth is nil, want secret")
				}
				if creds.ClientSecretAuth.ClientSecret != tt.wantSecret {
					t.Fatalf("ClientSecret = %q, want %q", creds.ClientSecretAuth.ClientSecret, tt.wantSecret)
				}
			}
			if len(sensitive) != len(tt.wantSensitive) {
				t.Fatalf("SensitiveValues len = %d, want %d", len(sensitive), len(tt.wantSensitive))
			}
			for i := range tt.wantSensitive {
				if sensitive[i] != tt.wantSensitive[i] {
					t.Fatalf("SensitiveValues[%d] = %q, want %q", i, sensitive[i], tt.wantSensitive[i])
				}
			}
		})
	}
}

func TestPrepareTransportAuthSuccessCases(t *testing.T) {
	tests := []struct {
		name          string
		mgr           *AuthManager
		cfg           spec.MCPServerConfig
		wantEnv       map[string]string
		wantSensitive []string
		wantStatus    spec.MCPAuthStatus
	}{
		{
			name: "stdio secret env refs",
			mgr:  NewAuthManager(StaticSecretResolver{"stdio-secret-ref": testResolvedSecret}),
			cfg: spec.MCPServerConfig{
				ID:        testStdIOServerID,
				Transport: spec.MCPTransportStdio,
				Stdio: &spec.MCPStdioConfig{
					Command: "server-binary",
					//nolint:gosec // Test.
					SecretEnvRefs: map[string]string{
						"TOKEN": "stdio-secret-ref",
					},
				},
			},
			wantEnv:       map[string]string{"TOKEN": testResolvedSecret},
			wantSensitive: []string{testResolvedSecret},
			wantStatus: spec.MCPAuthStatus{
				ServerID: testStdIOServerID,
				AuthMode: spec.MCPHTTPAuthNone,
				State:    spec.MCPAuthStateNotRequired,
			},
		},
		{
			name: "streamable http none",
			mgr:  NewAuthManager(nil),
			cfg: spec.MCPServerConfig{
				ID:        testHTTPServerID,
				Transport: spec.MCPTransportStreamableHTTP,
				StreamableHTTP: &spec.MCPStreamableHTTPConfig{
					URL:      " https://example.test/mcp ",
					AuthMode: "",
				},
			},
			wantEnv:       map[string]string{},
			wantSensitive: []string{},
			wantStatus: spec.MCPAuthStatus{
				ServerID: testHTTPServerID,
				AuthMode: spec.MCPHTTPAuthNone,
				State:    spec.MCPAuthStateNotRequired,
				Resource: testMCPResource,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.mgr.PrepareTransportAuth(t.Context(), tt.cfg)
			if err != nil {
				t.Fatalf("PrepareTransportAuth: %v", err)
			}
			if got.OAuthHandler != nil {
				t.Fatalf("OAuthHandler = %#v, want nil", got.OAuthHandler)
			}
			if len(got.Env) != len(tt.wantEnv) {
				t.Fatalf("Env len = %d, want %d", len(got.Env), len(tt.wantEnv))
			}
			for k, want := range tt.wantEnv {
				if got.Env[k] != want {
					t.Fatalf("Env[%q] = %q, want %q", k, got.Env[k], want)
				}
			}
			if len(got.SensitiveValues) != len(tt.wantSensitive) {
				t.Fatalf("SensitiveValues len = %d, want %d", len(got.SensitiveValues), len(tt.wantSensitive))
			}
			for i := range tt.wantSensitive {
				if got.SensitiveValues[i] != tt.wantSensitive[i] {
					t.Fatalf("SensitiveValues[%d] = %q, want %q", i, got.SensitiveValues[i], tt.wantSensitive[i])
				}
			}

			st, ok := tt.mgr.GetAuthStatus(tt.cfg.ID)
			if !ok {
				t.Fatalf("missing auth status")
			}
			if st.ServerID != tt.wantStatus.ServerID {
				t.Fatalf("ServerID = %q, want %q", st.ServerID, tt.wantStatus.ServerID)
			}
			if st.AuthMode != tt.wantStatus.AuthMode {
				t.Fatalf("AuthMode = %q, want %q", st.AuthMode, tt.wantStatus.AuthMode)
			}
			if st.State != tt.wantStatus.State {
				t.Fatalf("State = %q, want %q", st.State, tt.wantStatus.State)
			}
			if st.Resource != tt.wantStatus.Resource {
				t.Fatalf("Resource = %q, want %q", st.Resource, tt.wantStatus.Resource)
			}
		})
	}
}

func TestPrepareTransportAuthErrorCases(t *testing.T) {
	tests := []struct {
		name            string
		cfg             spec.MCPServerConfig
		wantErrIs       error
		wantErrContains string
		wantStatus      spec.MCPAuthStatus
	}{
		{
			name: errStrMissingStdIOConfig,
			cfg: spec.MCPServerConfig{
				ID:        testStdIOServerID,
				Transport: spec.MCPTransportStdio,
			},
			wantErrIs:       spec.ErrMCPInvalidRequest,
			wantErrContains: errStrMissingStdIOConfig,
			wantStatus: spec.MCPAuthStatus{
				ServerID: testStdIOServerID,
				AuthMode: spec.MCPHTTPAuthNone,
				State:    spec.MCPAuthStateError,
			},
		},
		{
			name: "missing streamable http config",
			cfg: spec.MCPServerConfig{
				ID:        testHTTPServerID,
				Transport: spec.MCPTransportStreamableHTTP,
			},
			wantErrIs:       spec.ErrMCPInvalidRequest,
			wantErrContains: "missing streamableHttp config",
			wantStatus: spec.MCPAuthStatus{
				ServerID: testHTTPServerID,
				AuthMode: spec.MCPHTTPAuthNone,
				State:    spec.MCPAuthStateError,
			},
		},
		{
			name: errStrUnsupportedTransport,
			cfg: spec.MCPServerConfig{
				ID:        "bad-server",
				Transport: spec.MCPTransportType("bogus"),
			},
			wantErrIs:       spec.ErrMCPInvalidRequest,
			wantErrContains: errStrUnsupportedTransport,
			wantStatus: spec.MCPAuthStatus{
				ServerID: "bad-server",
				AuthMode: spec.MCPHTTPAuthNone,
				State:    spec.MCPAuthStateError,
			},
		},
		{
			name: "oauth missing broker",
			cfg: spec.MCPServerConfig{
				ID:        "oauth-server",
				Transport: spec.MCPTransportStreamableHTTP,
				StreamableHTTP: &spec.MCPStreamableHTTPConfig{
					URL:      testMCPResource,
					AuthMode: spec.MCPHTTPAuthOAuth,
				},
			},
			wantErrIs:       spec.ErrMCPAuthRequired,
			wantErrContains: "OAuth authorization code flow is not configured",
			wantStatus: spec.MCPAuthStatus{
				ServerID: "oauth-server",
				AuthMode: spec.MCPHTTPAuthOAuth,
				State:    spec.MCPAuthStateRequired,
				Resource: testMCPResource,
			},
		},
		{
			name: "client credentials missing ref",
			cfg: spec.MCPServerConfig{
				ID:        "cc-server",
				Transport: spec.MCPTransportStreamableHTTP,
				StreamableHTTP: &spec.MCPStreamableHTTPConfig{
					URL:      testMCPResource,
					AuthMode: spec.MCPHTTPAuthClientCredentials,
				},
			},
			wantErrIs:       spec.ErrMCPAuthRequired,
			wantErrContains: "clientCredentialRef is required",
			wantStatus: spec.MCPAuthStatus{
				ServerID: "cc-server",
				AuthMode: spec.MCPHTTPAuthClientCredentials,
				State:    spec.MCPAuthStateRequired,
				Resource: testMCPResource,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewAuthManager(nil)
			_, err := mgr.PrepareTransportAuth(t.Context(), tt.cfg)
			if err == nil {
				t.Fatalf("PrepareTransportAuth succeeded, want error")
			}
			if !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("errors.Is(err, %v) = false, err=%v", tt.wantErrIs, err)
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrContains)
			}

			st, ok := mgr.GetAuthStatus(tt.cfg.ID)
			if !ok {
				t.Fatalf("missing auth status")
			}
			if st.ServerID != tt.wantStatus.ServerID {
				t.Fatalf("ServerID = %q, want %q", st.ServerID, tt.wantStatus.ServerID)
			}
			if st.AuthMode != tt.wantStatus.AuthMode {
				t.Fatalf("AuthMode = %q, want %q", st.AuthMode, tt.wantStatus.AuthMode)
			}
			if st.State != tt.wantStatus.State {
				t.Fatalf("State = %q, want %q", st.State, tt.wantStatus.State)
			}
			if tt.wantStatus.Resource != "" && st.Resource != tt.wantStatus.Resource {
				t.Fatalf("Resource = %q, want %q", st.Resource, tt.wantStatus.Resource)
			}
			if !strings.Contains(st.LastError, tt.wantErrContains) {
				t.Fatalf("LastError = %q, want substring %q", st.LastError, tt.wantErrContains)
			}
		})
	}
}

func TestPrepareTransportAuthNilManager(t *testing.T) {
	var mgr *AuthManager

	got, err := mgr.PrepareTransportAuth(t.Context(), spec.MCPServerConfig{ID: "nil-manager"})
	if err != nil {
		t.Fatalf("PrepareTransportAuth: %v", err)
	}
	if got.OAuthHandler != nil {
		t.Fatalf("OAuthHandler = %#v, want nil", got.OAuthHandler)
	}
	if len(got.Env) != 0 {
		t.Fatalf("Env len = %d, want 0", len(got.Env))
	}
	if got.Status.ServerID != "nil-manager" {
		t.Fatalf("ServerID = %q, want %q", got.Status.ServerID, "nil-manager")
	}
	if got.Status.AuthMode != spec.MCPHTTPAuthNone {
		t.Fatalf("AuthMode = %q, want none", got.Status.AuthMode)
	}
	if got.Status.State != spec.MCPAuthStateNotRequired {
		t.Fatalf("State = %q, want notRequired", got.Status.State)
	}
}
