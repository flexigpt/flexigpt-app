package store

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestStorePatchMCPSettingsAndValidation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test skipped on Windows")
	}
	ctx := t.Context()
	st, err := NewMCPStore(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	got, err := st.GetMCPSettings(ctx)
	if err != nil {
		t.Fatalf("GetMCPSettings: %v", err)
	}
	if got == nil {
		t.Fatal("GetMCPSettings returned nil")
	}
	if got.OAuthLoopbackListenAddr != "" {
		t.Fatalf("default OAuthLoopbackListenAddr = %q, want empty", got.OAuthLoopbackListenAddr)
	}

	addr := " 127.0.0.1:4567 "
	patched, err := st.PatchMCPSettings(ctx, &spec.PatchMCPSettingsRequest{
		Body: &spec.PatchMCPSettingsRequestBody{
			OAuthLoopbackListenAddr: &addr,
		},
	})
	if err != nil {
		t.Fatalf("PatchMCPSettings(valid): %v", err)
	}
	if patched.OAuthLoopbackListenAddr != "127.0.0.1:4567" {
		t.Fatalf("patched OAuthLoopbackListenAddr = %q, want %q", patched.OAuthLoopbackListenAddr, "127.0.0.1:4567")
	}

	reopened, err := NewMCPStore(ctx, t.TempDir())
	if err != nil {
		// This store is a different temp dir; the reopen check below uses the same dir.
		_ = reopened
	}

	reopenDir := t.TempDir()
	st2, err := NewMCPStore(ctx, reopenDir)
	if err != nil {
		t.Fatalf("NewMCPStore(reopen seed): %v", err)
	}
	if _, err := st2.PatchMCPSettings(ctx, &spec.PatchMCPSettingsRequest{
		Body: &spec.PatchMCPSettingsRequestBody{
			OAuthLoopbackListenAddr: &addr,
		},
	}); err != nil {
		t.Fatalf("PatchMCPSettings(reopen seed): %v", err)
	}
	if err := st2.Close(); err != nil {
		t.Fatalf("Close(reopen seed): %v", err)
	}

	st3, err := NewMCPStore(ctx, reopenDir)
	if err != nil {
		t.Fatalf("NewMCPStore(reopen): %v", err)
	}
	t.Cleanup(func() { _ = st3.Close() })

	reloaded, err := st3.GetMCPSettings(ctx)
	if err != nil {
		t.Fatalf("GetMCPSettings(reopen): %v", err)
	}
	if reloaded.OAuthLoopbackListenAddr != "127.0.0.1:4567" {
		t.Fatalf("reloaded OAuthLoopbackListenAddr = %q, want %q", reloaded.OAuthLoopbackListenAddr, "127.0.0.1:4567")
	}

	t.Run("validate helper", func(t *testing.T) {
		if err := validateMCPSettings(spec.MCPSettings{}); err != nil {
			t.Fatalf("validateMCPSettings(empty): %v", err)
		}

		tests := []struct {
			name            string
			settings        spec.MCPSettings
			wantErrContains string
		}{
			{
				name: "non-loopback host",
				settings: spec.MCPSettings{
					OAuthLoopbackListenAddr: "example.test:4567",
				},
				wantErrContains: "host must be loopback",
			},
			{
				name: "non-numeric port",
				settings: spec.MCPSettings{
					OAuthLoopbackListenAddr: "127.0.0.1:not-a-port",
				},
				wantErrContains: "port must be numeric",
			},
			{
				name: "out of range port",
				settings: spec.MCPSettings{
					OAuthLoopbackListenAddr: "127.0.0.1:0",
				},
				wantErrContains: "port must be 1..65535",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validateMCPSettings(tt.settings)
				if err == nil {
					t.Fatalf(
						"validateMCPSettings(%+v) succeeded, want error containing %q",
						tt.settings,
						tt.wantErrContains,
					)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
				}
			})
		}
	})
}

