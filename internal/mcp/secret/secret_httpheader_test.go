package secret

import (
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type testError struct{ msg string }

func TestHTTPHeaderSecretRefRoundTrip(t *testing.T) {
	bundleID := bundleitemutils.BundleID("bundle-a")
	serverID := spec.MCPServerID("server-a")

	raw, err := NewMCPSecretRefString(bundleID, serverID, spec.MCPSecretKindHTTPHeader, " X-API-Key ")
	if err != nil {
		t.Fatalf("NewMCPSecretRefString: %v", err)
	}

	ref, err := ParseMCPSecretRef(raw)
	if err != nil {
		t.Fatalf("ParseMCPSecretRef: %v", err)
	}
	if ref.BundleID != bundleID {
		t.Fatalf("BundleID = %q, want %q", ref.BundleID, bundleID)
	}
	if ref.ServerID != serverID {
		t.Fatalf("ServerID = %q, want %q", ref.ServerID, serverID)
	}
	if ref.Kind != spec.MCPSecretKindHTTPHeader {
		t.Fatalf("Kind = %q, want %q", ref.Kind, spec.MCPSecretKindHTTPHeader)
	}
	if ref.Slot != "x-api-key" {
		t.Fatalf("Slot = %q, want %q", ref.Slot, "x-api-key")
	}

	if err := ValidateMCPSecretRef(raw, bundleID, serverID, spec.MCPSecretKindHTTPHeader, "x-api-key"); err != nil {
		t.Fatalf("ValidateMCPSecretRef: %v", err)
	}
	if got := GetMCPSecretRefStorageKey(ref); got == "" {
		t.Fatalf("GetMCPSecretRefStorageKey returned empty")
	}
	if got := GetMCPSecretRefString(ref); got != raw {
		t.Fatalf("GetMCPSecretRefString = %q, want %q", got, raw)
	}
}

func TestHTTPHeaderSecretRefValidationBranches(t *testing.T) {
	bundleID := bundleitemutils.BundleID("bundle-a")
	serverID := spec.MCPServerID("server-a")

	tests := []struct {
		name            string
		fn              func() error
		wantErrContains string
	}{
		{
			name: "normalizeAndValidateSecretSlot accepts header names",
			fn: func() error {
				slot, err := normalizeAndValidateSecretSlot(spec.MCPSecretKindHTTPHeader, " X-API-Key ")
				if err != nil {
					return err
				}
				if slot != "x-api-key" {
					return &testError{msg: "slot = " + slot}
				}
				return nil
			},
		},
		{
			name: "validateHTTPHeaderSecretSlot empty",
			fn: func() error {
				return validateHTTPHeaderSecretSlot("   ")
			},
			wantErrContains: "header name is empty",
		},
		{
			name: "validateHTTPHeaderSecretSlot leading whitespace",
			fn: func() error {
				return validateHTTPHeaderSecretSlot(" Authorization")
			},
			wantErrContains: "leading/trailing whitespace",
		},
		{
			name: "validateHTTPHeaderSecretSlot invalid character",
			fn: func() error {
				return validateHTTPHeaderSecretSlot("X API Key")
			},
			wantErrContains: "invalid character",
		},
		{
			name: "NewMCPSecretRef rejects invalid header slot",
			fn: func() error {
				_, err := NewMCPSecretRef(bundleID, serverID, spec.MCPSecretKindHTTPHeader, "Bad Header")
				return err
			},
			wantErrContains: "invalid character",
		},
		{
			name: "ValidateMCPSecretRef slot mismatch",
			fn: func() error {
				raw, err := NewMCPSecretRefString(bundleID, serverID, spec.MCPSecretKindHTTPHeader, "X-API-Key")
				if err != nil {
					return err
				}
				return ValidateMCPSecretRef(raw, bundleID, serverID, spec.MCPSecretKindHTTPHeader, "different")
			},
			wantErrContains: "does not match expected slot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if tt.wantErrContains == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q", tt.wantErrContains)
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func (e *testError) Error() string { return e.msg }
