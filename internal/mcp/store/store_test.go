package store

import (
	"errors"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestStorePutGetListPatchDeleteAndPersistence(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()

	st, err := NewMCPStore(t.Context(), dir)
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

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
	betaSecretRef, err := secret.NewMCPSecretRefString(betaID, spec.MCPSecretKindStdioEnv, "TOKEN")
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
		ServerID: alphaID,
		Body:     alphaPayload,
	}); err != nil {
		t.Fatalf("PutMCPServer(alpha): %v", err)
	}
	if _, err := st.PutMCPServer(ctx, &spec.PutMCPServerRequest{
		ServerID: betaID,
		Body:     betaPayload,
	}); err != nil {
		t.Fatalf("PutMCPServer(beta): %v", err)
	}

	gotAlpha, err := st.GetMCPServer(ctx, &spec.GetMCPServerRequest{ServerID: alphaID})
	if err != nil {
		t.Fatalf("GetMCPServer(alpha): %v", err)
	}
	if gotAlpha.Body.DisplayName != "Alpha HTTP" {
		t.Fatalf("Alpha DisplayName = %q, want %q", gotAlpha.Body.DisplayName, "Alpha HTTP")
	}
	if gotAlpha.Body.Availability != spec.MCPServerAvailabilityManual {
		t.Fatalf("Alpha Availability = %q, want manual", gotAlpha.Body.Availability)
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
	gotAlpha2, err := st.GetMCPServer(ctx, &spec.GetMCPServerRequest{ServerID: alphaID})
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
		ServerID: betaID,
		Body:     &spec.PatchMCPServerEnabledRequestBody{Enabled: false},
	}); err != nil {
		t.Fatalf("PatchMCPServerEnabled(beta): %v", err)
	}

	enabledOnly, err := st.ListMCPServers(ctx, &spec.ListMCPServersRequest{
		Enabled: new(true),
	})
	if err != nil {
		t.Fatalf("ListMCPServers(enabled=true): %v", err)
	}
	if len(enabledOnly.Body.Servers) != 1 || enabledOnly.Body.Servers[0].ID != alphaID {
		t.Fatalf("ListMCPServers(enabled=true) = %#v, want only alpha", enabledOnly.Body.Servers)
	}

	page1, err := st.ListMCPServers(ctx, &spec.ListMCPServersRequest{
		PageSize: 1,
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
		PageToken: *page1.Body.NextPageToken,
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

	snap := spec.MCPDiscoverySnapshot{
		ServerID: alphaID,
		Tools: []spec.MCPToolCapability{
			{
				ServerID:    alphaID,
				ToolName:    "echo",
				DisplayName: "Echo",
				Digest:      "digest-echo",
			},
		},
	}
	if err := st.SaveLastKnownSnapshot(ctx, snap); err != nil {
		t.Fatalf("SaveLastKnownSnapshot(alpha): %v", err)
	}
	gotSnap, ok, err := st.GetLastKnownSnapshot(ctx, alphaID)
	if err != nil {
		t.Fatalf("GetLastKnownSnapshot(alpha): %v", err)
	}
	if !ok {
		t.Fatalf("GetLastKnownSnapshot(alpha): ok=false")
	}
	if len(gotSnap.Tools) != 1 || gotSnap.Tools[0].ToolName != "echo" {
		t.Fatalf("snapshot = %#v", gotSnap.Tools)
	}
	gotSnap.Tools[0].ToolName = "mutated"
	gotSnap2, ok, err := st.GetLastKnownSnapshot(ctx, alphaID)
	if err != nil {
		t.Fatalf("GetLastKnownSnapshot(alpha #2): %v", err)
	}
	if !ok {
		t.Fatalf("GetLastKnownSnapshot(alpha #2): ok=false")
	}
	if gotSnap2.Tools[0].ToolName != "echo" {
		t.Fatalf("snapshot clone not preserved: %#v", gotSnap2.Tools)
	}

	if err := st.Close(); err != nil {
		t.Fatalf("Close(store): %v", err)
	}

	st2, err := NewMCPStore(t.Context(), dir)
	if err != nil {
		t.Fatalf("NewMCPStore(reopen): %v", err)
	}
	t.Cleanup(func() { _ = st2.Close() })

	reAlpha, err := st2.GetMCPServer(ctx, &spec.GetMCPServerRequest{ServerID: alphaID})
	if err != nil {
		t.Fatalf("GetMCPServer(alpha reopen): %v", err)
	}
	if reAlpha.Body.DefaultPolicy.DefaultApprovalRule != spec.MCPApprovalRuleDeny ||
		reAlpha.Body.DefaultPolicy.DefaultExecutionMode != spec.MCPExecutionModeAuto {
		t.Fatalf("patched policy not persisted: %#v", reAlpha.Body.DefaultPolicy)
	}

	reBeta, err := st2.GetMCPServer(ctx, &spec.GetMCPServerRequest{ServerID: betaID})
	if err != nil {
		t.Fatalf("GetMCPServer(beta reopen): %v", err)
	}
	if reBeta.Body.Enabled {
		t.Fatalf("beta Enabled = true, want false")
	}

	reSnap, ok, err := st2.GetLastKnownSnapshot(ctx, alphaID)
	if err != nil {
		t.Fatalf("GetLastKnownSnapshot(alpha reopen): %v", err)
	}
	if !ok {
		t.Fatalf("snapshot missing after reopen")
	}
	if len(reSnap.Tools) != 1 || reSnap.Tools[0].ToolName != "echo" {
		t.Fatalf("snapshot not persisted: %#v", reSnap.Tools)
	}

	if _, err := st2.DeleteMCPServer(ctx, &spec.DeleteMCPServerRequest{ServerID: alphaID}); err != nil {
		t.Fatalf("DeleteMCPServer(alpha): %v", err)
	}
	if _, err := st2.GetMCPServer(
		ctx,
		&spec.GetMCPServerRequest{ServerID: alphaID},
	); !errors.Is(
		err,
		spec.ErrMCPServerDeleting,
	) {
		t.Fatalf("GetMCPServer(alpha deleted) error = %v, want ErrMCPServerDeleting", err)
	}
	deleted, err := st2.GetMCPServer(ctx, &spec.GetMCPServerRequest{
		ServerID:       alphaID,
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("GetMCPServer(alpha deleted, includeDeleted): %v", err)
	}
	if deleted.Body.SoftDeletedAt == nil {
		t.Fatalf("deleted server SoftDeletedAt is nil")
	}
	if _, ok, err := st2.GetLastKnownSnapshot(ctx, alphaID); err != nil {
		t.Fatalf("GetLastKnownSnapshot(alpha deleted): %v", err)
	} else if ok {
		t.Fatalf("snapshot still present after delete")
	}

	listAfterDelete, err := st2.ListMCPServers(ctx, &spec.ListMCPServersRequest{})
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
		make func(id spec.MCPServerID) *spec.PutMCPServerPayload
		want string
	}{
		{
			name: "stdio missing config",
			make: func(id spec.MCPServerID) *spec.PutMCPServerPayload {
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
			make: func(id spec.MCPServerID) *spec.PutMCPServerPayload {
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
			make: func(id spec.MCPServerID) *spec.PutMCPServerPayload {
				ref, err := secret.NewMCPSecretRefString(id, spec.MCPSecretKindStdioEnv, "TOKEN")
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
			make: func(id spec.MCPServerID) *spec.PutMCPServerPayload {
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
			make: func(id spec.MCPServerID) *spec.PutMCPServerPayload {
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
			make: func(id spec.MCPServerID) *spec.PutMCPServerPayload {
				ref, err := secret.NewMCPSecretRefString(
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

			id := spec.MCPServerID("server")
			_, err = st.PutMCPServer(t.Context(), &spec.PutMCPServerRequest{
				ServerID: id,
				Body:     tt.make(id),
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

func TestStoreRejectsBadPageToken(t *testing.T) {
	st, err := NewMCPStore(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if _, err := st.ListMCPServers(t.Context(), &spec.ListMCPServersRequest{
		PageToken: "not-a-token",
	}); !errors.Is(err, spec.ErrMCPInvalidRequest) {
		t.Fatalf("ListMCPServers(bad token) error = %v, want ErrMCPInvalidRequest", err)
	}
}