func TestStoreSetupDeclarationHelpers(t *testing.T) {
	tests := []struct {
		name       string
		setup      *spec.MCPServerSetup
		wantClient bool
		wantAPIKey bool
	}{
		{
			name:       "nil setup",
			setup:      nil,
			wantClient: false,
			wantAPIKey: false,
		},
		{
			name: "non-required inputs do not declare setup",
			setup: &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
				{
					ID:   "client",
					Kind: spec.MCPSetupKindOAuthClientCredentials,
					OAuthClientCredentials: &spec.MCPSetupOAuthClientCredentialsInput{
						ClientSecretRequired: true,
					},
				},
				{
					ID:   "header",
					Kind: spec.MCPSetupKindHTTPHeader,
					HTTPHeader: &spec.MCPSetupHTTPHeaderInput{
						HeaderName: "X-API-Key",
						Secret:     true,
					},
				},
			}},
			wantClient: false,
			wantAPIKey: false,
		},
		{
			name: "required oauth client credentials declared",
			setup: &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
				{
					ID:       "client",
					Kind:     spec.MCPSetupKindOAuthClientCredentials,
					Required: true,
					OAuthClientCredentials: &spec.MCPSetupOAuthClientCredentialsInput{
						ClientSecretRequired: true,
					},
				},
			}},
			wantClient: true,
			wantAPIKey: false,
		},
		{
			name: "required api key header declared",
			setup: &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
				{
					ID:       "header",
					Kind:     spec.MCPSetupKindHTTPHeader,
					Required: true,
					HTTPHeader: &spec.MCPSetupHTTPHeaderInput{
						HeaderName: "X-API-Key",
						Secret:     true,
					},
				},
			}},
			wantClient: false,
			wantAPIKey: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setupDeclaresClientCredentials(tt.setup); got != tt.wantClient {
				t.Fatalf("setupDeclaresClientCredentials() = %v, want %v", got, tt.wantClient)
			}
			if got := setupDeclaresAPIKeyHeader(tt.setup); got != tt.wantAPIKey {
				t.Fatalf("setupDeclaresAPIKeyHeader() = %v, want %v", got, tt.wantAPIKey)
			}
		})
	}
}

