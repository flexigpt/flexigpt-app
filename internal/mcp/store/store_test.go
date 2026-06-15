package store

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	storeTestBundleA = "bundle-a"
	storeTestBundleB = "bundle-b"
	storeTestBundleC = "bundle-c"
	storeTestBundleD = "bundle-d"
	storeTestServerA = "server-a"
	storeTestServerB = "server-b"
)

func TestStorePutGetListPatchDeleteAndPersistence(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()

	st, err := NewMCPStore(t.Context(), dir)
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	bundleID := bundleitemutils.BundleID("bundle-a")
	putTestBundle(t, st, bundleID, "Bundle A")

	alphaPolicy := spec.MCPServerPolicy{
		DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
		DefaultExecutionMode: spec.MCPExecutionModeManual,
	}
	alphaID := spec.MCPServerID("alpha")
	alphaPayload := &spec.PutMCPServerPayload{
		DisplayName: "Alpha HTTP",
		Enabled:     true,
		Transport:   spec.MCPTransportStreamableHTTP,
		StreamableHTTP: &spec.MCPStreamableHTTPConfig{
			URL:      "http://127.0.0.1:1234/mcp",
			AuthMode: spec.MCPHTTPAuthNone,
		},
		DefaultPolicy: &alphaPolicy,
	}

	betaID := spec.MCPServerID("beta")
	betaSecretRef, err := secret.NewMCPSecretRefString(bundleID, betaID, spec.MCPSecretKindStdioEnv, "TOKEN")
	if err != nil {
		t.Fatalf("NewMCPSecretRefString: %v", err)
	}
	betaPayload := &spec.PutMCPServerPayload{
		DisplayName: "Beta STDIO",
		Enabled:     true,
		Transport:   spec.MCPTransportStdio,
		Stdio: &spec.MCPStdioConfig{
			Command: "test-server",
			Env: map[string]string{
				"PLAIN": "1",
			},
			SecretEnvRefs: map[string]string{
				"TOKEN": betaSecretRef,
			},
		},
	}

	if _, err := st.PutMCPServer(ctx, &spec.PutMCPServerRequest{
		BundleID: bundleID,
		ServerID: alphaID,
		Body:     alphaPayload,
	}); err != nil {
		t.Fatalf("PutMCPServer(alpha): %v", err)
	}
	if _, err := st.PutMCPServer(ctx, &spec.PutMCPServerRequest{
		BundleID: bundleID,
		ServerID: betaID,
		Body:     betaPayload,
	}); err != nil {
		t.Fatalf("PutMCPServer(beta): %v", err)
	}

	gotAlpha, err := st.GetMCPServer(ctx, &spec.GetMCPServerRequest{BundleID: bundleID, ServerID: alphaID})
	if err != nil {
		t.Fatalf("GetMCPServer(alpha): %v", err)
	}
	if gotAlpha.Body.DisplayName != "Alpha HTTP" {
		t.Fatalf("Alpha DisplayName = %q, want %q", gotAlpha.Body.DisplayName, "Alpha HTTP")
	}

	if gotAlpha.Body.TrustLevel != spec.MCPTrustLevelUntrusted {
		t.Fatalf("Alpha TrustLevel = %q, want untrusted", gotAlpha.Body.TrustLevel)
	}
	if gotAlpha.Body.DefaultPolicy != alphaPolicy {
		t.Fatalf("Alpha DefaultPolicy = %#v, want %#v", gotAlpha.Body.DefaultPolicy, alphaPolicy)
	}

	gotAlpha.Body.DisplayName = "mutated"
	if gotAlpha.Body.StreamableHTTP != nil {
		gotAlpha.Body.StreamableHTTP.URL = "mutated"
	}
	gotAlpha2, err := st.GetMCPServer(ctx, &spec.GetMCPServerRequest{BundleID: bundleID, ServerID: alphaID})
	if err != nil {
		t.Fatalf("GetMCPServer(alpha #2): %v", err)
	}
	if gotAlpha2.Body.DisplayName != "Alpha HTTP" {
		t.Fatalf("Alpha clone not preserved: DisplayName = %q", gotAlpha2.Body.DisplayName)
	}
	if gotAlpha2.Body.StreamableHTTP == nil || gotAlpha2.Body.StreamableHTTP.URL != "http://127.0.0.1:1234/mcp" {
		t.Fatalf("Alpha clone not preserved: StreamableHTTP = %#v", gotAlpha2.Body.StreamableHTTP)
	}

	if _, err := st.PatchMCPServerPolicy(ctx, &spec.PatchMCPServerPolicyRequest{
		BundleID: bundleID,
		ServerID: alphaID,
		Body: &spec.PatchMCPServerPolicyPayload{
			DefaultPolicy: &spec.MCPServerPolicy{
				DefaultApprovalRule:  spec.MCPApprovalRuleDeny,
				DefaultExecutionMode: spec.MCPExecutionModeAuto,
			},
		},
	}); err != nil {
		t.Fatalf("PatchMCPServerPolicy(alpha): %v", err)
	}

	if _, err := st.PatchMCPServerEnabled(ctx, &spec.PatchMCPServerEnabledRequest{
		BundleID: bundleID,
		ServerID: betaID,
		Body:     &spec.PatchMCPServerEnabledRequestBody{Enabled: false},
	}); err != nil {
		t.Fatalf("PatchMCPServerEnabled(beta): %v", err)
	}

	enabled := true
	enabledOnly, err := st.ListMCPServers(ctx, &spec.ListMCPServersRequest{
		BundleID: bundleID,
		Enabled:  &enabled,
	})
	if err != nil {
		t.Fatalf("ListMCPServers(enabled=true): %v", err)
	}
	if len(enabledOnly.Body.Servers) != 1 || enabledOnly.Body.Servers[0].ID != alphaID {
		t.Fatalf("ListMCPServers(enabled=true) = %#v, want only alpha", enabledOnly.Body.Servers)
	}

	page1, err := st.ListMCPServers(ctx, &spec.ListMCPServersRequest{
		BundleID:        bundleID,
		IncludeDisabled: true,
		PageSize:        1,
	})
	if err != nil {
		t.Fatalf("ListMCPServers(page1): %v", err)
	}
	if len(page1.Body.Servers) != 1 {
		t.Fatalf("page1 len = %d, want 1", len(page1.Body.Servers))
	}
	if page1.Body.NextPageToken == nil {
		t.Fatalf("page1 NextPageToken is nil")
	}

	page2, err := st.ListMCPServers(ctx, &spec.ListMCPServersRequest{
		BundleID:        bundleID,
		IncludeDisabled: true,
		PageToken:       *page1.Body.NextPageToken,
	})
	if err != nil {
		t.Fatalf("ListMCPServers(page2): %v", err)
	}
	if len(page2.Body.Servers) != 1 {
		t.Fatalf("page2 len = %d, want 1", len(page2.Body.Servers))
	}
	seen := map[spec.MCPServerID]bool{
		page1.Body.Servers[0].ID: true,
		page2.Body.Servers[0].ID: true,
	}
	if !seen[alphaID] || !seen[betaID] {
		t.Fatalf("pagination returned wrong IDs: %#v", seen)
	}

	if err := st.Close(); err != nil {
		t.Fatalf("Close(store): %v", err)
	}

	st2, err := NewMCPStore(t.Context(), dir)
	if err != nil {
		t.Fatalf("NewMCPStore(reopen): %v", err)
	}
	t.Cleanup(func() { _ = st2.Close() })

	reAlpha, err := st2.GetMCPServer(ctx, &spec.GetMCPServerRequest{BundleID: bundleID, ServerID: alphaID})
	if err != nil {
		t.Fatalf("GetMCPServer(alpha reopen): %v", err)
	}
	if reAlpha.Body.DefaultPolicy.DefaultApprovalRule != spec.MCPApprovalRuleDeny ||
		reAlpha.Body.DefaultPolicy.DefaultExecutionMode != spec.MCPExecutionModeAuto {
		t.Fatalf("patched policy not persisted: %#v", reAlpha.Body.DefaultPolicy)
	}

	reBeta, err := st2.GetMCPServer(ctx, &spec.GetMCPServerRequest{BundleID: bundleID, ServerID: betaID})
	if err != nil {
		t.Fatalf("GetMCPServer(beta reopen): %v", err)
	}
	if reBeta.Body.Enabled {
		t.Fatalf("beta Enabled = true, want false")
	}

	if _, err := st2.DeleteMCPServer(
		ctx,
		&spec.DeleteMCPServerRequest{BundleID: bundleID, ServerID: alphaID},
	); err != nil {
		t.Fatalf("DeleteMCPServer(alpha): %v", err)
	}
	if _, err := st2.GetMCPServer(
		ctx,
		&spec.GetMCPServerRequest{BundleID: bundleID, ServerID: alphaID},
	); !errors.Is(
		err,
		spec.ErrMCPServerNotFound,
	) {
		t.Fatalf("GetMCPServer(alpha deleted) error = %v, want ErrMCPServerNotFound", err)
	}

	listAfterDelete, err := st2.ListMCPServers(
		ctx,
		&spec.ListMCPServersRequest{BundleID: bundleID, IncludeDisabled: true},
	)
	if err != nil {
		t.Fatalf("ListMCPServers(after delete): %v", err)
	}
	if len(listAfterDelete.Body.Servers) != 1 || listAfterDelete.Body.Servers[0].ID != betaID {
		t.Fatalf("ListMCPServers(after delete) = %#v, want only beta", listAfterDelete.Body.Servers)
	}
}

