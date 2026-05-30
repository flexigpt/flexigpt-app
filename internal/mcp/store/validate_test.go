package store

import (
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestValidateServerConfigBranches(t *testing.T) {
	baseHTTP := func() spec.MCPServerConfig {
		return spec.MCPServerConfig{
			SchemaVersion: spec.MCPSchemaVersion,
			ID:            "server-a",
			DisplayName:   "Server A",
			Enabled:       true,
			Transport:     spec.MCPTransportStreamableHTTP,
			StreamableHTTP: &spec.MCPStreamableHTTPConfig{
				URL:      "https://example.test/mcp",
				AuthMode: spec.MCPHTTPAuthNone,
			},
			DefaultPolicy: spec.DefaultMCPServerPolicy(),
			CreatedAt:     time.Now().UTC().Add(-time.Minute),
			ModifiedAt:    time.Now().UTC(),
		}
	}

	baseStdio := func() spec.MCPServerConfig {
		return spec.MCPServerConfig{
			SchemaVersion: spec.MCPSchemaVersion,
			ID:            "server-b",
			DisplayName:   "Server B",
			Enabled:       true,
			Transport:     spec.MCPTransportStdio,
			Stdio: &spec.MCPStdioConfig{
				Command: "server-binary",
			},
			DefaultPolicy: spec.DefaultMCPServerPolicy(),
			CreatedAt:     time.Now().UTC().Add(-time.Minute),
			ModifiedAt:    time.Now().UTC(),
		}
	}

	tests := []struct {
		name             string
		cfg              spec.MCPServerConfig
		wantErrContains  string
		wantAvailability spec.MCPServerAvailability
		wantTrustLevel   spec.MCPTrustLevel
	}{
		{
			name:            "nil config",
			cfg:             spec.MCPServerConfig{},
			wantErrContains: "schemaVersion",
		},
		{
			name:            "schema version mismatch",
			cfg:             func() spec.MCPServerConfig { c := baseHTTP(); c.SchemaVersion = "bad"; return c }(),
			wantErrContains: "schemaVersion",
		},
		{
			name:            "blank id",
			cfg:             func() spec.MCPServerConfig { c := baseHTTP(); c.ID = " "; return c }(),
			wantErrContains: "id must match",
		},
		{
			name:            "blank display name",
			cfg:             func() spec.MCPServerConfig { c := baseHTTP(); c.DisplayName = " "; return c }(),
			wantErrContains: "displayName is empty",
		},
		{
			name:            "whitespace display name",
			cfg:             func() spec.MCPServerConfig { c := baseHTTP(); c.DisplayName = "  Server A  "; return c }(),
			wantErrContains: "leading/trailing whitespace",
		},
		{
			name: "too long display name",
			cfg: func() spec.MCPServerConfig {
				c := baseHTTP()
				c.DisplayName = strings.Repeat("x", maxMCPDisplayNameLen+1)
				return c
			}(),
			wantErrContains: "displayName too long",
		},
		{
			name: "zero timestamps",
			cfg: func() spec.MCPServerConfig {
				c := baseHTTP()
				c.CreatedAt = time.Time{}
				c.ModifiedAt = time.Time{}
				return c
			}(),
			wantErrContains: "createdAt/modifiedAt is zero",
		},
		{
			name: "modified before created",
			cfg: func() spec.MCPServerConfig {
				c := baseHTTP()
				c.CreatedAt = time.Now().UTC()
				c.ModifiedAt = c.CreatedAt.Add(-time.Minute)
				return c
			}(),
			wantErrContains: "modifiedAt is before createdAt",
		},
		{
			name: "soft deleted cannot be enabled",
			cfg: func() spec.MCPServerConfig {
				c := baseHTTP()
				now := time.Now().UTC()
				c.SoftDeletedAt = &now
				c.Enabled = true
				return c
			}(),
			wantErrContains: "soft-deleted server cannot be enabled",
		},
		{
			name:            "invalid availability",
			cfg:             func() spec.MCPServerConfig { c := baseHTTP(); c.Availability = "bogus"; return c }(),
			wantErrContains: "invalid availability",
		},
		{
			name:            "invalid trust level",
			cfg:             func() spec.MCPServerConfig { c := baseHTTP(); c.TrustLevel = "bogus"; return c }(),
			wantErrContains: "invalid trustLevel",
		},
		{
			name: "invalid policy",
			cfg: func() spec.MCPServerConfig {
				c := baseHTTP()
				c.DefaultPolicy.DefaultApprovalRule = "bogus"
				return c
			}(),
			wantErrContains: "defaultPolicy",
		},
		{
			name: "invalid tool policy key mismatch",
			cfg: func() spec.MCPServerConfig {
				c := baseHTTP()
				c.ToolPolicies = map[string]spec.MCPToolPolicyOverride{
					"alpha": {
						ToolName: "beta",
					},
				}
				return c
			}(),
			wantErrContains: "toolPolicies key",
		},
		{
			name: "valid http config defaults availability and trust",
			cfg: func() spec.MCPServerConfig {
				c := baseHTTP()
				c.Availability = ""
				c.TrustLevel = ""
				return c
			}(),
			wantAvailability: spec.MCPServerAvailabilityManual,
			wantTrustLevel:   spec.MCPTrustLevelUntrusted,
		},
		{
			name: "valid stdio config defaults availability and trust",
			cfg: func() spec.MCPServerConfig {
				c := baseStdio()
				c.Availability = ""
				c.TrustLevel = ""
				return c
			}(),
			wantAvailability: spec.MCPServerAvailabilityManual,
			wantTrustLevel:   spec.MCPTrustLevelUntrusted,
		},
		{
			name: "valid http config trusted/autoAttach",
			cfg: func() spec.MCPServerConfig {
				c := baseHTTP()
				c.Availability = spec.MCPServerAvailabilityAutoAttach
				c.TrustLevel = spec.MCPTrustLevelTrusted
				return c
			}(),
			wantAvailability: spec.MCPServerAvailabilityAutoAttach,
			wantTrustLevel:   spec.MCPTrustLevelTrusted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
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
			if tt.wantAvailability != "" && cfg.Availability != tt.wantAvailability {
				t.Fatalf("Availability = %q, want %q", cfg.Availability, tt.wantAvailability)
			}
			if tt.wantTrustLevel != "" && cfg.TrustLevel != tt.wantTrustLevel {
				t.Fatalf("TrustLevel = %q, want %q", cfg.TrustLevel, tt.wantTrustLevel)
			}
		})
	}
}

