package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestCloneDiscoverySnapshotMakesDeepCopy(t *testing.T) {
	orig := spec.MCPDiscoverySnapshot{
		ServerID: "server",
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
				ServerID:         "server",
				ToolName:         "tool",
				ProviderToolName: "provider",
				ChoiceID:         "choice",
				Digest:           "digest",
				InputSchema:      map[string]any{"a": 1},
				OutputSchema:     map[string]any{"b": 2},
				Annotations:      &spec.MCPToolAnnotations{Title: "tool-annotation"},
				App: &spec.MCPToolAppInfo{
					ResourceURI: "ui://demo",
					Visibility:  []string{"model", "app"},
				},
			},
		},
		Resources: []spec.MCPResourceRef{
			{
				ServerID:    "server",
				URI:         "res://a",
				Annotations: map[string]any{"k": "v"},
			},
		},
		ResourceTemplates: []spec.MCPResourceTemplateRef{
			{
				ServerID:    "server",
				URITemplate: "tmpl://{x}",
				Arguments:   map[string]string{"x": "1"},
				Annotations: map[string]any{"a": "b"},
			},
		},
		Prompts: []spec.MCPPromptRef{
			{
				ServerID:   "server",
				PromptName: "prompt-a",
				Arguments:  map[string]string{"p": "q"},
			},
		},
	}

	clone := cloneDiscoverySnapshot(orig)

	orig.ServerInfo.Name = "changed"
	orig.ServerCapabilities.Experimental["x"] = 99
	orig.ServerCapabilities.Extensions["y"] = 88
	orig.Tools[0].InputSchema["a"] = 42
	orig.Tools[0].OutputSchema["b"] = 43
	orig.Tools[0].Annotations.Title = "changed"
	orig.Tools[0].App.Visibility[0] = "changed"
	orig.Resources[0].Annotations["k"] = "changed"
	orig.ResourceTemplates[0].Arguments["x"] = "changed"
	orig.ResourceTemplates[0].Annotations["a"] = "changed"
	orig.Prompts[0].Arguments["p"] = "changed"

	if clone.ServerInfo == nil || clone.ServerInfo.Name != "name" || clone.ServerInfo.Version != "v1" {
		t.Fatalf("ServerInfo was not deep-cloned: %#v", clone.ServerInfo)
	}
	if clone.ServerCapabilities == nil || clone.ServerCapabilities.Experimental["x"] != 1 ||
		clone.ServerCapabilities.Extensions["y"] != 2 {
		t.Fatalf("ServerCapabilities were not deep-cloned: %#v", clone.ServerCapabilities)
	}
	if clone.Tools[0].InputSchema["a"] != 1 || clone.Tools[0].OutputSchema["b"] != 2 {
		t.Fatalf("Tool schemas were not deep-cloned: %#v", clone.Tools[0])
	}
	if clone.Tools[0].Annotations == nil || clone.Tools[0].Annotations.Title != "tool-annotation" {
		t.Fatalf("Tool annotations were not deep-cloned: %#v", clone.Tools[0].Annotations)
	}
	if clone.Tools[0].App == nil || clone.Tools[0].App.Visibility[0] != "model" {
		t.Fatalf("Tool app info was not deep-cloned: %#v", clone.Tools[0].App)
	}
	if clone.Resources[0].Annotations["k"] != "v" {
		t.Fatalf("Resource annotations were not deep-cloned: %#v", clone.Resources[0].Annotations)
	}
	if clone.ResourceTemplates[0].Arguments["x"] != "1" || clone.ResourceTemplates[0].Annotations["a"] != "b" {
		t.Fatalf("Resource template fields were not deep-cloned: %#v", clone.ResourceTemplates[0])
	}
	if clone.Prompts[0].Arguments["p"] != "q" {
		t.Fatalf("Prompt arguments were not deep-cloned: %#v", clone.Prompts[0].Arguments)
	}
}

