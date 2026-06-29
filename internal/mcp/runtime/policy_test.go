package runtime

import (
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestEvaluationSummaryAndPolicyBranches(t *testing.T) {
	bundleID := bundleitemutils.BundleID("bundle-a")

	t.Run("summary canonicalizes request args", func(t *testing.T) {
		in := EvaluationInput{
			Server: spec.MCPServerConfig{BundleID: bundleID, ID: "server", DisplayName: "Server"},
			Tool: spec.MCPToolCapability{
				BundleID:     bundleID,
				ServerID:     "server",
				ToolName:     "tool",
				Digest:       "digest",
				Enabled:      true,
				InferredRisk: spec.MCPToolRiskRead,
			},
			Req: spec.InvokeMCPToolRequestBody{
				ToolName:  "tool",
				Arguments: map[string]any{"b": 2, "a": 1},
			},
		}

		sum := summary(in)
		if sum == nil {
			t.Fatalf("summary is nil")
		}
		if sum.Arguments != `{"a":1,"b":2}` {
			t.Fatalf("Arguments = %q, want canonical JSON", sum.Arguments)
		}
		if sum.BundleID != bundleID || sum.ServerID != "server" || sum.ToolName != "tool" {
			t.Fatalf("summary = %#v", sum)
		}
	})

	t.Run("summary uses empty object for nil args", func(t *testing.T) {
		in := EvaluationInput{
			Server: spec.MCPServerConfig{BundleID: bundleID, ID: "server"},
			Tool: spec.MCPToolCapability{
				BundleID:     bundleID,
				ServerID:     "server",
				ToolName:     "tool",
				Digest:       "digest",
				Enabled:      true,
				InferredRisk: spec.MCPToolRiskUnknown,
			},
			Req: spec.InvokeMCPToolRequestBody{
				ToolName: "tool",
			},
		}

		sum := summary(in)
		if sum.Arguments != "{}" {
			t.Fatalf("Arguments = %q, want {}", sum.Arguments)
		}
	})

	tests := []struct {
		name         string
		server       spec.MCPServerConfig
		tool         spec.MCPToolCapability
		wantDecision spec.MCPApprovalDecision
		wantReason   string
	}{
		{
			name: "write risk requires approval",
			server: spec.MCPServerConfig{
				BundleID: bundleID,
				ID:       "server",
				DefaultPolicy: spec.MCPServerPolicy{
					DefaultApprovalRule:     spec.MCPApprovalRuleAllow,
					DefaultExecutionMode:    spec.MCPExecutionModeManual,
					RequireApprovalForWrite: true,
				},
			},
			tool: spec.MCPToolCapability{
				BundleID:     bundleID,
				ServerID:     "server",
				ToolName:     "write-tool",
				Digest:       "digest",
				Enabled:      true,
				InferredRisk: spec.MCPToolRiskWrite,
			},
			wantDecision: spec.MCPApprovalDecisionApprovalRequired,
			wantReason:   "write-risk tool requires approval",
		},
		{
			name: "destructive risk requires approval",
			server: spec.MCPServerConfig{
				BundleID: bundleID,
				ID:       "server",
				DefaultPolicy: spec.MCPServerPolicy{
					DefaultApprovalRule:           spec.MCPApprovalRuleAllow,
					DefaultExecutionMode:          spec.MCPExecutionModeManual,
					RequireApprovalForDestructive: true,
				},
			},
			tool: spec.MCPToolCapability{
				BundleID:     bundleID,
				ServerID:     "server",
				ToolName:     "delete-tool",
				Digest:       "digest",
				Enabled:      true,
				InferredRisk: spec.MCPToolRiskDestructive,
			},
			wantDecision: spec.MCPApprovalDecisionApprovalRequired,
			wantReason:   "destructive-risk tool requires approval",
		},
		{
			name: "tool policy deny wins",
			server: spec.MCPServerConfig{
				BundleID: bundleID,
				ID:       "server",
				DefaultPolicy: spec.MCPServerPolicy{
					DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
					DefaultExecutionMode: spec.MCPExecutionModeManual,
				},
				ToolPolicies: map[string]spec.MCPToolPolicyOverride{
					"tool": {
						ToolName: "tool",
						ApprovalRule: func() *spec.MCPApprovalRule {
							v := spec.MCPApprovalRuleDeny
							return &v
						}(),
					},
				},
			},
			tool: spec.MCPToolCapability{
				BundleID:     bundleID,
				ServerID:     "server",
				ToolName:     "tool",
				Digest:       "digest",
				Enabled:      true,
				InferredRisk: spec.MCPToolRiskRead,
			},
			wantDecision: spec.MCPApprovalDecisionDenied,
			wantReason:   toolPolicyDeniesReason,
		},
		{
			name: "allow stale digest avoids deny",
			server: spec.MCPServerConfig{
				BundleID: bundleID,
				ID:       "server",
				DefaultPolicy: spec.MCPServerPolicy{
					DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
					DefaultExecutionMode: spec.MCPExecutionModeManual,
				},
				ToolPolicies: map[string]spec.MCPToolPolicyOverride{
					"tool": {
						ToolName:         "tool",
						ExpectedDigest:   "other-digest",
						AllowStaleDigest: true,
					},
				},
			},
			tool: spec.MCPToolCapability{
				BundleID:     bundleID,
				ServerID:     "server",
				ToolName:     "tool",
				Digest:       "current-digest",
				Enabled:      true,
				InferredRisk: spec.MCPToolRiskRead,
			},
			wantDecision: spec.MCPApprovalDecisionAllowed,
			wantReason:   policyAllowedReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(EvaluationInput{
				Server: tt.server,
				Tool:   tt.tool,
				Req: spec.InvokeMCPToolRequestBody{
					Source:   spec.MCPInvocationSourceUser,
					ToolName: tt.tool.ToolName,
				},
			})

			if got.Decision != tt.wantDecision {
				t.Fatalf("Decision = %q, want %q", got.Decision, tt.wantDecision)
			}
			if got.Reason != tt.wantReason {
				t.Fatalf("Reason = %q, want %q", got.Reason, tt.wantReason)
			}
			if got.Summary == nil {
				t.Fatalf("Summary is nil")
			}
		})
	}
}