func TestValidateStdioAndHTTPHelpers(t *testing.T) {
	t.Run("stdio validation branches", func(t *testing.T) {
		serverID := spec.MCPServerID("server-stdio")
		validRef, err := secret.NewMCPSecretRefString(serverID, spec.MCPSecretKindStdioEnv, "TOKEN")
		if err != nil {
			t.Fatalf("NewMCPSecretRefString: %v", err)
		}

		tests := []struct {
			name            string
			cfg             *spec.MCPStdioConfig
			wantErrContains string
		}{
			{
				name: "empty command",
				cfg: &spec.MCPStdioConfig{
					Command: "",
				},
				wantErrContains: "stdio.command is empty",
			},
			{
				name: "shell command",
				cfg: &spec.MCPStdioConfig{
					Command: commandBash,
				},
				wantErrContains: "must execute the server directly",
			},
			{
				name: "invalid env key",
				cfg: &spec.MCPStdioConfig{
					Command: "server-binary",
					Env: map[string]string{
						" BAD": "x",
					},
				},
				wantErrContains: "stdio.env",
			},
			{
				name: "invalid secret ref",
				cfg: &spec.MCPStdioConfig{
					Command: "server-binary",
					SecretEnvRefs: map[string]string{
						"TOKEN": "not-a-ref",
					},
				},
				wantErrContains: "stdio.secretEnvRefs",
			},
			{
				name: "env overlap",
				cfg: &spec.MCPStdioConfig{
					Command: "server-binary",
					Env: map[string]string{
						"token": "x",
					},
					SecretEnvRefs: map[string]string{
						"TOKEN": validRef,
					},
				},
				wantErrContains: "both define",
			},
			{
				name: "valid config",
				cfg: &spec.MCPStdioConfig{
					Command: "server-binary",
					Env: map[string]string{
						"PLAIN": "x",
					},
					SecretEnvRefs: map[string]string{
						"TOKEN": validRef,
					},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validateStdioConfig(serverID, tt.cfg)
				if tt.wantErrContains != "" {
					if err == nil {
						t.Fatalf("validateStdioConfig succeeded, want error containing %q", tt.wantErrContains)
					}
					if !strings.Contains(err.Error(), tt.wantErrContains) {
						t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
					}
					return
				}
				if err != nil {
					t.Fatalf("validateStdioConfig: %v", err)
				}
			})
		}
	})

	t.Run("HTTP validation branches", func(t *testing.T) {
		serverID := spec.MCPServerID("server-http")
		clientRef, err := secret.NewMCPSecretRefString(
			serverID,
			spec.MCPSecretKindOAuthClientCredentials,
			"clientCredentials",
		)
		if err != nil {
			t.Fatalf("NewMCPSecretRefString: %v", err)
		}

		tests := []struct {
			name            string
			cfg             *spec.MCPStreamableHTTPConfig
			wantErrContains string
		}{
			{
				name: "empty url",
				cfg: &spec.MCPStreamableHTTPConfig{
					URL:      "",
					AuthMode: spec.MCPHTTPAuthNone,
				},
				wantErrContains: "streamableHttp.url is empty",
			},
			{
				name: "userinfo",
				cfg: &spec.MCPStreamableHTTPConfig{
					URL:      "https://user@example.test/mcp",
					AuthMode: spec.MCPHTTPAuthNone,
				},
				wantErrContains: "must not contain user info",
			},
			{
				name: "fragment",
				cfg: &spec.MCPStreamableHTTPConfig{
					URL:      "https://example.test/mcp#frag",
					AuthMode: spec.MCPHTTPAuthNone,
				},
				wantErrContains: "must not contain a fragment",
			},
			{
				name: "http non-loopback",
				cfg: &spec.MCPStreamableHTTPConfig{
					URL:      "http://example.test/mcp",
					AuthMode: spec.MCPHTTPAuthNone,
				},
				wantErrContains: "only allowed for loopback hosts",
			},
			{
				name: "invalid auth mode",
				cfg: &spec.MCPStreamableHTTPConfig{
					URL:      "https://example.test/mcp",
					AuthMode: spec.MCPHTTPAuthMode("bogus"),
				},
				wantErrContains: "invalid streamableHttp.authMode",
			},
			{
				name: "clientCredentials missing ref",
				cfg: &spec.MCPStreamableHTTPConfig{
					URL:      "https://example.test/mcp",
					AuthMode: spec.MCPHTTPAuthClientCredentials,
				},
				wantErrContains: "clientCredentialRef is required",
			},
			{
				name: "clientCredentials with metadata document disallowed",
				cfg: &spec.MCPStreamableHTTPConfig{
					URL:                         "https://example.test/mcp",
					AuthMode:                    spec.MCPHTTPAuthClientCredentials,
					ClientCredentialRef:         clientRef,
					ClientIDMetadataDocumentURL: "https://client.example.com/flexigpt.json",
				},
				wantErrContains: "only allowed for oauth authMode",
			},
			{
				name: "oauth with client ref and metadata document",
				cfg: &spec.MCPStreamableHTTPConfig{
					URL:                         "https://example.test/mcp",
					AuthMode:                    spec.MCPHTTPAuthOAuth,
					ClientCredentialRef:         clientRef,
					ClientIDMetadataDocumentURL: "https://client.example.com/flexigpt.json",
				},
			},
			{
				name: "valid none auth",
				cfg: &spec.MCPStreamableHTTPConfig{
					URL:      "https://example.test/mcp",
					AuthMode: spec.MCPHTTPAuthNone,
				},
			},
			{
				name: "valid loopback http",
				cfg: &spec.MCPStreamableHTTPConfig{
					URL:      "http://127.0.0.1:8080/mcp",
					AuthMode: spec.MCPHTTPAuthNone,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validateHTTPConfig(serverID, tt.cfg)
				if tt.wantErrContains != "" {
					if err == nil {
						t.Fatalf("validateHTTPConfig succeeded, want error containing %q", tt.wantErrContains)
					}
					if !strings.Contains(err.Error(), tt.wantErrContains) {
						t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
					}
					return
				}
				if err != nil {
					t.Fatalf("validateHTTPConfig: %v", err)
				}
			})
		}
	})

	t.Run("policy and env helpers", func(t *testing.T) {
		if err := validatePolicy(spec.MCPServerPolicy{
			DefaultApprovalRule:  spec.MCPApprovalRuleAsk,
			DefaultExecutionMode: spec.MCPExecutionModeManual,
		}); err != nil {
			t.Fatalf("validatePolicy(valid): %v", err)
		}

		tests := []struct {
			name            string
			fn              func() error
			wantErrContains string
		}{
			{
				name: "invalid approval rule",
				fn: func() error {
					return validatePolicy(spec.MCPServerPolicy{
						DefaultApprovalRule:  "bogus",
						DefaultExecutionMode: spec.MCPExecutionModeManual,
					})
				},
				wantErrContains: "invalid defaultApprovalRule",
			},
			{
				name: "invalid execution mode",
				fn: func() error {
					return validatePolicy(spec.MCPServerPolicy{
						DefaultApprovalRule:  spec.MCPApprovalRuleAsk,
						DefaultExecutionMode: "bogus",
					})
				},
				wantErrContains: "invalid defaultExecutionMode",
			},
			{
				name: "invalid tool policy name",
				fn: func() error {
					return validateToolPolicyOverride(spec.MCPToolPolicyOverride{})
				},
				wantErrContains: "toolName is empty",
			},
			{
				name: "invalid tool policy approval",
				fn: func() error {
					bad := spec.MCPApprovalRule("bogus")
					return validateToolPolicyOverride(spec.MCPToolPolicyOverride{
						ToolName:     "tool",
						ApprovalRule: &bad,
					})
				},
				wantErrContains: "invalid approvalRule",
			},
			{
				name: "invalid tool policy execution",
				fn: func() error {
					bad := spec.MCPExecutionMode("bogus")
					return validateToolPolicyOverride(spec.MCPToolPolicyOverride{
						ToolName:      "tool",
						ExecutionMode: &bad,
					})
				},
				wantErrContains: "invalid executionMode",
			},
			{
				name: "env key overlap",
				fn: func() error {
					return validateNoEnvKeyOverlap(
						map[string]string{"token": "x"},
						map[string]string{"TOKEN": "ref"},
					)
				},
				wantErrContains: "both define",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.fn()
				if err == nil {
					t.Fatalf("expected error containing %q", tt.wantErrContains)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
				}
			})
		}
	})

	t.Run("loopback helper", func(t *testing.T) {
		if !isLoopbackHost("localhost") {
			t.Fatalf("localhost should be loopback")
		}
		if !isLoopbackHost("127.0.0.1") {
			t.Fatalf("127.0.0.1 should be loopback")
		}
		if isLoopbackHost("example.test") {
			t.Fatalf("example.test should not be loopback")
		}
	})
}
