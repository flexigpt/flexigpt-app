package secret

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type secretRefWire struct {
	BundleID bundleitemutils.BundleID `json:"bundleID"`
	ServerID spec.MCPServerID         `json:"serverID"`
	Kind     spec.MCPSecretKind       `json:"kind"`
	Slot     string                   `json:"slot"`
}

func TestSecretRefRoundTripAndCanonicalStorageKey(t *testing.T) {
	tests := []struct {
		name     string
		bundleID bundleitemutils.BundleID
		serverID spec.MCPServerID
		kind     spec.MCPSecretKind
		slot     string
		wantSlot string
		wantJSON string
	}{
		{
			name:     "stdio env",
			bundleID: bundleitemutils.BundleID("  bundle-a  "),
			serverID: spec.MCPServerID("  server-a  "),
			kind:     spec.MCPSecretKindStdioEnv,
			slot:     " TOKEN ",
			wantSlot: "token",
			wantJSON: `{"bundleID":"bundle-a","serverID":"server-a","kind":"stdioEnv","slot":"token"}`,
		},
		{
			name:     "oauth client credentials",
			bundleID: bundleitemutils.BundleID("  bundle-a  "),
			serverID: spec.MCPServerID("  server-b  "),
			kind:     spec.MCPSecretKindOAuthClientCredentials,
			slot:     " clientCredentials ",
			wantSlot: "clientcredentials",
			wantJSON: `{"bundleID":"bundle-a","serverID":"server-b","kind":"oauthClientCredentials","slot":"clientcredentials"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := NewMCPSecretRef(tt.bundleID, tt.serverID, tt.kind, tt.slot)
			if err != nil {
				t.Fatalf("NewMCPSecretRef: %v", err)
			}

			if ref.BundleID != bundleitemutils.BundleID("bundle-a") {
				t.Fatalf("BundleID = %q, want %q", ref.BundleID, bundleitemutils.BundleID("bundle-a"))
			}
			if ref.ServerID != spec.MCPServerID(strings.TrimSpace(string(tt.serverID))) {
				t.Fatalf("ServerID = %q, want %q", ref.ServerID, strings.TrimSpace(string(tt.serverID)))
			}
			if ref.Kind != tt.kind {
				t.Fatalf("Kind = %q, want %q", ref.Kind, tt.kind)
			}
			if ref.Slot != tt.wantSlot {
				t.Fatalf("Slot = %q, want %q", ref.Slot, tt.wantSlot)
			}

			raw, err := canonicalSecret(ref)
			if err != nil {
				t.Fatalf("canonicalSecret: %v", err)
			}
			if got := string(raw); got != tt.wantJSON {
				t.Fatalf("canonicalSecret = %q, want %q", got, tt.wantJSON)
			}

			refString, err := NewMCPSecretRefString(tt.bundleID, tt.serverID, tt.kind, tt.slot)
			if err != nil {
				t.Fatalf("NewMCPSecretRefString: %v", err)
			}
			wantString := spec.SecretRefVersion + ":" + base64.RawURLEncoding.EncodeToString(raw)
			if refString != wantString {
				t.Fatalf("NewMCPSecretRefString = %q, want %q", refString, wantString)
			}
			if got := GetMCPSecretRefString(ref); got != refString {
				t.Fatalf("GetMCPSecretRefString = %q, want %q", got, refString)
			}

			parsed, err := ParseMCPSecretRef(refString)
			if err != nil {
				t.Fatalf("ParseMCPSecretRef: %v", err)
			}
			if parsed != ref {
				t.Fatalf("ParseMCPSecretRef = %#v, want %#v", parsed, ref)
			}

			if err := ValidateMCPSecretRef(refString, tt.bundleID, tt.serverID, tt.kind, tt.slot); err != nil {
				t.Fatalf("ValidateMCPSecretRef: %v", err)
			}

			sum := sha256.Sum256(raw)
			wantStorageKey := spec.SecretRefVersion + ":" + hex.EncodeToString(sum[:])
			if got := GetMCPSecretRefStorageKey(ref); got != wantStorageKey {
				t.Fatalf("GetMCPSecretRefStorageKey = %q, want %q", got, wantStorageKey)
			}
		})
	}
}

func TestSecretRefParseAndValidationErrorBranches(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "empty",
			raw:  "",
			want: "secret ref is empty",
		},
		{
			name: "bad prefix",
			raw:  "not-a-secret-ref",
			want: "is not a mcpv1 ref",
		},
		{
			name: "bad base64",
			raw:  spec.SecretRefVersion + ":!!!",
			want: "not valid base64",
		},
		{
			name: "bad json",
			raw:  spec.SecretRefVersion + ":" + base64.RawURLEncoding.EncodeToString([]byte("[]")),
			want: "not valid json",
		},
		{
			name: "empty bundleID",
			raw: encodeSecretWire(t, secretRefWire{
				BundleID: "",
				ServerID: "server-a",
				Kind:     spec.MCPSecretKindStdioEnv,
				Slot:     "TOKEN",
			}),
			want: "bundleID is empty",
		},
		{
			name: "empty serverID",
			raw: encodeSecretWire(t, secretRefWire{
				BundleID: bundleitemutils.BundleID("bundle-a"),
				ServerID: "",
				Kind:     spec.MCPSecretKindStdioEnv,
				Slot:     "TOKEN",
			}),
			want: "serverID is empty",
		},
		{
			name: "invalid kind",
			raw: encodeSecretWire(t, secretRefWire{
				BundleID: bundleitemutils.BundleID("bundle-a"),
				ServerID: "server-a",
				Kind:     spec.MCPSecretKind("bogus"),
				Slot:     "TOKEN",
			}),
			want: "kind",
		},
		{
			name: "stdio env invalid slot",
			raw: encodeSecretWire(t, secretRefWire{
				BundleID: bundleitemutils.BundleID("bundle-a"),
				ServerID: "server-a",
				Kind:     spec.MCPSecretKindStdioEnv,
				Slot:     "TO=KEN",
			}),
			want: "must not contain '='",
		},
		{
			name: "oauth client credentials invalid slot",
			raw: encodeSecretWire(t, secretRefWire{
				BundleID: bundleitemutils.BundleID("bundle-a"),
				ServerID: "server-a",
				Kind:     spec.MCPSecretKindOAuthClientCredentials,
				Slot:     "not-clientCredentials",
			}),
			want: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseMCPSecretRef(tt.raw)
			if err == nil {
				t.Fatalf("ParseMCPSecretRef succeeded, want error containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}

	t.Run("invalid refs render empty strings", func(t *testing.T) {
		if got := GetMCPSecretRefString(spec.MCPSecretRef{}); got != "" {
			t.Fatalf("GetMCPSecretRefString(invalid) = %q, want empty", got)
		}
		if got := GetMCPSecretRefStorageKey(spec.MCPSecretRef{}); got != "" {
			t.Fatalf("GetMCPSecretRefStorageKey(invalid) = %q, want empty", got)
		}
		if _, err := canonicalSecret(spec.MCPSecretRef{}); err == nil {
			t.Fatalf("canonicalSecret(invalid) succeeded, want error")
		}
	})
}

func TestSecretRefNormalizationAndValidationHelpers(t *testing.T) {
	t.Run("normalize helpers", func(t *testing.T) {
		if got := normalizeSecretKind(spec.MCPSecretKind("  stdioEnv  ")); got != spec.MCPSecretKindStdioEnv {
			t.Fatalf("normalizeSecretKind = %q, want %q", got, spec.MCPSecretKindStdioEnv)
		}
		if got := normalizeSecretSlot("  MiXeD  "); got != "mixed" {
			t.Fatalf("normalizeSecretSlot = %q, want %q", got, "mixed")
		}
	})

	t.Run("normalizeAndValidateSecretSlot", func(t *testing.T) {
		slot, err := normalizeAndValidateSecretSlot(spec.MCPSecretKindStdioEnv, " TOKEN_1 ")
		if err != nil {
			t.Fatalf("normalizeAndValidateSecretSlot(stdio): %v", err)
		}
		if slot != "token_1" {
			t.Fatalf("normalizeAndValidateSecretSlot(stdio) = %q, want %q", slot, "token_1")
		}

		if _, err := normalizeAndValidateSecretSlot(spec.MCPSecretKindStdioEnv, "TO=KEN"); err == nil ||
			!strings.Contains(err.Error(), "must not contain '='") {
			t.Fatalf("normalizeAndValidateSecretSlot(stdio invalid) err = %v, want env key error", err)
		}

		slot, err = normalizeAndValidateSecretSlot(spec.MCPSecretKindOAuthClientCredentials, " clientCredentials ")
		if err != nil {
			t.Fatalf("normalizeAndValidateSecretSlot(oauth): %v", err)
		}
		if slot != "clientcredentials" {
			t.Fatalf("normalizeAndValidateSecretSlot(oauth) = %q, want %q", slot, "clientcredentials")
		}

		if _, err := normalizeAndValidateSecretSlot(spec.MCPSecretKindOAuthClientCredentials, "wrong"); err == nil ||
			!strings.Contains(err.Error(), "expected clientCredentials") {
			t.Fatalf("normalizeAndValidateSecretSlot(oauth invalid) err = %v, want clientCredentials error", err)
		}
	})

	t.Run("validateEnvSecretSlot", func(t *testing.T) {
		if err := validateEnvSecretSlot("TOKEN_1"); err != nil {
			t.Fatalf("validateEnvSecretSlot(valid): %v", err)
		}
		if err := validateEnvSecretSlot("TO\tKEN"); err == nil ||
			!strings.Contains(err.Error(), "control character") {
			t.Fatalf("validateEnvSecretSlot(control char) err = %v, want control character error", err)
		}
	})
}

func TestMCPSecretRefRoundTripAndStorageKeys(t *testing.T) {
	bundleID := bundleitemutils.BundleID("bundle-a")
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
			ref, err := NewMCPSecretRef(bundleID, tt.serverID, tt.kind, tt.slot)
			if err != nil {
				t.Fatalf("NewMCPSecretRef: %v", err)
			}
			if ref.BundleID != bundleID {
				t.Fatalf("BundleID = %q, want %q", ref.BundleID, bundleID)
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

			raw, err := NewMCPSecretRefString(bundleID, tt.serverID, tt.kind, tt.slot)
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
			if parsed.BundleID != bundleID {
				t.Fatalf("parsed.BundleID = %q, want %q", parsed.BundleID, bundleID)
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

			if err := ValidateMCPSecretRef(raw, bundleID, tt.serverID, tt.kind, tt.slot); err != nil {
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
	bundleID := bundleitemutils.BundleID("bundle-a")
	tests := []struct {
		name            string
		fn              func() error
		wantErrContains string
	}{
		{
			name:            "empty bundleID",
			fn:              func() error { _, err := NewMCPSecretRef("", "server", spec.MCPSecretKindStdioEnv, "TOKEN"); return err },
			wantErrContains: "bundleID is empty",
		},
		{
			name:            "empty serverID",
			fn:              func() error { _, err := NewMCPSecretRef(bundleID, "", spec.MCPSecretKindStdioEnv, "TOKEN"); return err },
			wantErrContains: "serverID is empty",
		},
		{name: "invalid kind", fn: func() error {
			_, err := NewMCPSecretRef(bundleID, "server", spec.MCPSecretKind("bogus"), "TOKEN")
			return err
		}, wantErrContains: "kind"},
		{name: "stdio env slot invalid", fn: func() error {
			_, err := NewMCPSecretRef(bundleID, "server", spec.MCPSecretKindStdioEnv, "bad=slot")
			return err
		}, wantErrContains: "env key must not contain"},
		{name: "oauth client credentials slot invalid", fn: func() error {
			_, err := NewMCPSecretRef(
				bundleID,
				"server",
				spec.MCPSecretKindOAuthClientCredentials,
				"not-clientCredentials",
			)
			return err
		}, wantErrContains: "expected clientCredentials"},
		{
			name:            "parse invalid prefix",
			fn:              func() error { _, err := ParseMCPSecretRef("not-a-secret-ref"); return err },
			wantErrContains: "is not a mcpv1 ref",
		},
		{name: "parse invalid json", fn: func() error {
			raw := "mcpv1:" + base64.RawURLEncoding.EncodeToString([]byte("[]"))
			_, err := ParseMCPSecretRef(raw)
			return err
		}, wantErrContains: "not valid json"},
		{name: "validate mismatched bundleID", fn: func() error {
			raw, err := NewMCPSecretRefString(bundleID, "server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
			if err != nil {
				return err
			}
			return ValidateMCPSecretRef(raw, "bundle-b", "server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
		}, wantErrContains: "does not match config bundleID"},
		{name: "validate mismatched serverID", fn: func() error {
			raw, err := NewMCPSecretRefString(bundleID, "server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
			if err != nil {
				return err
			}
			return ValidateMCPSecretRef(raw, bundleID, "server-b", spec.MCPSecretKindStdioEnv, "TOKEN")
		}, wantErrContains: "does not match config serverID"},
		{name: "validate mismatched kind", fn: func() error {
			raw, err := NewMCPSecretRefString(bundleID, "server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
			if err != nil {
				return err
			}
			return ValidateMCPSecretRef(
				raw,
				bundleID,
				"server-a",
				spec.MCPSecretKindOAuthClientCredentials,
				"clientCredentials",
			)
		}, wantErrContains: "does not match expected kind"},
		{name: "validate mismatched slot", fn: func() error {
			raw, err := NewMCPSecretRefString(bundleID, "server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
			if err != nil {
				return err
			}
			return ValidateMCPSecretRef(raw, bundleID, "server-a", spec.MCPSecretKindStdioEnv, "OTHER")
		}, wantErrContains: "does not match expected slot"},
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
	bundleID := bundleitemutils.BundleID("bundle-a")
	ref, err := NewMCPSecretRef(bundleID, "  server-a  ", spec.MCPSecretKindStdioEnv, "  TOKEN  ")
	if err != nil {
		t.Fatalf("NewMCPSecretRef: %v", err)
	}

	if ref.BundleID != bundleID {
		t.Fatalf("BundleID = %q, want %q", ref.BundleID, bundleID)
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

	raw, err := NewMCPSecretRefString(bundleID, "server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
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
	if parsed.BundleID != bundleID {
		t.Fatalf("parsed.BundleID = %q, want %q", parsed.BundleID, bundleID)
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

	if err := ValidateMCPSecretRef(raw, bundleID, "server-a", spec.MCPSecretKindStdioEnv, "TOKEN"); err != nil {
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
		{name: "valid uppercase env key", slot: "TOKEN"},
		{name: "leading whitespace", slot: " TOKEN", wantErrContains: "leading/trailing whitespace"},
		{name: "trailing whitespace", slot: "TOKEN ", wantErrContains: "leading/trailing whitespace"},
		{name: "equals sign", slot: "TO=KEN", wantErrContains: "must not contain '='"},
		{name: "nul", slot: "TO\x00KEN", wantErrContains: "must not contain '=' or NUL"},
		{name: "control char", slot: "TO\tKEN", wantErrContains: "control character"},
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
	bundleID := bundleitemutils.BundleID("bundle-a")
	tests := []struct {
		name            string
		fn              func() error
		wantErrContains string
	}{
		{
			name:            "empty bundleID",
			fn:              func() error { _, err := NewMCPSecretRef("", "server", spec.MCPSecretKindStdioEnv, "TOKEN"); return err },
			wantErrContains: "bundleID is empty",
		},
		{name: "invalid kind", fn: func() error {
			_, err := NewMCPSecretRef(bundleID, "server", spec.MCPSecretKind("bogus"), "TOKEN")
			return err
		}, wantErrContains: "kind"},
		{name: "oauth client credentials slot invalid", fn: func() error {
			_, err := NewMCPSecretRef(
				bundleID,
				"server",
				spec.MCPSecretKindOAuthClientCredentials,
				"not-clientCredentials",
			)
			return err
		}, wantErrContains: "expected clientCredentials"},
		{
			name:            "parse invalid prefix",
			fn:              func() error { _, err := ParseMCPSecretRef("not-a-secret-ref"); return err },
			wantErrContains: "is not a mcpv1 ref",
		},
		{name: "validate mismatched serverID", fn: func() error {
			raw, err := NewMCPSecretRefString(bundleID, "server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
			if err != nil {
				return err
			}
			return ValidateMCPSecretRef(raw, bundleID, "server-b", spec.MCPSecretKindStdioEnv, "TOKEN")
		}, wantErrContains: "does not match config serverID"},
		{name: "validate mismatched kind", fn: func() error {
			raw, err := NewMCPSecretRefString(bundleID, "server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
			if err != nil {
				return err
			}
			return ValidateMCPSecretRef(
				raw,
				bundleID,
				"server-a",
				spec.MCPSecretKindOAuthClientCredentials,
				"clientCredentials",
			)
		}, wantErrContains: "does not match expected kind"},
		{name: "validate mismatched slot", fn: func() error {
			raw, err := NewMCPSecretRefString(bundleID, "server-a", spec.MCPSecretKindStdioEnv, "TOKEN")
			if err != nil {
				return err
			}
			return ValidateMCPSecretRef(raw, bundleID, "server-a", spec.MCPSecretKindStdioEnv, "OTHER")
		}, wantErrContains: "does not match expected slot"},
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

func encodeSecretWire(t *testing.T, wire secretRefWire) string {
	t.Helper()

	raw, err := json.Marshal(wire)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return spec.SecretRefVersion + ":" + base64.RawURLEncoding.EncodeToString(raw)
}