func TestStoreValidateServerConfigSetupAndAuth(t *testing.T) {
	bundleID := bundleitemutils.BundleID("bundle-a")
	serverID := spec.MCPServerID("server-a")

	baseHTTP := func() spec.MCPServerConfig {
		return newValidStoreHTTPServer(bundleID, serverID, "HTTP Server", true)
	}
	baseStdio := func() spec.MCPServerConfig {
		return newValidStoreStdioServer(bundleID, serverID, "STDIO Server", true)
	}

	tests := []struct {
		name            string
		cfg             func() spec.MCPServerConfig
		wantErrContains string
	}{
		{
			name: "api key without secret header is rejected",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.StreamableHTTP.AuthMode = spec.MCPHTTPAuthAPIKey
				return cfg
			},
			wantErrContains: "apikey authMode requires a secret header",
		},
		{
			name: "api key with secret header ref is valid",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.StreamableHTTP.AuthMode = spec.MCPHTTPAuthAPIKey
				cfg.StreamableHTTP.SecretHeaderRefs = map[string]string{
					"X-API-Key": mustMCPSecretRefString(
						t,
						bundleID,
						serverID,
						spec.MCPSecretKindHTTPHeader,
						"X-API-Key",
					),
				}
				return cfg
			},
		},
		{
			name: "api key may be deferred by setup",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.StreamableHTTP.AuthMode = spec.MCPHTTPAuthAPIKey
				cfg.Setup = &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
					{
						ID:       "api-key",
						Kind:     spec.MCPSetupKindHTTPHeader,
						Required: true,
						HTTPHeader: &spec.MCPSetupHTTPHeaderInput{
							HeaderName: "X-API-Key",
							Secret:     true,
						},
					},
				}}
				return cfg
			},
		},
		{
			name: "client credentials with secret ref is valid",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.StreamableHTTP.AuthMode = spec.MCPHTTPAuthClientCredentials
				cfg.StreamableHTTP.ClientCredentialRef = mustMCPSecretRefString(
					t,
					bundleID,
					serverID,
					spec.MCPSecretKindOAuthClientCredentials,
					"clientCredentials",
				)
				return cfg
			},
		},
		{
			name: "client credentials may be deferred by setup",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.StreamableHTTP.AuthMode = spec.MCPHTTPAuthClientCredentials
				cfg.Setup = &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
					{
						ID:       "client-credentials",
						Kind:     spec.MCPSetupKindOAuthClientCredentials,
						Required: true,
						OAuthClientCredentials: &spec.MCPSetupOAuthClientCredentialsInput{
							ClientSecretRequired: true,
						},
					},
				}}
				return cfg
			},
		},
		{
			name: "oauth authorization header as plain header is rejected",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.StreamableHTTP.AuthMode = spec.MCPHTTPAuthOAuth
				cfg.StreamableHTTP.Headers = map[string]string{
					"Authorization": "Bearer token",
				}
				return cfg
			},
			wantErrContains: "Authorization header is not allowed",
		},
		{
			name: "oauth authorization header as secret header is rejected",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.StreamableHTTP.AuthMode = spec.MCPHTTPAuthOAuth
				cfg.StreamableHTTP.SecretHeaderRefs = map[string]string{
					"Authorization": mustMCPSecretRefString(
						t,
						bundleID,
						serverID,
						spec.MCPSecretKindHTTPHeader,
						"Authorization",
					),
				}
				return cfg
			},
			wantErrContains: "Authorization header is not allowed",
		},
		{
			name: "streamableHttpUrl setup input is valid",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.Setup = &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
					{
						ID:                "endpoint",
						Kind:              spec.MCPSetupKindStreamableHTTPURL,
						Required:          true,
						StreamableHTTPURL: &spec.MCPSetupStreamableHTTPURLInput{},
					},
				}}
				return cfg
			},
		},
		{
			name: "client ID metadata document url setup input is valid for oauth",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.StreamableHTTP.AuthMode = spec.MCPHTTPAuthOAuth
				cfg.StreamableHTTP.ClientIDMetadataDocumentURL = "https://client.example.com/flexigpt-mcp-client.json"
				cfg.Setup = &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
					{
						ID:                          "client-id-metadata",
						Kind:                        spec.MCPSetupKindClientIDMetadataDocURL,
						Required:                    true,
						ClientIDMetadataDocumentURL: &spec.MCPSetupClientIDMetadataDocumentURLInput{},
					},
				}}
				return cfg
			},
		},
		{
			name: "stdio env setup input is valid on stdio transport",
			cfg: func() spec.MCPServerConfig {
				cfg := baseStdio()
				cfg.Setup = &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
					{
						ID:       "token",
						Kind:     spec.MCPSetupKindStdioEnv,
						Required: true,
						StdioEnv: &spec.MCPSetupStdioEnvInput{
							EnvName: "TOKEN",
							Secret:  true,
						},
					},
				}}
				return cfg
			},
		},
		{
			name: "stdio env setup input on http transport is rejected",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.Setup = &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
					{
						ID:       "token",
						Kind:     spec.MCPSetupKindStdioEnv,
						Required: true,
						StdioEnv: &spec.MCPSetupStdioEnvInput{EnvName: "TOKEN"},
					},
				}}
				return cfg
			},
			wantErrContains: "stdioEnv requires stdio",
		},
		{
			name: "duplicate setup input IDs are rejected",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.Setup = &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
					{
						ID:         "dup",
						Kind:       spec.MCPSetupKindHTTPHeader,
						HTTPHeader: &spec.MCPSetupHTTPHeaderInput{HeaderName: "X-A"},
					},
					{
						ID:         "dup",
						Kind:       spec.MCPSetupKindHTTPHeader,
						HTTPHeader: &spec.MCPSetupHTTPHeaderInput{HeaderName: "X-B"},
					},
				}}
				return cfg
			},
			wantErrContains: "duplicated",
		},
		{
			name: "setup inputs must set exactly one kind-specific block",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.Setup = &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
					{
						ID:         "bad",
						Kind:       spec.MCPSetupKindHTTPHeader,
						HTTPHeader: &spec.MCPSetupHTTPHeaderInput{HeaderName: "X-A"},
						StdioEnv:   &spec.MCPSetupStdioEnvInput{EnvName: "TOKEN"},
					},
				}}
				return cfg
			},
			wantErrContains: "exactly one kind-specific block",
		},
		{
			name: "client ID metadata doc input on non-oauth is rejected",
			cfg: func() spec.MCPServerConfig {
				cfg := baseHTTP()
				cfg.Setup = &spec.MCPServerSetup{Inputs: []spec.MCPServerSetupInput{
					{
						ID:                          "doc",
						Kind:                        spec.MCPSetupKindClientIDMetadataDocURL,
						Required:                    true,
						ClientIDMetadataDocumentURL: &spec.MCPSetupClientIDMetadataDocumentURLInput{},
					},
				}}
				return cfg
			},
			wantErrContains: "requires oauth authMode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg()
			err := validateServerConfig(&cfg)
			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("validateServerConfig succeeded, want error containing %q", tt.wantErrContains)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateServerConfig: %v", err)
			}
		})
	}
}