func TestApplyToolPolicyOverlayUsesCurrentDefaults(t *testing.T) {
	baseTool := spec.MCPToolCapability{
		ToolName:      "tool",
		Digest:        "digest-current",
		ApprovalRule:  spec.MCPApprovalRuleAsk,
		ExecutionMode: spec.MCPExecutionModeManual,
	}

	t.Run("current defaults replace stale snapshot policy", func(t *testing.T) {
		got := applyToolPolicyOverlay(baseTool, spec.MCPServerConfig{
			DefaultPolicy: spec.MCPServerPolicy{
				DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
				DefaultExecutionMode: spec.MCPExecutionModeAuto,
			},
		})

		if got.ApprovalRule != spec.MCPApprovalRuleAllow {
			t.Fatalf("ApprovalRule = %q, want %q", got.ApprovalRule, spec.MCPApprovalRuleAllow)
		}
		if got.ExecutionMode != spec.MCPExecutionModeAuto {
			t.Fatalf("ExecutionMode = %q, want %q", got.ExecutionMode, spec.MCPExecutionModeAuto)
		}
	})

	t.Run("tool override wins over defaults", func(t *testing.T) {
		approvalRule := spec.MCPApprovalRuleAsk
		executionMode := spec.MCPExecutionModeManual

		got := applyToolPolicyOverlay(baseTool, spec.MCPServerConfig{
			DefaultPolicy: spec.MCPServerPolicy{
				DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
				DefaultExecutionMode: spec.MCPExecutionModeAuto,
			},
			ToolPolicies: map[string]spec.MCPToolPolicyOverride{
				"tool": {
					ToolName:      "tool",
					ApprovalRule:  &approvalRule,
					ExecutionMode: &executionMode,
				},
			},
		})

		if got.ApprovalRule != spec.MCPApprovalRuleAsk {
			t.Fatalf("ApprovalRule = %q, want %q", got.ApprovalRule, spec.MCPApprovalRuleAsk)
		}
		if got.ExecutionMode != spec.MCPExecutionModeManual {
			t.Fatalf("ExecutionMode = %q, want %q", got.ExecutionMode, spec.MCPExecutionModeManual)
		}
	})

	t.Run("partial default policy fills empty execution mode", func(t *testing.T) {
		got := applyToolPolicyOverlay(baseTool, spec.MCPServerConfig{
			DefaultPolicy: spec.MCPServerPolicy{
				DefaultApprovalRule: spec.MCPApprovalRuleAllow,
			},
		})

		if got.ApprovalRule != spec.MCPApprovalRuleAllow {
			t.Fatalf("ApprovalRule = %q, want %q", got.ApprovalRule, spec.MCPApprovalRuleAllow)
		}
		if got.ExecutionMode != spec.MCPExecutionModeManual {
			t.Fatalf("ExecutionMode = %q, want default manual", got.ExecutionMode)
		}
	})

	t.Run("allow stale digest does not mark tool stale", func(t *testing.T) {
		got := applyToolPolicyOverlay(baseTool, spec.MCPServerConfig{
			ToolPolicies: map[string]spec.MCPToolPolicyOverride{
				"tool": {
					ToolName:         "tool",
					ExpectedDigest:   "digest-old",
					AllowStaleDigest: true,
				},
			},
		})

		if got.Stale {
			t.Fatalf("Stale = true, want false when allowStaleDigest is set")
		}
	})
}

