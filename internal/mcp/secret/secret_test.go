package secret

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestMCPSecretRefRoundTripAndStorageKeys(t *testing.T) {
	tests := []struct {
		name       string
		serverID   spec.MCPServerID
		kind       spec.MCPSecretKind
		slot       string
		wantSlot   string
		wantParsed string
	}{
		{
			name:       "stdio env",
			serverID:   "server-a",
			kind:       spec.MCPSecretKindStdioEnv,
			slot:       "TOKEN",
			wantSlot:   "token",
			wantParsed: "token",
		},
		{
			name:       "oauth client credentials",
			serverID:   "server-b",
			kind:       spec.MCPSecretKindOAuthClientCredentials,
			slot:       "clientCredentials",
			wantSlot:   "clientcredentials",
			wantParsed: "clientcredentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := NewMCPSecretRef(tt.serverID, tt.kind, tt.slot)
			if err != nil {
				t.Fatalf("NewMCPSecretRef: %v", err)
			}
			if ref.ServerID != tt.serverID {
				t.Fatalf("ServerID = %q, want %q", ref.ServerID, tt.serverID)
			}
			if ref.Kind != tt.kind {
				t.Fatalf("Kind = %q, want %q", ref.Kind, tt.kind)
			}
			if ref.Slot != tt.wantSlot {
				t.Fatalf("Slot = %q, want %q", ref.Slot, tt.wantSlot)
			}

			raw, err := NewMCPSecretRefString(tt.serverID, tt.kind, tt.slot)
			if err != nil {
				t.Fatalf("NewMCPSecretRefString: %v", err)
			}
			if !strings.HasPrefix(raw, spec.SecretRefVersion+":") {
				t.Fatalf("raw ref = %q, want %q prefix", raw, spec.SecretRefVersion+":")
			}

			parsed, err := ParseMCPSecretRef(raw)
			if err != nil {
				t.Fatalf("ParseMCPSecretRef: %v", err)
			}
			if parsed.ServerID != tt.serverID {
				t.Fatalf("parsed.ServerID = %q, want %q", parsed.ServerID, tt.serverID)
			}
			if parsed.Kind != tt.kind {
				t.Fatalf("parsed.Kind = %q, want %q", parsed.Kind, tt.kind)
			}
			if parsed.Slot != tt.wantParsed {
				t.Fatalf("parsed.Slot = %q, want %q", parsed.Slot, tt.wantParsed)
			}

			if err := ValidateMCPSecretRef(raw, tt.serverID, tt.kind, tt.slot); err != nil {
				t.Fatalf("ValidateMCPSecretRef: %v", err)
			}

			storageKey := GetMCPSecretRefStorageKey(ref)
			if storageKey == "" {
				t.Fatalf("storageKey is empty")
			}
			if !strings.HasPrefix(storageKey, spec.SecretRefVersion+":") {
				t.Fatalf("storageKey = %q, want %q prefix", storageKey, spec.SecretRefVersion+":")
			}

			encoded := GetMCPSecretRefString(ref)
			if encoded == "" {
				t.Fatalf("GetMCPSecretRefString returned empty")
			}
			if encoded != raw {
				t.Fatalf("GetMCPSecretRefString = %q, want %q", encoded, raw)
			}
		})
	}
}