func TestStoreSetupOverlayPaths(t *testing.T) {
	ctx := t.Context()
	st, err := NewMCPStore(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	bid, sid, cfg, ok := findBuiltInServerByTransport(st.builtinData, spec.MCPTransportStreamableHTTP)
	if !ok {
		bid, sid, cfg, ok = findBuiltInServerByTransport(st.builtinData, spec.MCPTransportStdio)
	}
	if !ok {
		t.Fatal("no built-in server available for setup overlay test")
	}

	switch cfg.Transport {
	case spec.MCPTransportStreamableHTTP:
		secretRef1 := mustMCPSecretRefString(t, bid, sid, spec.MCPSecretKindHTTPHeader, "X-Overlay-Secret")
		patch1 := spec.MCPBuiltInServerOverlay{
			StreamableHTTP: &spec.MCPStreamableHTTPConfigOverlay{
				Headers: map[string]string{
					"X-Overlay-A": "a",
				},
				SecretHeaderRefs: map[string]string{
					"X-Overlay-Secret": secretRef1,
				},
			},
		}
		got1, err := st.ApplyBuiltInServerSetupOverlay(ctx, bid, sid, patch1, true)
		if err != nil {
			t.Fatalf("ApplyBuiltInServerSetupOverlay(reset=true): %v", err)
		}
		if got1.StreamableHTTP == nil {
			t.Fatal("updated built-in server missing streamableHTTP config")
		}
		if got1.StreamableHTTP.Headers["X-Overlay-A"] != "a" {
			t.Fatalf("overlay header missing from first apply: %#v", got1.StreamableHTTP.Headers)
		}
		if got1.StreamableHTTP.SecretHeaderRefs["X-Overlay-Secret"] != secretRef1 {
			t.Fatalf("overlay secret ref missing from first apply: %#v", got1.StreamableHTTP.SecretHeaderRefs)
		}

		secretRef2 := mustMCPSecretRefString(t, bid, sid, spec.MCPSecretKindHTTPHeader, "X-Overlay-Secret-2")
		patch2 := spec.MCPBuiltInServerOverlay{
			StreamableHTTP: &spec.MCPStreamableHTTPConfigOverlay{
				Headers: map[string]string{
					"X-Overlay-B": "b",
				},
				SecretHeaderRefs: map[string]string{
					"X-Overlay-Secret-2": secretRef2,
				},
			},
		}
		got2, err := st.ApplyBuiltInServerSetupOverlay(ctx, bid, sid, patch2, false)
		if err != nil {
			t.Fatalf("ApplyBuiltInServerSetupOverlay(reset=false): %v", err)
		}
		if got2.StreamableHTTP.Headers["X-Overlay-A"] != "a" || got2.StreamableHTTP.Headers["X-Overlay-B"] != "b" {
			t.Fatalf("overlay headers were not merged: %#v", got2.StreamableHTTP.Headers)
		}
		if got2.StreamableHTTP.SecretHeaderRefs["X-Overlay-Secret"] != secretRef1 ||
			got2.StreamableHTTP.SecretHeaderRefs["X-Overlay-Secret-2"] != secretRef2 {
			t.Fatalf("overlay secret refs were not merged: %#v", got2.StreamableHTTP.SecretHeaderRefs)
		}

		patch3 := spec.MCPBuiltInServerOverlay{
			StreamableHTTP: &spec.MCPStreamableHTTPConfigOverlay{
				Headers: map[string]string{
					"X-Overlay-C": "c",
				},
			},
		}
		got3, err := st.ApplyBuiltInServerSetupOverlay(ctx, bid, sid, patch3, true)
		if err != nil {
			t.Fatalf("ApplyBuiltInServerSetupOverlay(reset=true second): %v", err)
		}
		if got3.StreamableHTTP.Headers["X-Overlay-A"] != "" || got3.StreamableHTTP.Headers["X-Overlay-B"] != "" {
			t.Fatalf("reset overlay should have cleared previous headers: %#v", got3.StreamableHTTP.Headers)
		}
		if got3.StreamableHTTP.Headers["X-Overlay-C"] != "c" {
			t.Fatalf("reset overlay missing new header: %#v", got3.StreamableHTTP.Headers)
		}

		if stdioBid, stdioSid, _, ok := findBuiltInServerByTransport(st.builtinData, spec.MCPTransportStdio); ok {
			_, err := st.ApplyBuiltInServerSetupOverlay(ctx, stdioBid, stdioSid, spec.MCPBuiltInServerOverlay{
				StreamableHTTP: &spec.MCPStreamableHTTPConfigOverlay{Headers: map[string]string{"X-Bad": "1"}},
			}, false)
			if err == nil || !strings.Contains(err.Error(), "streamableHttp overlay on stdio server") {
				t.Fatalf("ApplyBuiltInServerSetupOverlay(mismatched transport) = %v", err)
			}
		}

	case spec.MCPTransportStdio:
		secretRef1 := mustMCPSecretRefString(t, bid, sid, spec.MCPSecretKindStdioEnv, "TOKEN")
		patch1 := spec.MCPBuiltInServerOverlay{
			Stdio: &spec.MCPStdioConfigOverlay{
				Env: map[string]string{
					"PLAIN": "1",
				},
				SecretEnvRefs: map[string]string{
					"TOKEN": secretRef1,
				},
			},
		}
		got1, err := st.ApplyBuiltInServerSetupOverlay(ctx, bid, sid, patch1, true)
		if err != nil {
			t.Fatalf("ApplyBuiltInServerSetupOverlay(reset=true): %v", err)
		}
		if got1.Stdio == nil {
			t.Fatal("updated built-in server missing stdio config")
		}
		if got1.Stdio.Env["PLAIN"] != "1" {
			t.Fatalf("overlay env missing from first apply: %#v", got1.Stdio.Env)
		}
		if got1.Stdio.SecretEnvRefs["TOKEN"] != secretRef1 {
			t.Fatalf("overlay secret env ref missing from first apply: %#v", got1.Stdio.SecretEnvRefs)
		}

		secretRef2 := mustMCPSecretRefString(t, bid, sid, spec.MCPSecretKindStdioEnv, "TOKEN_2")
		patch2 := spec.MCPBuiltInServerOverlay{
			Stdio: &spec.MCPStdioConfigOverlay{
				Env: map[string]string{
					"PLAIN_2": "2",
				},
				SecretEnvRefs: map[string]string{
					"TOKEN_2": secretRef2,
				},
			},
		}
		got2, err := st.ApplyBuiltInServerSetupOverlay(ctx, bid, sid, patch2, false)
		if err != nil {
			t.Fatalf("ApplyBuiltInServerSetupOverlay(reset=false): %v", err)
		}
		if got2.Stdio.Env["PLAIN"] != "1" || got2.Stdio.Env["PLAIN_2"] != "2" {
			t.Fatalf("overlay envs were not merged: %#v", got2.Stdio.Env)
		}
		if got2.Stdio.SecretEnvRefs["TOKEN"] != secretRef1 || got2.Stdio.SecretEnvRefs["TOKEN_2"] != secretRef2 {
			t.Fatalf("overlay secret env refs were not merged: %#v", got2.Stdio.SecretEnvRefs)
		}

		patch3 := spec.MCPBuiltInServerOverlay{
			Stdio: &spec.MCPStdioConfigOverlay{
				Env: map[string]string{
					"ONLY": "3",
				},
			},
		}
		got3, err := st.ApplyBuiltInServerSetupOverlay(ctx, bid, sid, patch3, true)
		if err != nil {
			t.Fatalf("ApplyBuiltInServerSetupOverlay(reset=true second): %v", err)
		}
		if got3.Stdio.Env["PLAIN"] != "" || got3.Stdio.Env["PLAIN_2"] != "" {
			t.Fatalf("reset overlay should have cleared previous env values: %#v", got3.Stdio.Env)
		}
		if got3.Stdio.Env["ONLY"] != "3" {
			t.Fatalf("reset overlay missing new env value: %#v", got3.Stdio.Env)
		}

		if httpBid, httpSid, _, ok := findBuiltInServerByTransport(
			st.builtinData,
			spec.MCPTransportStreamableHTTP,
		); ok {
			_, err := st.ApplyBuiltInServerSetupOverlay(ctx, httpBid, httpSid, spec.MCPBuiltInServerOverlay{
				Stdio: &spec.MCPStdioConfigOverlay{Env: map[string]string{"BAD": "1"}},
			}, false)
			if err == nil || !strings.Contains(err.Error(), "stdio overlay on streamableHttp server") {
				t.Fatalf("ApplyBuiltInServerSetupOverlay(mismatched transport) = %v", err)
			}
		}
	}

	userBundle := bundleitemutils.BundleID("user-bundle")
	putTestBundle(t, st, userBundle, "User Bundle")
	userServer := spec.MCPServerID("user-server")
	mustCreateHTTPServer(t, st, userBundle, userServer, "User HTTP", true)

	userPatch := spec.MCPBuiltInServerOverlay{
		StreamableHTTP: &spec.MCPStreamableHTTPConfigOverlay{
			Headers: map[string]string{"X-User-Overlay": "true"},
		},
	}
	updated, err := st.ApplyUserServerSetupOverlay(ctx, userBundle, userServer, userPatch)
	if err != nil {
		t.Fatalf("ApplyUserServerSetupOverlay: %v", err)
	}
	if updated.StreamableHTTP == nil || updated.StreamableHTTP.Headers["X-User-Overlay"] != "true" {
		t.Fatalf("ApplyUserServerSetupOverlay did not persist overlay: %#v", updated.StreamableHTTP)
	}

	reloaded, err := st.GetMCPServer(ctx, &spec.GetMCPServerRequest{BundleID: userBundle, ServerID: userServer})
	if err != nil {
		t.Fatalf("GetMCPServer(after user overlay): %v", err)
	}
	if reloaded.Body.StreamableHTTP == nil || reloaded.Body.StreamableHTTP.Headers["X-User-Overlay"] != "true" {
		t.Fatalf("GetMCPServer(after user overlay) missing overlay: %#v", reloaded.Body.StreamableHTTP)
	}
}

func TestStoreCloneServerSetupDeepCopy(t *testing.T) {
	now := time.Now().UTC()
	cfg := spec.MCPServerConfig{
		SchemaVersion: spec.MCPSchemaVersion,
		BundleID:      bundleitemutils.BundleID("bundle-a"),
		ID:            spec.MCPServerID("server-a"),
		DisplayName:   "Server A",
		Enabled:       true,
		Transport:     spec.MCPTransportStreamableHTTP,
		StreamableHTTP: &spec.MCPStreamableHTTPConfig{
			URL:      "http://127.0.0.1:1234/mcp",
			AuthMode: spec.MCPHTTPAuthNone,
		},
		DefaultPolicy: spec.DefaultMCPServerPolicy(),
		CreatedAt:     now.Add(-time.Minute),
		ModifiedAt:    now,
		Setup: &spec.MCPServerSetup{
			Note: "setup note",
			Inputs: []spec.MCPServerSetupInput{
				{
					ID:       "client",
					Kind:     spec.MCPSetupKindOAuthClientCredentials,
					Required: true,
					OAuthClientCredentials: &spec.MCPSetupOAuthClientCredentialsInput{
						ClientSecretRequired: true,
					},
				},
				{
					ID:       "header",
					Kind:     spec.MCPSetupKindHTTPHeader,
					Required: true,
					HTTPHeader: &spec.MCPSetupHTTPHeaderInput{
						HeaderName:  "X-Test",
						Secret:      true,
						ValuePrefix: "pre",
						ValueSuffix: "suf",
					},
				},
				{
					ID:       "env",
					Kind:     spec.MCPSetupKindStdioEnv,
					Required: true,
					StdioEnv: &spec.MCPSetupStdioEnvInput{
						EnvName:     "TOKEN",
						Secret:      true,
						ValuePrefix: "pre",
						ValueSuffix: "suf",
					},
				},
				{
					ID:                "url",
					Kind:              spec.MCPSetupKindStreamableHTTPURL,
					Required:          true,
					StreamableHTTPURL: &spec.MCPSetupStreamableHTTPURLInput{},
				},
				{
					ID:                          "doc",
					Kind:                        spec.MCPSetupKindClientIDMetadataDocURL,
					Required:                    true,
					ClientIDMetadataDocumentURL: &spec.MCPSetupClientIDMetadataDocumentURLInput{},
				},
			},
		},
	}

	cloned := cloneServerConfig(cfg)

	cfg.Setup.Note = "mutated"
	cfg.Setup.Inputs[0].Label = "changed"
	cfg.Setup.Inputs[0].OAuthClientCredentials.ClientSecretRequired = false
	cfg.Setup.Inputs[1].HTTPHeader.HeaderName = "Changed"
	cfg.Setup.Inputs[2].StdioEnv.EnvName = "CHANGED"
	cfg.Setup.Inputs[3].StreamableHTTPURL = nil
	cfg.Setup.Inputs[4].ClientIDMetadataDocumentURL = nil

	if cloned.Setup == nil {
		t.Fatal("cloneServerConfig returned nil setup")
	}
	if cloned.Setup.Note != "setup note" {
		t.Fatalf("cloned setup note = %q, want %q", cloned.Setup.Note, "setup note")
	}
	if cloned.Setup.Inputs[0].Label != "" {
		t.Fatalf("cloned setup input label = %q, want empty", cloned.Setup.Inputs[0].Label)
	}
	if cloned.Setup.Inputs[0].OAuthClientCredentials == nil ||
		!cloned.Setup.Inputs[0].OAuthClientCredentials.ClientSecretRequired {
		t.Fatalf("cloned oauth client credentials mutated: %#v", cloned.Setup.Inputs[0].OAuthClientCredentials)
	}
	if cloned.Setup.Inputs[1].HTTPHeader == nil || cloned.Setup.Inputs[1].HTTPHeader.HeaderName != "X-Test" {
		t.Fatalf("cloned HTTP header mutated: %#v", cloned.Setup.Inputs[1].HTTPHeader)
	}
	if cloned.Setup.Inputs[2].StdioEnv == nil || cloned.Setup.Inputs[2].StdioEnv.EnvName != "TOKEN" {
		t.Fatalf("cloned stdio env mutated: %#v", cloned.Setup.Inputs[2].StdioEnv)
	}
	if cloned.Setup.Inputs[3].StreamableHTTPURL == nil {
		t.Fatalf("cloned streamableHTTPURL mutated: %#v", cloned.Setup.Inputs[3].StreamableHTTPURL)
	}
	if cloned.Setup.Inputs[4].ClientIDMetadataDocumentURL == nil {
		t.Fatalf("cloned clientIDMetadataDocumentURL mutated: %#v", cloned.Setup.Inputs[4].ClientIDMetadataDocumentURL)
	}
}

func findBuiltInServerByTransport(
	data *BuiltInData,
	transport spec.MCPTransportType,
) (bundleitemutils.BundleID, spec.MCPServerID, spec.MCPServerConfig, bool) {
	if data == nil {
		return "", "", spec.MCPServerConfig{}, false
	}

	bundles, servers, err := data.ListBuiltInData(context.Background())
	if err != nil {
		return "", "", spec.MCPServerConfig{}, false
	}

	bundleIDs := make([]string, 0, len(bundles))
	for id := range bundles {
		bundleIDs = append(bundleIDs, string(id))
	}
	for _, bundleStr := range bundleIDs {
		bid := bundleitemutils.BundleID(bundleStr)
		serverMap := servers[bid]
		if len(serverMap) == 0 {
			continue
		}
		serverIDs := make([]string, 0, len(serverMap))
		for sid := range serverMap {
			serverIDs = append(serverIDs, string(sid))
		}
		for _, serverStr := range serverIDs {
			sid := spec.MCPServerID(serverStr)
			cfg, ok := serverMap[sid]
			if !ok || cfg.Transport != transport {
				continue
			}
			return bid, sid, cfg, true
		}
	}
	return "", "", spec.MCPServerConfig{}, false
}

func mustMCPSecretRefString(
	t *testing.T,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	kind spec.MCPSecretKind,
	slot string,
) string {
	t.Helper()
	ref, err := secret.NewMCPSecretRefString(bundleID, serverID, kind, slot)
	if err != nil {
		t.Fatalf("NewMCPSecretRefString(%s/%s/%s/%s): %v", bundleID, serverID, kind, slot, err)
	}
	return ref
}

func newValidStoreStdioServer(
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	displayName string,
	enabled bool,
) spec.MCPServerConfig {
	now := time.Now().UTC()
	return spec.MCPServerConfig{
		SchemaVersion: spec.MCPSchemaVersion,
		BundleID:      bundleID,
		ID:            serverID,
		DisplayName:   displayName,
		Enabled:       enabled,
		Transport:     spec.MCPTransportStdio,
		Stdio: &spec.MCPStdioConfig{
			Command: "server-binary",
		},
		DefaultPolicy: spec.DefaultMCPServerPolicy(),
		CreatedAt:     now.Add(-time.Minute),
		ModifiedAt:    now,
	}
}

var _ = errors.New