func TestNormalizeSnapshotSortsAndDigestIsStable(t *testing.T) {
	snap1 := spec.MCPDiscoverySnapshot{
		ServerID: "server",
		Tools: []spec.MCPToolCapability{
			{ServerID: "server", ToolName: "beta", Digest: "2"},
			{ServerID: "server", ToolName: "alpha", Digest: "1"},
		},
		Resources: []spec.MCPResourceRef{
			{ServerID: "server", URI: "z://res"},
			{ServerID: "server", URI: "a://res"},
		},
		ResourceTemplates: []spec.MCPResourceTemplateRef{
			{ServerID: "server", URITemplate: "z://tmpl"},
			{ServerID: "server", URITemplate: "a://tmpl"},
		},
		Prompts: []spec.MCPPromptRef{
			{ServerID: "server", PromptName: "z-prompt"},
			{ServerID: "server", PromptName: "a-prompt"},
		},
	}

	snap2 := cloneDiscoverySnapshot(snap1)

	normalizeSnapshot(&snap1)
	time.Sleep(2 * time.Millisecond)
	normalizeSnapshot(&snap2)

	if snap1.SyncedAt == "" || snap1.Digest == "" {
		t.Fatalf("normalized snapshot missing syncedAt/digest: %#v", snap1)
	}
	if snap1.Tools[0].ToolName != "alpha" || snap1.Tools[1].ToolName != "beta" {
		t.Fatalf("tools not sorted: %#v", snap1.Tools)
	}
	if snap1.Resources[0].URI != "a://res" || snap1.Resources[1].URI != "z://res" {
		t.Fatalf("resources not sorted: %#v", snap1.Resources)
	}
	if snap1.ResourceTemplates[0].URITemplate != "a://tmpl" || snap1.ResourceTemplates[1].URITemplate != "z://tmpl" {
		t.Fatalf("resource templates not sorted: %#v", snap1.ResourceTemplates)
	}
	if snap1.Prompts[0].PromptName != "a-prompt" || snap1.Prompts[1].PromptName != "z-prompt" {
		t.Fatalf("prompts not sorted: %#v", snap1.Prompts)
	}

	if got := computeDiscoverySnapshotDigest(snap1); got != snap1.Digest {
		t.Fatalf("Digest = %q, want computeDiscoverySnapshotDigest = %q", snap1.Digest, got)
	}
	if snap1.Digest != snap2.Digest {
		t.Fatalf("digest not stable across equivalent snapshots: %q vs %q", snap1.Digest, snap2.Digest)
	}
}

func TestMergeResolvedTransportAuth(t *testing.T) {
	t.Run("copies env, sensitive values, and status", func(t *testing.T) {
		dst := &auth.ResolvedTransportAuth{
			Env: map[string]string{
				"KEEP": "1",
			},
			SensitiveValues: []string{"old"},
			Status: spec.MCPAuthStatus{
				ServerID: "server",
				AuthMode: spec.MCPHTTPAuthNone,
				State:    spec.MCPAuthStateNotRequired,
			},
		}
		src := auth.ResolvedTransportAuth{
			Env: map[string]string{
				"KEEP": "2",
				"NEW":  "3",
			},
			SensitiveValues: []string{"secret-a", "secret-b"},
			Status: spec.MCPAuthStatus{
				ServerID: "server",
				AuthMode: spec.MCPHTTPAuthOAuth,
				State:    spec.MCPAuthStateRequired,
				Resource: "https://example.test/mcp",
			},
		}

		mergeResolvedTransportAuth(dst, src)

		if dst.Env["KEEP"] != "2" || dst.Env["NEW"] != "3" {
			t.Fatalf("Env = %#v", dst.Env)
		}
		if len(dst.SensitiveValues) != 3 || dst.SensitiveValues[0] != "old" || dst.SensitiveValues[1] != "secret-a" ||
			dst.SensitiveValues[2] != "secret-b" {
			t.Fatalf("SensitiveValues = %#v", dst.SensitiveValues)
		}
		if dst.Status.AuthMode != spec.MCPHTTPAuthOAuth || dst.Status.State != spec.MCPAuthStateRequired ||
			dst.Status.Resource != "https://example.test/mcp" {
			t.Fatalf("Status = %#v", dst.Status)
		}
	})

	t.Run("empty source status does not overwrite destination status", func(t *testing.T) {
		dst := &auth.ResolvedTransportAuth{
			Env: map[string]string{
				"KEEP": "1",
			},
			Status: spec.MCPAuthStatus{
				ServerID: "server",
				AuthMode: spec.MCPHTTPAuthNone,
				State:    spec.MCPAuthStateNotRequired,
			},
		}

		mergeResolvedTransportAuth(dst, auth.ResolvedTransportAuth{
			Env: map[string]string{
				"NEW": "2",
			},
			Status: spec.MCPAuthStatus{
				AuthMode: spec.MCPHTTPAuthOAuth,
				State:    spec.MCPAuthStateRequired,
			},
		})

		if dst.Status.AuthMode != spec.MCPHTTPAuthNone || dst.Status.State != spec.MCPAuthStateNotRequired {
			t.Fatalf("Status was overwritten unexpectedly: %#v", dst.Status)
		}
		if dst.Env["NEW"] != "2" {
			t.Fatalf("Env was not merged: %#v", dst.Env)
		}
	})
}