func TestStoreRejectsInvalidConfigs(t *testing.T) {
	tests := []struct {
		name string
		make func(bundleID bundleitemutils.BundleID, id spec.MCPServerID) *spec.PutMCPServerPayload
		want string
	}{
		{
			name: "stdio missing config",
			make: func(bundleID bundleitemutils.BundleID, id spec.MCPServerID) *spec.PutMCPServerPayload {
				return &spec.PutMCPServerPayload{
					DisplayName: "Server",
					Enabled:     true,
					Transport:   spec.MCPTransportStdio,
				}
			},
			want: "stdio config required",
		},
		{
			name: "stdio shell command",
			make: func(bundleID bundleitemutils.BundleID, id spec.MCPServerID) *spec.PutMCPServerPayload {
				return &spec.PutMCPServerPayload{
					DisplayName: "Server",
					Enabled:     true,
					Transport:   spec.MCPTransportStdio,
					Stdio: &spec.MCPStdioConfig{
						Command: commandBash,
					},
				}
			},
			want: "must execute the server directly",
		},
		{
			name: "stdio env overlap",
			make: func(bundleID bundleitemutils.BundleID, id spec.MCPServerID) *spec.PutMCPServerPayload {
				ref, err := secret.NewMCPSecretRefString(bundleID, id, spec.MCPSecretKindStdioEnv, "TOKEN")
				if err != nil {
					t.Fatalf("NewMCPSecretRefString: %v", err)
				}
				return &spec.PutMCPServerPayload{
					DisplayName: "Server",
					Enabled:     true,
					Transport:   spec.MCPTransportStdio,
					Stdio: &spec.MCPStdioConfig{
						Command: "test-server",
						Env: map[string]string{
							"token": "visible",
						},
						SecretEnvRefs: map[string]string{
							"TOKEN": ref,
						},
					},
				}
			},
			want: "both define",
		},
		{
			name: "http userinfo",
			make: func(bundleID bundleitemutils.BundleID, id spec.MCPServerID) *spec.PutMCPServerPayload {
				return &spec.PutMCPServerPayload{
					DisplayName: "Server",
					Enabled:     true,
					Transport:   spec.MCPTransportStreamableHTTP,
					StreamableHTTP: &spec.MCPStreamableHTTPConfig{
						URL:      "https://user@example.test/mcp",
						AuthMode: spec.MCPHTTPAuthNone,
					},
				}
			},
			want: "must not contain user info",
		},
		{
			name: "client credentials missing ref",
			make: func(bundleID bundleitemutils.BundleID, id spec.MCPServerID) *spec.PutMCPServerPayload {
				return &spec.PutMCPServerPayload{
					DisplayName: "Server",
					Enabled:     true,
					Transport:   spec.MCPTransportStreamableHTTP,
					StreamableHTTP: &spec.MCPStreamableHTTPConfig{
						URL:      "http://127.0.0.1:1234/mcp",
						AuthMode: spec.MCPHTTPAuthClientCredentials,
					},
				}
			},
			want: "clientCredentialRef is required",
		},
		{
			name: "stdio wrong secret ref kind",
			make: func(bundleID bundleitemutils.BundleID, id spec.MCPServerID) *spec.PutMCPServerPayload {
				ref, err := secret.NewMCPSecretRefString(
					bundleID,
					id,
					spec.MCPSecretKindOAuthClientCredentials,
					"clientCredentials",
				)
				if err != nil {
					t.Fatalf("NewMCPSecretRefString: %v", err)
				}
				return &spec.PutMCPServerPayload{
					DisplayName: "Server",
					Enabled:     true,
					Transport:   spec.MCPTransportStdio,
					Stdio: &spec.MCPStdioConfig{
						Command: "test-server",
						SecretEnvRefs: map[string]string{
							"TOKEN": ref,
						},
					},
				}
			},
			want: "kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, err := NewMCPStore(t.Context(), t.TempDir())
			if err != nil {
				t.Fatalf("NewMCPStore: %v", err)
			}
			t.Cleanup(func() { _ = st.Close() })

			bundleID := bundleitemutils.BundleID("bundle-a")
			putTestBundle(t, st, bundleID, "Bundle A")

			id := spec.MCPServerID("server")
			_, err = st.PutMCPServer(t.Context(), &spec.PutMCPServerRequest{
				BundleID: bundleID,
				ServerID: id,
				Body:     tt.make(bundleID, id),
			})
			if err == nil {
				t.Fatalf("PutMCPServer succeeded, want error containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestStoreRequiresBundleIDForServerOps(t *testing.T) {
	st, err := NewMCPStore(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	_, err = st.PutMCPServer(t.Context(), &spec.PutMCPServerRequest{
		ServerID: "server",
		Body: &spec.PutMCPServerPayload{
			DisplayName: "Server",
			Enabled:     true,
			Transport:   spec.MCPTransportStreamableHTTP,
			StreamableHTTP: &spec.MCPStreamableHTTPConfig{
				URL:      "https://example.test/mcp",
				AuthMode: spec.MCPHTTPAuthNone,
			},
		},
	})
	if !errors.Is(err, spec.ErrMCPInvalidRequest) {
		t.Fatalf("PutMCPServer missing bundleID error = %v, want ErrMCPInvalidRequest", err)
	}
}

func TestStoreRejectsDuplicateServerIDsAcrossBundles(t *testing.T) {
	st, err := NewMCPStore(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	bundleA := bundleitemutils.BundleID("bundle-a")
	bundleB := bundleitemutils.BundleID("bundle-b")
	putTestBundle(t, st, bundleA, "Bundle A")
	putTestBundle(t, st, bundleB, "Bundle B")

	serverID := spec.MCPServerID("shared-server")
	payload := &spec.PutMCPServerPayload{
		DisplayName: "HTTP Server",
		Enabled:     true,
		Transport:   spec.MCPTransportStreamableHTTP,
		StreamableHTTP: &spec.MCPStreamableHTTPConfig{
			URL:      "http://127.0.0.1:1234/mcp",
			AuthMode: spec.MCPHTTPAuthNone,
		},
	}

	if _, err := st.PutMCPServer(t.Context(), &spec.PutMCPServerRequest{
		BundleID: bundleA,
		ServerID: serverID,
		Body:     payload,
	}); err != nil {
		t.Fatalf("PutMCPServer(bundleA): %v", err)
	}

	_, err = st.PutMCPServer(t.Context(), &spec.PutMCPServerRequest{
		BundleID: bundleB,
		ServerID: serverID,
		Body:     payload,
	})
	if !errors.Is(err, spec.ErrMCPConflict) {
		t.Fatalf("PutMCPServer(bundleB duplicate) error = %v, want ErrMCPConflict", err)
	}
}

func TestStoreRejectsPutOnDeletedBundle(t *testing.T) {
	st, err := NewMCPStore(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	bundleID := bundleitemutils.BundleID("bundle-a")
	putTestBundle(t, st, bundleID, "Bundle A")

	if _, err := st.DeleteMCPBundle(t.Context(), &spec.DeleteMCPBundleRequest{BundleID: bundleID}); err != nil {
		t.Fatalf("DeleteMCPBundle: %v", err)
	}

	_, err = st.PutMCPBundle(t.Context(), &spec.PutMCPBundleRequest{
		BundleID: bundleID,
		Body: &spec.PutMCPBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug(bundleID),
			DisplayName: "Bundle A",
			IsEnabled:   true,
		},
	})
	if !errors.Is(err, spec.ErrMCPBundleDeleting) {
		t.Fatalf("PutMCPBundle(after delete) error = %v, want ErrMCPBundleDeleting", err)
	}
}

func TestStoreRejectsPutOnDeletedServer(t *testing.T) {
	st, err := NewMCPStore(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	bundleID := bundleitemutils.BundleID("bundle-a")
	putTestBundle(t, st, bundleID, "Bundle A")

	serverID := spec.MCPServerID("server")
	payload := &spec.PutMCPServerPayload{
		DisplayName: "HTTP Server",
		Enabled:     true,
		Transport:   spec.MCPTransportStreamableHTTP,
		StreamableHTTP: &spec.MCPStreamableHTTPConfig{
			URL:      "http://127.0.0.1:1234/mcp",
			AuthMode: spec.MCPHTTPAuthNone,
		},
	}

	if _, err := st.PutMCPServer(t.Context(), &spec.PutMCPServerRequest{
		BundleID: bundleID,
		ServerID: serverID,
		Body:     payload,
	}); err != nil {
		t.Fatalf("PutMCPServer: %v", err)
	}

	if _, err := st.DeleteMCPServer(t.Context(), &spec.DeleteMCPServerRequest{
		BundleID: bundleID,
		ServerID: serverID,
	}); err != nil {
		t.Fatalf("DeleteMCPServer: %v", err)
	}
}

func TestStoreRejectsBadPageToken(t *testing.T) {
	st, err := NewMCPStore(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if _, err := st.ListMCPServers(t.Context(), &spec.ListMCPServersRequest{
		BundleID:  spec.BaseMCPBundleID,
		PageToken: "not-a-token",
	}); !errors.Is(err, spec.ErrMCPInvalidRequest) {
		t.Fatalf("ListMCPServers(bad token) error = %v, want ErrMCPInvalidRequest", err)
	}
}

func TestStoreValidationHelpers(t *testing.T) {
	t.Run("bundle server id requirements", func(t *testing.T) {
		if err := requireMCPBundleID(""); err == nil ||
			!strings.Contains(err.Error(), "bundleID required") {
			t.Fatalf("requireMCPBundleID(empty) = %v", err)
		}
		if err := requireMCPBundleID(storeTestBundleA); err != nil {
			t.Fatalf("requireMCPBundleID(valid): %v", err)
		}

		tests := []struct {
			name            string
			bundleID        bundleitemutils.BundleID
			serverID        spec.MCPServerID
			wantErrContains string
		}{
			{name: "both missing", bundleID: "", serverID: "", wantErrContains: "bundleID and serverID required"},
			{name: "bundle missing", bundleID: "", serverID: storeTestServerA, wantErrContains: "bundleID required"},
			{name: "server missing", bundleID: storeTestBundleA, serverID: "", wantErrContains: "serverID required"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := requireMCPBundleServerIDs(tt.bundleID, tt.serverID)
				if err == nil || !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf(
						"requireMCPBundleServerIDs(%q,%q) = %v, want substring %q",
						tt.bundleID,
						tt.serverID,
						err,
						tt.wantErrContains,
					)
				}
			})
		}
		if err := requireMCPBundleServerIDs(storeTestBundleA, storeTestServerA); err != nil {
			t.Fatalf("requireMCPBundleServerIDs(valid): %v", err)
		}
	})

	t.Run("bundle validation", func(t *testing.T) {
		if err := validateBundle(nil); err == nil || !strings.Contains(err.Error(), "bundle is nil") {
			t.Fatalf("validateBundle(nil) = %v", err)
		}

		bundle := newValidStoreBundle(storeTestBundleA, bundleitemutils.BundleSlug(storeTestBundleA), "Bundle A", true)
		if err := validateBundle(&bundle); err != nil {
			t.Fatalf("validateBundle(valid): %v", err)
		}
	})

	t.Run("server validation", func(t *testing.T) {
		if err := validateServerConfig(nil); err == nil || !strings.Contains(err.Error(), "server config is nil") {
			t.Fatalf("validateServerConfig(nil) = %v", err)
		}

		cfg := newValidStoreHTTPServer(storeTestBundleA, storeTestServerA, "Server A", true)
		cfg.TrustLevel = ""
		if err := validateServerConfig(&cfg); err != nil {
			t.Fatalf("validateServerConfig(valid): %v", err)
		}

		if cfg.TrustLevel != spec.MCPTrustLevelUntrusted {
			t.Fatalf("TrustLevel = %q, want untrusted", cfg.TrustLevel)
		}
	})

	t.Run("base bundle helpers and loopback helper", func(t *testing.T) {
		if !isBaseMCPBundleID(spec.BaseMCPBundleID) {
			t.Fatalf("BaseMCPBundleID not recognized")
		}
		if isBaseMCPBundleID(storeTestBundleA) {
			t.Fatalf("non-base bundle recognized as base")
		}
		if !isBaseMCPBundleSlug(spec.BaseMCPBundleSlug) {
			t.Fatalf("BaseMCPBundleSlug not recognized")
		}
		if isBaseMCPBundleSlug(bundleitemutils.BundleSlug(storeTestBundleA)) {
			t.Fatalf("non-base slug recognized as base")
		}

		if !isLoopbackHost("localhost") || !isLoopbackHost("127.0.0.1") || !isLoopbackHost("::1") {
			t.Fatalf("loopback host detection failed")
		}
		if isLoopbackHost("example.test") {
			t.Fatalf("non-loopback host detected as loopback")
		}
	})
}

func TestStoreCloneHelpers(t *testing.T) {
	t.Run("clone bundle and bundle maps", func(t *testing.T) {
		deleted := time.Now().UTC()
		b := spec.MCPBundle{
			SchemaVersion: spec.MCPSchemaVersion,
			ID:            storeTestBundleA,
			Slug:          bundleitemutils.BundleSlug(storeTestBundleA),
			DisplayName:   "Bundle A",
			Description:   "bundle",
			IsEnabled:     true,
			CreatedAt:     deleted.Add(-time.Hour),
			ModifiedAt:    deleted,
			SoftDeletedAt: &deleted,
		}

		clone := cloneBundle(b)
		b.DisplayName = "mutated"
		*b.SoftDeletedAt = b.SoftDeletedAt.Add(time.Hour)

		if clone.DisplayName != "Bundle A" {
			t.Fatalf("cloneBundle mutated: %#v", clone)
		}

		m := map[bundleitemutils.BundleID]spec.MCPBundle{
			storeTestBundleA: b,
		}
		clonedMap := cloneBundleMap(m)
		m[storeTestBundleA] = newValidStoreBundle(
			storeTestBundleA,
			bundleitemutils.BundleSlug("other"),
			"Changed",
			false,
		)
		if clonedMap[storeTestBundleA].DisplayName != "mutated" {
			t.Fatalf("cloneBundleMap did not preserve value: %#v", clonedMap[storeTestBundleA])
		}
	})

	t.Run("clone server config and server maps", func(t *testing.T) {
		deleted := time.Now().UTC()
		cfg := spec.MCPServerConfig{
			SchemaVersion: spec.MCPSchemaVersion,
			BundleID:      storeTestBundleA,
			ID:            storeTestServerA,
			DisplayName:   "Server A",
			Enabled:       true,
			Transport:     spec.MCPTransportStdio,
			Stdio: &spec.MCPStdioConfig{
				Command: "server-binary",
				Args:    []string{"--flag"},
				Env: map[string]string{
					"A": "1",
				},
				SecretEnvRefs: map[string]string{
					"TOKEN": "ref-a",
				},
			},
			StreamableHTTP: &spec.MCPStreamableHTTPConfig{
				URL:      "https://example.test/mcp",
				AuthMode: spec.MCPHTTPAuthOAuth,
			},
			DefaultPolicy: spec.MCPServerPolicy{
				DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
				DefaultExecutionMode: spec.MCPExecutionModeAuto,
			},
			ToolPolicies: map[string]spec.MCPToolPolicyOverride{
				"tool": {
					ToolName: "tool",
				},
			},
			AppsPolicy: &spec.MCPAppsPolicy{
				Enabled: true,
			},

			CreatedAt:  deleted.Add(-time.Hour),
			ModifiedAt: deleted,
		}

		cloned := cloneServerConfig(cfg)
		cfg.DisplayName = "mutated"
		cfg.Stdio.Args[0] = "--mutated"
		cfg.Stdio.Env["A"] = "2"
		cfg.Stdio.SecretEnvRefs["TOKEN"] = "ref-b"
		cfg.StreamableHTTP.URL = "https://mutated.test"
		cfg.ToolPolicies["tool"] = spec.MCPToolPolicyOverride{ToolName: "changed"}
		cfg.AppsPolicy.Enabled = false

		if cloned.DisplayName != "Server A" || cloned.Stdio.Args[0] != "--flag" || cloned.Stdio.Env["A"] != "1" ||
			cloned.Stdio.SecretEnvRefs["TOKEN"] != "ref-a" || cloned.StreamableHTTP.URL != "https://example.test/mcp" ||
			cloned.ToolPolicies["tool"].ToolName != "tool" || !cloned.AppsPolicy.Enabled {
			t.Fatalf("cloneServerConfig did not deep copy: %#v", cloned)
		}

		serverMap := map[spec.MCPServerID]spec.MCPServerConfig{
			storeTestServerA: cfg,
		}
		clonedServerMap := cloneServerMap(serverMap)
		serverMap[storeTestServerA] = newValidStoreHTTPServer(storeTestBundleA, storeTestServerA, "Changed", false)
		if clonedServerMap[storeTestServerA].DisplayName != "mutated" {
			t.Fatalf("cloneServerMap did not preserve value: %#v", clonedServerMap[storeTestServerA])
		}

		all := map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig{
			storeTestBundleA: {
				storeTestServerA: cfg,
			},
		}
		clonedAll := cloneAllServerMaps(all)
		all[storeTestBundleA][storeTestServerA] = newValidStoreHTTPServer(
			storeTestBundleA,
			storeTestServerA,
			"ChangedAgain",
			false,
		)
		if clonedAll[storeTestBundleA][storeTestServerA].DisplayName != "mutated" {
			t.Fatalf("cloneAllServerMaps did not preserve value: %#v", clonedAll[storeTestBundleA][storeTestServerA])
		}
	})

	t.Run("clone discovery snapshots", func(t *testing.T) {
		ts := time.Now().UTC()
		snap := spec.MCPDiscoverySnapshot{
			BundleID: storeTestBundleA,
			ServerID: storeTestServerA,
			ServerInfo: &spec.MCPImplementationInfo{
				Name:    "name",
				Version: "v1",
			},
			ServerCapabilities: &spec.MCPServerCapabilitiesSummary{
				Experimental: map[string]any{"x": 1},
				Extensions:   map[string]any{"y": 2},
			},
			Tools: []spec.MCPToolCapability{
				{
					BundleID:     storeTestBundleA,
					ServerID:     storeTestServerA,
					ToolName:     "tool",
					DisplayName:  "Tool",
					Digest:       "digest",
					InputSchema:  map[string]any{"a": 1},
					OutputSchema: map[string]any{"b": 2},
					Annotations: &spec.MCPToolAnnotations{
						Title: "annotation",
					},
					App: &spec.MCPToolAppInfo{
						ResourceURI: "ui://demo",
						Visibility:  []string{"model", "app"},
					},
				},
			},
			Resources: []spec.MCPResourceRef{
				{
					BundleID:    storeTestBundleA,
					ServerID:    storeTestServerA,
					URI:         "file:///resource",
					DisplayName: "Resource",
					Annotations: map[string]any{"a": 1},
				},
			},
			ResourceTemplates: []spec.MCPResourceTemplateRef{
				{
					BundleID:    storeTestBundleA,
					ServerID:    storeTestServerA,
					URITemplate: "file:///{id}",
					DisplayName: "Template",
					Arguments: map[string]spec.MCPArgumentDefinition{
						"id": {Name: "id", Required: true},
					},
					Annotations: map[string]any{"a": 1},
				},
			},
			Prompts: []spec.MCPPromptRef{
				{
					BundleID:    storeTestBundleA,
					ServerID:    storeTestServerA,
					PromptName:  "prompt",
					DisplayName: "Prompt",
					Arguments: map[string]spec.MCPArgumentDefinition{
						"name": {Name: "name", Required: true},
					},
				},
			},
			SyncedAt: ts.Format(time.RFC3339Nano),
		}

		clone := cloneDiscoverySnapshot(snap)
		snap.ServerInfo.Name = "mutated"
		snap.ServerCapabilities.Experimental["x"] = 9
		snap.ServerCapabilities.Extensions["y"] = 8
		snap.Tools[0].InputSchema["a"] = 99
		snap.Tools[0].OutputSchema["b"] = 88
		snap.Tools[0].Annotations.Title = "changed"
		snap.Tools[0].App.Visibility[0] = "changed"
		snap.Resources[0].Annotations["a"] = 99
		snap.ResourceTemplates[0].Arguments["id"] = spec.MCPArgumentDefinition{Name: "changed"}
		snap.ResourceTemplates[0].Annotations["a"] = 99
		snap.Prompts[0].Arguments["name"] = spec.MCPArgumentDefinition{Name: "changed"}

		if clone.ServerInfo == nil || clone.ServerInfo.Name != "name" || clone.ServerCapabilities == nil ||
			clone.ServerCapabilities.Experimental["x"] != 1 || clone.ServerCapabilities.Extensions["y"] != 2 ||
			clone.Tools[0].InputSchema["a"] != 1 || clone.Tools[0].OutputSchema["b"] != 2 ||
			clone.Tools[0].Annotations.Title != "annotation" || clone.Tools[0].App.Visibility[0] != "model" ||
			clone.Resources[0].Annotations["a"] != 1 || clone.ResourceTemplates[0].Arguments["id"].Name != "id" ||
			clone.ResourceTemplates[0].Annotations["a"] != 1 || clone.Prompts[0].Arguments["name"].Name != "name" {
			t.Fatalf("cloneDiscoverySnapshot did not deep copy: %#v", clone)
		}
	})
}

func TestStorePaginationHelpers(t *testing.T) {
	now := time.Now().UTC()
	items := []spec.MCPBundle{
		{
			ID:          storeTestBundleB,
			Slug:        bundleitemutils.BundleSlug(storeTestBundleB),
			DisplayName: "Bundle B",
			IsEnabled:   true,
			CreatedAt:   now.Add(-3 * time.Minute),
			ModifiedAt:  now.Add(-3 * time.Minute),
		},
		{
			ID:          storeTestBundleA,
			Slug:        bundleitemutils.BundleSlug(storeTestBundleA),
			DisplayName: "Bundle A",
			IsEnabled:   false,
			CreatedAt:   now.Add(-2 * time.Minute),
			ModifiedAt:  now.Add(-2 * time.Minute),
		},
		{
			ID:          storeTestBundleC,
			Slug:        bundleitemutils.BundleSlug(storeTestBundleC),
			DisplayName: "Bundle C",
			IsEnabled:   true,
			CreatedAt:   now.Add(-time.Minute),
			ModifiedAt:  now.Add(-time.Minute),
		},
	}

	sortBundles(items)
	if items[0].ID != storeTestBundleC || items[1].ID != storeTestBundleA || items[2].ID != storeTestBundleB {
		t.Fatalf("sortBundles order = %#v", items)
	}

	wanted := map[bundleitemutils.BundleID]struct{}{
		storeTestBundleA: {},
		storeTestBundleC: {},
	}
	filtered := filterBundles(items, wanted, false)
	if len(filtered) != 1 || filtered[0].ID != storeTestBundleC {
		t.Fatalf("filterBundles(includeDisabled=false) = %#v", filtered)
	}
	filteredAll := filterBundles(items, wanted, true)
	if len(filteredAll) != 2 {
		t.Fatalf("filterBundles(includeDisabled=true) = %#v", filteredAll)
	}

	if got := bundleCursorStart(items, time.Time{}, ""); got != 0 {
		t.Fatalf("bundleCursorStart(zero) = %d, want 0", got)
	}
	if got := bundleCursorStart(items, items[0].ModifiedAt, items[0].ID); got != 1 {
		t.Fatalf("bundleCursorStart(after first) = %d, want 1", got)
	}

	next := nextBundlePageToken(items, 1, 2, []bundleitemutils.BundleID{storeTestBundleC, storeTestBundleA}, true)
	if next == nil {
		t.Fatalf("nextBundlePageToken returned nil")
	}
	decoded, err := jsonutil.Base64JSONDecode[spec.MCPBundlePageToken](*next)
	if err != nil {
		t.Fatalf("decode nextBundlePageToken: %v", err)
	}
	if decoded.PageSize != 2 || decoded.CursorID != storeTestBundleC || decoded.IncludeDisabled != true {
		t.Fatalf("decoded next token = %#v", decoded)
	}
	if !slices.Equal(decoded.BundleIDs, []bundleitemutils.BundleID{storeTestBundleA, storeTestBundleC}) {
		t.Fatalf("decoded BundleIDs = %#v", decoded.BundleIDs)
	}

	if got := nextBundlePageToken(items, len(items), 2, nil, false); got != nil {
		t.Fatalf("nextBundlePageToken(end-of-list) = %v, want nil", *got)
	}

	pageToken := jsonutil.Base64JSONEncode(spec.MCPBundlePageToken{
		PageSize:        2,
		BundleIDs:       []bundleitemutils.BundleID{storeTestBundleC, storeTestBundleA},
		IncludeDisabled: true,
		CursorMod:       items[0].ModifiedAt.Format(time.RFC3339Nano),
		CursorID:        items[0].ID,
	})

	pageSize, cursorAt, cursorID, includeDisabled, filterIDs, err := parseBundleListPage(&spec.ListMCPBundlesRequest{
		PageToken: pageToken,
	})
	if err != nil {
		t.Fatalf("parseBundleListPage(valid): %v", err)
	}
	if pageSize != 2 || !cursorAt.Equal(items[0].ModifiedAt) || cursorID != items[0].ID || !includeDisabled {
		t.Fatalf(
			"parseBundleListPage(valid) = pageSize=%d cursorAt=%v cursorID=%q includeDisabled=%v",
			pageSize,
			cursorAt,
			cursorID,
			includeDisabled,
		)
	}
	if !slices.Equal(filterIDs, []bundleitemutils.BundleID{storeTestBundleC, storeTestBundleA}) {
		t.Fatalf("parseBundleListPage filterIDs = %#v", filterIDs)
	}

	if _, _, _, _, _, err := parseBundleListPage(&spec.ListMCPBundlesRequest{
		PageToken: "not-base64",
	}); err == nil || !strings.Contains(err.Error(), "bad pageToken") {
		t.Fatalf("parseBundleListPage(bad token) = %v", err)
	}

	badCursor := jsonutil.Base64JSONEncode(spec.MCPBundlePageToken{
		PageSize:  2,
		CursorMod: "not-a-time",
		CursorID:  storeTestBundleA,
	})
	if _, _, _, _, _, err := parseBundleListPage(&spec.ListMCPBundlesRequest{
		PageToken: badCursor,
	}); err == nil || !strings.Contains(err.Error(), "cannot parse") {
		t.Fatalf("parseBundleListPage(bad cursor) = %v", err)
	}
}

func TestBuiltInDataLifecycleAndOverlayPersistence(t *testing.T) {
	t.Run("constructor validation", func(t *testing.T) {
		if _, err := NewBuiltInData(t.Context(), "", 0); err == nil ||
			!strings.Contains(err.Error(), "overlayBaseDir") {
			t.Fatalf("NewBuiltInData(empty) = %v", err)
		}
	})

	t.Run("toggle and reopen persists", func(t *testing.T) {
		overlayDir := t.TempDir()

		data, err := NewBuiltInData(t.Context(), overlayDir, 0)
		if err != nil {
			t.Fatalf("NewBuiltInData: %v", err)
		}
		defer func() {
			if data != nil {
				_ = data.Close()
			}
		}()

		bundles, _, err := data.ListBuiltInData(t.Context())
		if err != nil {
			t.Fatalf("ListBuiltInData: %v", err)
		}
		if len(bundles) == 0 {
			t.Fatalf("no built-in bundles")
		}

		bundleID, serverID := mustSelectBuiltInPair(t, data)

		bundle, err := data.GetBuiltInBundle(t.Context(), bundleID)
		if err != nil {
			t.Fatalf("GetBuiltInBundle: %v", err)
		}
		server, err := data.GetBuiltInServer(t.Context(), bundleID, serverID)
		if err != nil {
			t.Fatalf("GetBuiltInServer: %v", err)
		}
		if _, err := data.FindBuiltInServerByID(t.Context(), serverID); err != nil {
			t.Fatalf("FindBuiltInServerByID: %v", err)
		}

		cloneBundle := bundle
		cloneBundle.DisplayName = "mutated"
		bundleAgain, err := data.GetBuiltInBundle(t.Context(), bundleID)
		if err != nil {
			t.Fatalf("GetBuiltInBundle #2: %v", err)
		}
		if bundleAgain.DisplayName == "mutated" {
			t.Fatalf("GetBuiltInBundle returned shared state")
		}

		cloneServer := server
		cloneServer.DisplayName = "mutated"
		serverAgain, err := data.GetBuiltInServer(t.Context(), bundleID, serverID)
		if err != nil {
			t.Fatalf("GetBuiltInServer #2: %v", err)
		}
		if serverAgain.DisplayName == "mutated" {
			t.Fatalf("GetBuiltInServer returned shared state")
		}

		if _, err := data.GetBuiltInBundle(t.Context(), "missing-bundle"); err == nil ||
			!strings.Contains(err.Error(), "mcp bundle not found") {
			t.Fatalf("GetBuiltInBundle(missing) = %v", err)
		}
		if _, err := data.GetBuiltInServer(t.Context(), bundleID, "missing-server"); err == nil ||
			!strings.Contains(err.Error(), "mcp server not found") {
			t.Fatalf("GetBuiltInServer(missing) = %v", err)
		}
		if _, err := data.FindBuiltInServerByID(t.Context(), "missing-server"); err == nil ||
			!strings.Contains(err.Error(), "mcp server not found") {
			t.Fatalf("FindBuiltInServerByID(missing) = %v", err)
		}

		bundleEnabled := !bundle.IsEnabled
		serverEnabled := !server.Enabled

		toggledBundle, err := data.SetBundleEnabled(t.Context(), bundleID, bundleEnabled)
		if err != nil {
			t.Fatalf("SetBundleEnabled: %v", err)
		}
		if toggledBundle.IsEnabled != bundleEnabled {
			t.Fatalf("SetBundleEnabled returned %#v, want enabled=%v", toggledBundle, bundleEnabled)
		}

		toggledServer, err := data.SetServerEnabled(t.Context(), bundleID, serverID, serverEnabled)
		if err != nil {
			t.Fatalf("SetServerEnabled: %v", err)
		}
		if toggledServer.Enabled != serverEnabled {
			t.Fatalf("SetServerEnabled returned %#v, want enabled=%v", toggledServer, serverEnabled)
		}

		if err := data.Close(); err != nil {
			t.Fatalf("Close(data): %v", err)
		}
		data = nil

		reopened, err := NewBuiltInData(t.Context(), overlayDir, 0)
		if err != nil {
			t.Fatalf("NewBuiltInData(reopen): %v", err)
		}
		t.Cleanup(func() { _ = reopened.Close() })

		bundle2, err := reopened.GetBuiltInBundle(t.Context(), bundleID)
		if err != nil {
			t.Fatalf("GetBuiltInBundle(reopen): %v", err)
		}
		if bundle2.IsEnabled != bundleEnabled {
			t.Fatalf("bundle enabled after reopen = %v, want %v", bundle2.IsEnabled, bundleEnabled)
		}

		server2, err := reopened.GetBuiltInServer(t.Context(), bundleID, serverID)
		if err != nil {
			t.Fatalf("GetBuiltInServer(reopen): %v", err)
		}
		if server2.Enabled != serverEnabled {
			t.Fatalf("server enabled after reopen = %v, want %v", server2.Enabled, serverEnabled)
		}

		list2, _, err := reopened.ListBuiltInData(t.Context())
		if err != nil {
			t.Fatalf("ListBuiltInData(reopen): %v", err)
		}
		if len(list2) == 0 {
			t.Fatalf("ListBuiltInData(reopen) empty")
		}
	})
}

func TestStoreFunctionalBundleServerLifecycle(t *testing.T) {
	ctx := t.Context()
	st, err := NewMCPStore(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// Base bundle hydration and built-in bundle listing.
	baseBundles, err := st.ListMCPBundles(ctx, &spec.ListMCPBundlesRequest{
		BundleIDs:       []bundleitemutils.BundleID{spec.BaseMCPBundleID},
		IncludeDisabled: true,
	})
	if err != nil {
		t.Fatalf("ListMCPBundles(base): %v", err)
	}
	if len(baseBundles.Body.Bundles) != 1 || baseBundles.Body.Bundles[0].ID != spec.BaseMCPBundleID {
		t.Fatalf("base bundle listing = %#v", baseBundles.Body.Bundles)
	}

	bundleA := bundleitemutils.BundleID(storeTestBundleA)
	bundleB := bundleitemutils.BundleID(storeTestBundleB)
	bundleC := bundleitemutils.BundleID(storeTestBundleC)
	bundleD := bundleitemutils.BundleID(storeTestBundleD)

	mustCreateBundle(t, st, bundleA, bundleitemutils.BundleSlug(storeTestBundleA), "Bundle A", true)
	mustCreateBundle(t, st, bundleB, bundleitemutils.BundleSlug(storeTestBundleB), "Bundle B", true)

	// Duplicate slug conflict.
	_, err = st.PutMCPBundle(ctx, &spec.PutMCPBundleRequest{
		BundleID: bundleC,
		Body: &spec.PutMCPBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug(storeTestBundleA),
			DisplayName: "Bundle C",
			IsEnabled:   true,
		},
	})
	if !errors.Is(err, spec.ErrMCPConflict) || !strings.Contains(err.Error(), "bundle slug") {
		t.Fatalf("PutMCPBundle duplicate slug = %v", err)
	}

	// Reserved base slug conflict.
	_, err = st.PutMCPBundle(ctx, &spec.PutMCPBundleRequest{
		BundleID: bundleD,
		Body: &spec.PutMCPBundleRequestBody{
			Slug:        spec.BaseMCPBundleSlug,
			DisplayName: "Bundle D",
			IsEnabled:   true,
		},
	})
	if !errors.Is(err, spec.ErrMCPConflict) || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("PutMCPBundle reserved slug = %v", err)
	}

	// Built-in bundle/server read-only guards.
	if st.builtinData == nil {
		t.Fatalf("builtinData is nil")
	}
	builtInBundleID, builtInServerID := mustSelectBuiltInPair(t, st.builtinData)

	_, err = st.PutMCPBundle(ctx, &spec.PutMCPBundleRequest{
		BundleID: builtInBundleID,
		Body: &spec.PutMCPBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug("built-in-copy"),
			DisplayName: "Built-in copy",
			IsEnabled:   true,
		},
	})
	if !errors.Is(err, spec.ErrMCPBuiltInReadOnly) {
		t.Fatalf("PutMCPBundle(built-in) = %v, want ErrMCPBuiltInReadOnly", err)
	}

	_, err = st.PatchMCPBundle(ctx, &spec.PatchMCPBundleRequest{
		BundleID: builtInBundleID,
		Body:     &spec.PatchMCPBundleRequestBody{IsEnabled: false},
	})
	if err != nil {
		t.Fatalf("go err in patching builtin %v", err)
	}

	_, err = st.DeleteMCPBundle(ctx, &spec.DeleteMCPBundleRequest{BundleID: builtInBundleID})
	if !errors.Is(err, spec.ErrMCPBuiltInReadOnly) {
		t.Fatalf("DeleteMCPBundle(built-in) = %v, want ErrMCPBuiltInReadOnly", err)
	}

	_, err = st.PutMCPServer(ctx, &spec.PutMCPServerRequest{
		BundleID: bundleA,
		ServerID: builtInServerID,
		Body: &spec.PutMCPServerPayload{
			DisplayName: "Built-in server copy",
			Enabled:     true,
			Transport:   spec.MCPTransportStreamableHTTP,
			StreamableHTTP: &spec.MCPStreamableHTTPConfig{
				URL:      "http://127.0.0.1:1234/mcp",
				AuthMode: spec.MCPHTTPAuthNone,
			},
			DefaultPolicy: func() *spec.MCPServerPolicy {
				p := spec.DefaultMCPServerPolicy()
				return &p
			}(),
		},
	})
	if !errors.Is(err, spec.ErrMCPConflict) {
		t.Fatalf("PutMCPServer(built-in ID conflict) = %v, want ErrMCPConflict", err)
	}

	_, err = st.PatchMCPServerEnabled(ctx, &spec.PatchMCPServerEnabledRequest{
		BundleID: builtInBundleID,
		ServerID: builtInServerID,
		Body:     &spec.PatchMCPServerEnabledRequestBody{Enabled: false},
	})
	if err != nil {
		t.Fatalf("PatchMCPServerEnabled(built-in) = %v, want nil", err)
	}

	_, err = st.DeleteMCPServer(ctx, &spec.DeleteMCPServerRequest{
		BundleID: builtInBundleID,
		ServerID: builtInServerID,
	})
	if !errors.Is(err, spec.ErrMCPBuiltInReadOnly) {
		t.Fatalf("DeleteMCPServer(built-in) = %v, want ErrMCPBuiltInReadOnly", err)
	}

	// User bundle server lifecycle.
	mustCreateHTTPServer(t, st, bundleA, storeTestServerA, "Server A", true)

	_, err = st.PatchMCPBundle(ctx, &spec.PatchMCPBundleRequest{
		BundleID: bundleB,
		Body:     &spec.PatchMCPBundleRequestBody{IsEnabled: false},
	})
	if err != nil {
		t.Fatalf("PatchMCPBundle(bundleB disable): %v", err)
	}

	enabledOnly, err := st.ListMCPBundles(ctx, &spec.ListMCPBundlesRequest{
		BundleIDs: []bundleitemutils.BundleID{bundleA, bundleB},
	})
	if err != nil {
		t.Fatalf("ListMCPBundles(enabledOnly): %v", err)
	}
	if len(enabledOnly.Body.Bundles) != 1 || enabledOnly.Body.Bundles[0].ID != bundleA {
		t.Fatalf("ListMCPBundles(enabledOnly) = %#v, want only bundleA", enabledOnly.Body.Bundles)
	}

	page1, err := st.ListMCPBundles(ctx, &spec.ListMCPBundlesRequest{
		BundleIDs:       []bundleitemutils.BundleID{bundleA, bundleB},
		IncludeDisabled: true,
		PageSize:        1,
	})
	if err != nil {
		t.Fatalf("ListMCPBundles(page1): %v", err)
	}
	if len(page1.Body.Bundles) != 1 || page1.Body.NextPageToken == nil {
		t.Fatalf("ListMCPBundles(page1) = %#v", page1.Body)
	}
	page2, err := st.ListMCPBundles(ctx, &spec.ListMCPBundlesRequest{
		PageToken: *page1.Body.NextPageToken,
	})
	if err != nil {
		t.Fatalf("ListMCPBundles(page2): %v", err)
	}
	gotIDs := make([]bundleitemutils.BundleID, 0, 2)
	gotIDs = append(gotIDs, page1.Body.Bundles[0].ID)
	if len(page2.Body.Bundles) != 1 {
		t.Fatalf("ListMCPBundles(page2) = %#v", page2.Body)
	}
	gotIDs = append(gotIDs, page2.Body.Bundles[0].ID)
	slices.Sort(gotIDs)
	if !slices.Equal(gotIDs, []bundleitemutils.BundleID{bundleA, bundleB}) {
		t.Fatalf("ListMCPBundles pagination IDs = %#v", gotIDs)
	}

	// Deleting a non-empty bundle should fail first.
	_, err = st.DeleteMCPBundle(ctx, &spec.DeleteMCPBundleRequest{BundleID: bundleA})
	if !errors.Is(err, spec.ErrMCPBundleNotEmpty) {
		t.Fatalf("DeleteMCPBundle(non-empty) = %v, want ErrMCPBundleNotEmpty", err)
	}

	// Delete server, verify deleted server visibility, then delete bundle.
	_, err = st.DeleteMCPServer(ctx, &spec.DeleteMCPServerRequest{
		BundleID: bundleA,
		ServerID: storeTestServerA,
	})
	if err != nil {
		t.Fatalf("DeleteMCPServer(bundleA/serverA): %v", err)
	}

	_, err = st.GetMCPServer(ctx, &spec.GetMCPServerRequest{
		BundleID: bundleA,
		ServerID: storeTestServerA,
	})
	if err == nil {
		t.Fatalf("GetMCPServer: %v", err)
	}
	if !errors.Is(err, spec.ErrMCPServerNotFound) {
		t.Fatalf("GetMCPServer(deleted) = %v, want ErrMCPServerNotFound", err)
	}

	_, err = st.PutMCPServer(ctx, &spec.PutMCPServerRequest{
		BundleID: bundleA,
		ServerID: storeTestServerA,
		Body: &spec.PutMCPServerPayload{
			DisplayName: "Server A",
			Enabled:     true,
			Transport:   spec.MCPTransportStreamableHTTP,
			StreamableHTTP: &spec.MCPStreamableHTTPConfig{
				URL:      "http://127.0.0.1:1234/mcp",
				AuthMode: spec.MCPHTTPAuthNone,
			},
			DefaultPolicy: func() *spec.MCPServerPolicy {
				p := spec.DefaultMCPServerPolicy()
				return &p
			}(),
		},
	})
	if err != nil {
		t.Fatalf("PutMCPServer(deleted) = %v, want nil", err)
	}

	_, err = st.DeleteMCPBundle(ctx, &spec.DeleteMCPBundleRequest{BundleID: bundleA})
	if err == nil {
		t.Fatalf("DeleteMCPBundle(bundleA): want err got nil")
	}
}

func TestStoreNewBuiltInDataAndUtilityErrors(t *testing.T) {
	overlayDir := t.TempDir()

	data, err := NewBuiltInData(t.Context(), overlayDir, time.Minute)
	if err != nil {
		t.Fatalf("NewBuiltInData: %v", err)
	}
	t.Cleanup(func() { _ = data.Close() })

	bundles, servers, err := data.ListBuiltInData(t.Context())
	if err != nil {
		t.Fatalf("ListBuiltInData: %v", err)
	}
	if len(bundles) == 0 || len(servers) == 0 {
		t.Fatalf("built-in data empty: bundles=%d servers=%d", len(bundles), len(servers))
	}

	bundleID, serverID := mustSelectBuiltInPair(t, data)
	if _, err := data.GetBuiltInBundle(t.Context(), bundleID); err != nil {
		t.Fatalf("GetBuiltInBundle: %v", err)
	}
	if _, err := data.GetBuiltInServer(t.Context(), bundleID, serverID); err != nil {
		t.Fatalf("GetBuiltInServer: %v", err)
	}
	if _, err := data.FindBuiltInServerByID(t.Context(), serverID); err != nil {
		t.Fatalf("FindBuiltInServerByID: %v", err)
	}

	if _, err := data.GetBuiltInBundle(t.Context(), "missing"); err == nil ||
		!strings.Contains(err.Error(), "mcp bundle not found") {
		t.Fatalf("GetBuiltInBundle(missing) = %v", err)
	}
	if _, err := data.GetBuiltInServer(t.Context(), bundleID, "missing"); err == nil ||
		!strings.Contains(err.Error(), "mcp server not found") {
		t.Fatalf("GetBuiltInServer(missing) = %v", err)
	}
	if _, err := data.FindBuiltInServerByID(t.Context(), "missing"); err == nil ||
		!strings.Contains(err.Error(), "mcp server not found") {
		t.Fatalf("FindBuiltInServerByID(missing) = %v", err)
	}

	bundle, err := data.GetBuiltInBundle(t.Context(), bundleID)
	if err != nil {
		t.Fatalf("GetBuiltInBundle: %v", err)
	}
	server, err := data.GetBuiltInServer(t.Context(), bundleID, serverID)
	if err != nil {
		t.Fatalf("GetBuiltInServer: %v", err)
	}

	wantBundleEnabled := !bundle.IsEnabled
	wantServerEnabled := !server.Enabled

	if _, err := data.SetBundleEnabled(t.Context(), bundleID, wantBundleEnabled); err != nil {
		t.Fatalf("SetBundleEnabled: %v", err)
	}
	if _, err := data.SetServerEnabled(t.Context(), bundleID, serverID, wantServerEnabled); err != nil {
		t.Fatalf("SetServerEnabled: %v", err)
	}

	if err := data.Close(); err != nil {
		t.Fatalf("Close(data): %v", err)
	}
	data = nil

	reopened, err := NewBuiltInData(t.Context(), overlayDir, time.Minute)
	if err != nil {
		t.Fatalf("NewBuiltInData(reopen): %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	bundle2, err := reopened.GetBuiltInBundle(t.Context(), bundleID)
	if err != nil {
		t.Fatalf("GetBuiltInBundle(reopen): %v", err)
	}
	if bundle2.IsEnabled != wantBundleEnabled {
		t.Fatalf("bundle enabled after reopen = %v, want %v", bundle2.IsEnabled, wantBundleEnabled)
	}

	server2, err := reopened.GetBuiltInServer(t.Context(), bundleID, serverID)
	if err != nil {
		t.Fatalf("GetBuiltInServer(reopen): %v", err)
	}
	if server2.Enabled != wantServerEnabled {
		t.Fatalf("server enabled after reopen = %v, want %v", server2.Enabled, wantServerEnabled)
	}
}

func TestStoreListBundlesAndServersWithBadTokens(t *testing.T) {
	st, err := NewMCPStore(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if _, err := st.ListMCPBundles(t.Context(), &spec.ListMCPBundlesRequest{
		PageToken: "not-base64",
	}); err == nil || !strings.Contains(err.Error(), "bad pageToken") {
		t.Fatalf("ListMCPBundles(bad token) = %v", err)
	}

	if _, err := st.ListMCPBundles(t.Context(), &spec.ListMCPBundlesRequest{
		PageToken: jsonutil.Base64JSONEncode(spec.MCPBundlePageToken{
			PageSize:  1,
			CursorMod: "not-a-time",
		}),
	}); err == nil || !strings.Contains(err.Error(), "cannot parse") {
		t.Fatalf("ListMCPBundles(bad cursor time) = %v", err)
	}
}

func newValidStoreBundle(
	id bundleitemutils.BundleID,
	slug bundleitemutils.BundleSlug,
	displayName string,
	enabled bool,
) spec.MCPBundle {
	now := time.Now().UTC()
	return spec.MCPBundle{
		SchemaVersion: spec.MCPSchemaVersion,
		ID:            id,
		Slug:          slug,
		DisplayName:   displayName,
		Description:   displayName + " bundle",
		IsEnabled:     enabled,
		CreatedAt:     now.Add(-time.Minute),
		ModifiedAt:    now,
	}
}

func newValidStoreHTTPServer(
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
		Transport:     spec.MCPTransportStreamableHTTP,
		StreamableHTTP: &spec.MCPStreamableHTTPConfig{
			URL:      "http://127.0.0.1:1234/mcp",
			AuthMode: spec.MCPHTTPAuthNone,
		},
		DefaultPolicy: spec.DefaultMCPServerPolicy(),
		CreatedAt:     now.Add(-time.Minute),
		ModifiedAt:    now,
	}
}

func mustCreateBundle(
	t *testing.T,
	st *Store,
	bundleID bundleitemutils.BundleID,
	slug bundleitemutils.BundleSlug,
	displayName string,
	enabled bool,
) {
	t.Helper()

	_, err := st.PutMCPBundle(t.Context(), &spec.PutMCPBundleRequest{
		BundleID: bundleID,
		Body: &spec.PutMCPBundleRequestBody{
			Slug:        slug,
			DisplayName: displayName,
			IsEnabled:   enabled,
			Description: displayName + " bundle",
		},
	})
	if err != nil {
		t.Fatalf("PutMCPBundle(%s): %v", bundleID, err)
	}
}

func mustCreateHTTPServer(
	t *testing.T,
	st *Store,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	displayName string,
	enabled bool,
) {
	t.Helper()

	_, err := st.PutMCPServer(t.Context(), &spec.PutMCPServerRequest{
		BundleID: bundleID,
		ServerID: serverID,
		Body: &spec.PutMCPServerPayload{
			DisplayName: displayName,
			Enabled:     enabled,
			Transport:   spec.MCPTransportStreamableHTTP,
			StreamableHTTP: &spec.MCPStreamableHTTPConfig{
				URL:      "http://127.0.0.1:1234/mcp",
				AuthMode: spec.MCPHTTPAuthNone,
			},
			DefaultPolicy: func() *spec.MCPServerPolicy {
				p := spec.DefaultMCPServerPolicy()
				return &p
			}(),
		},
	})
	if err != nil {
		t.Fatalf("PutMCPServer(%s/%s): %v", bundleID, serverID, err)
	}
}

func mustSelectBuiltInPair(t *testing.T, data *BuiltInData) (bundleitemutils.BundleID, spec.MCPServerID) {
	t.Helper()

	bundles, servers, err := data.ListBuiltInData(t.Context())
	if err != nil {
		t.Fatalf("ListBuiltInData: %v", err)
	}
	if len(bundles) == 0 {
		t.Fatalf("no built-in bundles")
	}
	bundleIDs := make([]string, 0, len(bundles))
	for id := range bundles {
		bundleIDs = append(bundleIDs, string(id))
	}
	slices.Sort(bundleIDs)
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
		slices.Sort(serverIDs)
		return bid, spec.MCPServerID(serverIDs[0])
	}
	t.Fatalf("no built-in bundle with servers")
	return "", ""
}

func putTestBundle(t *testing.T, st *Store, bundleID bundleitemutils.BundleID, displayName string) {
	t.Helper()
	if _, err := st.PutMCPBundle(t.Context(), &spec.PutMCPBundleRequest{
		BundleID: bundleID,
		Body: &spec.PutMCPBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug(bundleID),
			DisplayName: displayName,
			IsEnabled:   true,
			Description: displayName + " bundle",
		},
	}); err != nil {
		t.Fatalf("PutMCPBundle(%s): %v", bundleID, err)
	}
}