func TestSummaryMatchesAndExecutionModeBranches(t *testing.T) {
	bundleID := bundleitemutils.BundleID("bundle-a")
	stored := spec.MCPApprovalSummary{
		BundleID:   bundleID,
		ServerID:   "server",
		ToolName:   "tool",
		ToolDigest: "digest-a",
		Risk:       spec.MCPToolRiskWrite,
		Arguments:  spec.JSONRawString(`{"a":1,"b":2}`),
	}
	expected := spec.MCPApprovalSummary{
		BundleID:   bundleID,
		ServerID:   "server",
		ToolName:   "tool",
		ToolDigest: "digest-a",
		Risk:       spec.MCPToolRiskWrite,
		Arguments:  spec.JSONRawString(`{ "b":2, "a":1 }`),
	}

	if !summaryMatches(stored, expected) {
		t.Fatalf("summaryMatches should accept equivalent summaries")
	}

	t.Run("mismatches", func(t *testing.T) {
		cases := []struct {
			name   string
			stored spec.MCPApprovalSummary
			expect spec.MCPApprovalSummary
			want   bool
		}{
			{
				name:   "bundle mismatch",
				stored: stored,
				expect: func() spec.MCPApprovalSummary { x := expected; x.BundleID = "other"; return x }(),
				want:   false,
			},
			{
				name:   "server mismatch",
				stored: stored,
				expect: func() spec.MCPApprovalSummary { x := expected; x.ServerID = "other"; return x }(),
				want:   false,
			},
			{
				name:   "tool mismatch",
				stored: stored,
				expect: func() spec.MCPApprovalSummary { x := expected; x.ToolName = "other"; return x }(),
				want:   false,
			},
			{
				name:   "risk mismatch",
				stored: stored,
				expect: func() spec.MCPApprovalSummary { x := expected; x.Risk = spec.MCPToolRiskRead; return x }(),
				want:   false,
			},
			{
				name:   "digest mismatch",
				stored: stored,
				expect: func() spec.MCPApprovalSummary { x := expected; x.ToolDigest = "other"; return x }(),
				want:   false,
			},
			{
				name: "stored digest only",
				stored: func() spec.MCPApprovalSummary {
					x := stored
					x.ToolDigest = "digest-a"
					return x
				}(),
				expect: func() spec.MCPApprovalSummary {
					x := expected
					x.ToolDigest = ""
					return x
				}(),
				want: true,
			},
			{
				name: "stored args only",
				stored: func() spec.MCPApprovalSummary {
					x := stored
					x.Arguments = `{"a":1}`
					return x
				}(),
				expect: func() spec.MCPApprovalSummary {
					x := expected
					x.Arguments = ""
					return x
				}(),
				want: true,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got := summaryMatches(tc.stored, tc.expect)
				if got != tc.want {
					t.Fatalf("summaryMatches = %v, want %v", got, tc.want)
				}
			})
		}
	})

	t.Run("execution mode defaults and overrides", func(t *testing.T) {
		server := spec.MCPServerConfig{
			BundleID: bundleID,
			ID:       "server",
			DefaultPolicy: spec.MCPServerPolicy{
				DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
				DefaultExecutionMode: "",
			},
			ToolPolicies: map[string]spec.MCPToolPolicyOverride{
				"tool": {
					ToolName: "tool",
					ExecutionMode: func() *spec.MCPExecutionMode {
						v := spec.MCPExecutionModeAuto
						return &v
					}(),
				},
			},
		}

		if got := ExecutionMode(server, spec.MCPToolCapability{ToolName: "other"}); got != spec.MCPExecutionModeManual {
			t.Fatalf("ExecutionMode default = %q, want manual", got)
		}
		if got := ExecutionMode(server, spec.MCPToolCapability{ToolName: "tool"}); got != spec.MCPExecutionModeAuto {
			t.Fatalf("ExecutionMode override = %q, want auto", got)
		}
	})

	t.Run("ensure digest", func(t *testing.T) {
		if err := EnsureDigest("expected", "got", false); err == nil || !strings.Contains(err.Error(), "expected") {
			t.Fatalf("EnsureDigest mismatch = %v, want stale reference error", err)
		}
		if err := EnsureDigest("expected", "got", true); err != nil {
			t.Fatalf("EnsureDigest allowStale: %v", err)
		}
		if err := EnsureDigest("", "got", false); err != nil {
			t.Fatalf("EnsureDigest empty expected: %v", err)
		}
		if err := EnsureDigest("expected", "", false); err != nil {
			t.Fatalf("EnsureDigest empty got: %v", err)
		}
	})
}