func TestApplyToolPolicyOverlayAndEnsureDigest(t *testing.T) {
	t.Run("overlay applies override and stale digest", func(t *testing.T) {
		auto := spec.MCPExecutionModeAuto
		allow := spec.MCPApprovalRuleAllow

		tool := spec.MCPToolCapability{
			ToolName:      "echo",
			Digest:        "current-digest",
			ApprovalRule:  spec.MCPApprovalRuleAsk,
			ExecutionMode: spec.MCPExecutionModeManual,
		}

		cfg := spec.MCPServerConfig{
			ToolPolicies: map[string]spec.MCPToolPolicyOverride{
				"echo": {
					ToolName:         "echo",
					ApprovalRule:     &allow,
					ExecutionMode:    &auto,
					ExpectedDigest:   "different-digest",
					AllowStaleDigest: false,
				},
			},
		}

		got := applyToolPolicyOverlay(tool, cfg)
		if got.ApprovalRule != allow || got.ExecutionMode != auto {
			t.Fatalf("overlay did not apply: %#v", got)
		}
		if !got.Stale {
			t.Fatalf("expected stale tool when digest differs")
		}
	})

	t.Run("EnsureDigest", func(t *testing.T) {
		tests := []struct {
			name       string
			expected   string
			got        string
			allowStale bool
			wantErr    bool
		}{
			{
				name:     "empty expected",
				expected: "",
				got:      "got",
			},
			{
				name:     "same digest",
				expected: "abc",
				got:      "abc",
			},
			{
				name:       "allow stale",
				expected:   "abc",
				got:        "def",
				allowStale: true,
			},
			{
				name:     "mismatch",
				expected: "abc",
				got:      "def",
				wantErr:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := EnsureDigest(tt.expected, tt.got, tt.allowStale)
				if tt.wantErr {
					if !errors.Is(err, spec.ErrMCPStaleReference) {
						t.Fatalf("err = %v, want ErrMCPStaleReference", err)
					}
					return
				}
				if err != nil {
					t.Fatalf("EnsureDigest: %v", err)
				}
			})
		}
	})
}

func TestWithDefaultRequestTimeoutAndIdentifiers(t *testing.T) {
	t.Run("withDefaultRequestTimeout handles nil context and preserves existing deadline", func(t *testing.T) {
		ctx, cancel := withDefaultRequestTimeout(t.Context())
		defer cancel()

		if _, ok := ctx.Deadline(); !ok {
			t.Fatalf("nil context did not receive a deadline")
		}

		base, baseCancel := context.WithTimeout(t.Context(), time.Minute)
		defer baseCancel()

		derived, derivedCancel := withDefaultRequestTimeout(base)
		defer derivedCancel()

		baseDeadline, _ := base.Deadline()
		derivedDeadline, ok := derived.Deadline()
		if !ok {
			t.Fatalf("derived context missing deadline")
		}
		if !baseDeadline.Equal(derivedDeadline) {
			t.Fatalf("deadline changed: base=%v derived=%v", baseDeadline, derivedDeadline)
		}
	})

	t.Run("paginateDiscoveryItems defaults and clamps page size", func(t *testing.T) {
		items := make([]int, 300)
		for i := range items {
			items[i] = i
		}

		out, next, err := paginateDiscoveryItems(
			"server",
			"digest",
			discoveryPageKindTools,
			items,
			0,
			"",
		)
		if err != nil {
			t.Fatalf("paginateDiscoveryItems(default): %v", err)
		}
		if len(out) != spec.DefaultMCPPageSize {
			t.Fatalf("default page size len = %d, want %d", len(out), spec.DefaultMCPPageSize)
		}
		if next == nil {
			t.Fatalf("default page size should have next token")
		}

		out2, next2, err := paginateDiscoveryItems(
			"server",
			"digest",
			discoveryPageKindTools,
			items,
			spec.MaxMCPServerPageSize+100,
			"",
		)
		if err != nil {
			t.Fatalf("paginateDiscoveryItems(clamped): %v", err)
		}
		if len(out2) != spec.MaxMCPServerPageSize {
			t.Fatalf("clamped page size len = %d, want %d", len(out2), spec.MaxMCPServerPageSize)
		}
		if next2 == nil {
			t.Fatalf("clamped page size should have next token")
		}
	})

	t.Run("ProviderToolName and ChoiceID are deterministic and bounded", func(t *testing.T) {
		serverID := spec.MCPServerID(strings.Repeat("server-", 6) + "123")
		toolName := strings.Repeat("tool-name-", 10)

		got1 := ProviderToolName(serverID, toolName)
		got2 := ProviderToolName(serverID, toolName)
		if got1 != got2 {
			t.Fatalf("ProviderToolName not deterministic: %q vs %q", got1, got2)
		}
		if len(got1) > maxProviderToolNameLen {
			t.Fatalf("ProviderToolName too long: %d", len(got1))
		}

		parts := strings.Split(got1, "__")
		if len(parts) != 4 {
			t.Fatalf("ProviderToolName format invalid: %q", got1)
		}
		if len(parts[1]) > maxServerPartLen {
			t.Fatalf("server portion too long: %q", parts[1])
		}
		if len(parts[2]) < minToolPartLen {
			t.Fatalf("tool portion too short: %q", parts[2])
		}

		choice1 := ChoiceID(serverID, toolName)
		choice2 := ChoiceID(serverID, toolName)
		if choice1 != choice2 {
			t.Fatalf("ChoiceID not deterministic: %q vs %q", choice1, choice2)
		}
		if len(choice1) != len("mcp-")+16 {
			t.Fatalf("ChoiceID length = %d, want %d", len(choice1), len("mcp-")+16)
		}

		if choice1 == ChoiceID(serverID, toolName+"x") {
			t.Fatalf("ChoiceID should differ for different inputs")
		}
	})
}