func TestMCPSecretRefValidationErrors(t *testing.T) {
	tests := []struct {
		name            string
		fn              func() error
		wantErrContains string
	}{
		{
			name: "empty serverID",
			fn: func() error {
				_, err := NewMCPSecretRef("", spec.MCPSecretKindStdioEnv, "TOKEN")
				return err
			},
			wantErrContains: "serverID is empty",
		},
		{
			name: "invalid kind",
			fn: func() error {
				_, err := NewMCPSecretRef("server", spec.MCPSecretKind("bogus"), "TOKEN")
				return err
			},
			wantErrContains: "kind",
		},
		{
			name: "stdio env slot invalid",
			fn: func() error {
				_, err := NewMCPSecretRef("server", spec.MCPSecretKindStdioEnv, "bad=slot")
				return err
			},
			wantErrContains: "env key must not contain",
		},
		{
			name: "oauth client credentials slot invalid",
			fn: func() error {
				_, err := NewMCPSecretRef("server", spec.MCPSecretKindOAuthClientCredentials, "not-clientCredentials")
				return err
			},
			wantErrContains: "expected clientCredentials",
		},
		{
			name: "parse invalid prefix",
			fn: func() error {
				_, err := ParseMCPSecretRef("not-a-secret-ref")
				return err
			},
			wantErrContains: "is not a mcpv1 ref",
		},
		{
			name: "parse invalid json",
			fn: func() error {
				raw := "mcpv1:" + base64.RawURLEncoding.EncodeToString([]byte("[]"))
				_, err := ParseMCPSecretRef(raw)
				return err
			},
			wantErrContains: "not valid json",
		},
		{
			name: "validate mismatched serverID",
			fn: func() error {
				raw, err := NewMCPSecretRefString("server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
				if err != nil {
					return err
				}
				return ValidateMCPSecretRef(raw, "server-b", spec.MCPSecretKindStdioEnv, "TOKEN")
			},
			wantErrContains: "does not match config serverID",
		},
		{
			name: "validate mismatched kind",
			fn: func() error {
				raw, err := NewMCPSecretRefString("server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
				if err != nil {
					return err
				}
				return ValidateMCPSecretRef(
					raw,
					"server-a",
					spec.MCPSecretKindOAuthClientCredentials,
					"clientCredentials",
				)
			},
			wantErrContains: "does not match expected kind",
		},
		{
			name: "validate mismatched slot",
			fn: func() error {
				raw, err := NewMCPSecretRefString("server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
				if err != nil {
					return err
				}
				return ValidateMCPSecretRef(raw, "server-a", spec.MCPSecretKindStdioEnv, "OTHER")
			},
			wantErrContains: "does not match expected slot",
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
}

func TestSecretRefNormalizationAndCanonicalEncoding(t *testing.T) {
	ref, err := NewMCPSecretRef("  server-a  ", spec.MCPSecretKindStdioEnv, "  TOKEN  ")
	if err != nil {
		t.Fatalf("NewMCPSecretRef: %v", err)
	}

	if ref.ServerID != "server-a" {
		t.Fatalf("ServerID = %q, want %q", ref.ServerID, "server-a")
	}
	if ref.Kind != spec.MCPSecretKindStdioEnv {
		t.Fatalf("Kind = %q, want %q", ref.Kind, spec.MCPSecretKindStdioEnv)
	}
	if ref.Slot != "token" {
		t.Fatalf("Slot = %q, want %q", ref.Slot, "token")
	}

	raw, err := NewMCPSecretRefString("server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
	if err != nil {
		t.Fatalf("NewMCPSecretRefString: %v", err)
	}
	if !strings.HasPrefix(raw, spec.SecretRefVersion+":") {
		t.Fatalf("raw ref = %q, want %q prefix", raw, spec.SecretRefVersion+":")
	}

	parsed, err := ParseMCPSecretRef(raw)
	if err != nil {
		t.Fatalf("ParseMCPSecretRef: %v", err)
	}
	if parsed.ServerID != "server-a" {
		t.Fatalf("parsed.ServerID = %q, want %q", parsed.ServerID, "server-a")
	}
	if parsed.Kind != spec.MCPSecretKindStdioEnv {
		t.Fatalf("parsed.Kind = %q, want %q", parsed.Kind, spec.MCPSecretKindStdioEnv)
	}
	if parsed.Slot != "token" {
		t.Fatalf("parsed.Slot = %q, want %q", parsed.Slot, "token")
	}

	if err := ValidateMCPSecretRef(raw, "server-a", spec.MCPSecretKindStdioEnv, "TOKEN"); err != nil {
		t.Fatalf("ValidateMCPSecretRef(case-insensitive slot): %v", err)
	}

	storageKey1 := GetMCPSecretRefStorageKey(ref)
	storageKey2 := GetMCPSecretRefStorageKey(ref)
	if storageKey1 == "" || storageKey1 != storageKey2 {
		t.Fatalf("storage key not deterministic: %q vs %q", storageKey1, storageKey2)
	}

	encoded := GetMCPSecretRefString(ref)
	if encoded != raw {
		t.Fatalf("GetMCPSecretRefString = %q, want %q", encoded, raw)
	}
}

func TestValidateEnvSecretSlotAndKindHelpers(t *testing.T) {
	t.Run("normalize helpers", func(t *testing.T) {
		if got := normalizeSecretKind(spec.MCPSecretKind("  stdioEnv  ")); got != spec.MCPSecretKindStdioEnv {
			t.Fatalf("normalizeSecretKind = %q, want %q", got, spec.MCPSecretKindStdioEnv)
		}
		if got := normalizeSecretSlot("  MiXeD  "); got != "mixed" {
			t.Fatalf("normalizeSecretSlot = %q, want %q", got, "mixed")
		}
	})

	tests := []struct {
		name            string
		slot            string
		wantErrContains string
	}{
		{
			name: "valid uppercase env key",
			slot: "TOKEN",
		},
		{
			name:            "leading whitespace",
			slot:            " TOKEN",
			wantErrContains: "leading/trailing whitespace",
		},
		{
			name:            "trailing whitespace",
			slot:            "TOKEN ",
			wantErrContains: "leading/trailing whitespace",
		},
		{
			name:            "equals sign",
			slot:            "TO=KEN",
			wantErrContains: "must not contain '='",
		},
		{
			name:            "nul",
			slot:            "TO\x00KEN",
			wantErrContains: "must not contain '=' or NUL",
		},
		{
			name:            "control char",
			slot:            "TO\tKEN",
			wantErrContains: "control character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvSecretSlot(tt.slot)
			if tt.wantErrContains == "" {
				if err != nil {
					t.Fatalf("validateEnvSecretSlot(%q): %v", tt.slot, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateEnvSecretSlot(%q) succeeded, want error", tt.slot)
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestSecretRefValidationFailures(t *testing.T) {
	tests := []struct {
		name            string
		fn              func() error
		wantErrContains string
	}{
		{
			name: "empty serverID",
			fn: func() error {
				_, err := NewMCPSecretRef("", spec.MCPSecretKindStdioEnv, "TOKEN")
				return err
			},
			wantErrContains: "serverID is empty",
		},
		{
			name: "invalid kind",
			fn: func() error {
				_, err := NewMCPSecretRef("server", spec.MCPSecretKind("bogus"), "TOKEN")
				return err
			},
			wantErrContains: "kind",
		},

		{
			name: "oauth client credentials slot invalid",
			fn: func() error {
				_, err := NewMCPSecretRef("server", spec.MCPSecretKindOAuthClientCredentials, "not-clientCredentials")
				return err
			},
			wantErrContains: "expected clientCredentials",
		},
		{
			name: "parse invalid prefix",
			fn: func() error {
				_, err := ParseMCPSecretRef("not-a-secret-ref")
				return err
			},
			wantErrContains: "is not a mcpv1 ref",
		},
		{
			name: "validate mismatched serverID",
			fn: func() error {
				raw, err := NewMCPSecretRefString("server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
				if err != nil {
					return err
				}
				return ValidateMCPSecretRef(raw, "server-b", spec.MCPSecretKindStdioEnv, "TOKEN")
			},
			wantErrContains: "does not match config serverID",
		},
		{
			name: "validate mismatched kind",
			fn: func() error {
				raw, err := NewMCPSecretRefString("server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
				if err != nil {
					return err
				}
				return ValidateMCPSecretRef(
					raw,
					"server-a",
					spec.MCPSecretKindOAuthClientCredentials,
					"clientCredentials",
				)
			},
			wantErrContains: "does not match expected kind",
		},
		{
			name: "validate mismatched slot",
			fn: func() error {
				raw, err := NewMCPSecretRefString("server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
				if err != nil {
					return err
				}
				return ValidateMCPSecretRef(raw, "server-a", spec.MCPSecretKindStdioEnv, "OTHER")
			},
			wantErrContains: "does not match expected slot",
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
}
