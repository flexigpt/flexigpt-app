package secret

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func ValidateMCPSecretRef(raw string, serverID spec.MCPServerID, kind spec.MCPSecretKind, slot string) error {
	ref, err := ParseMCPSecretRef(raw)
	if err != nil {
		return err
	}
	serverID = spec.MCPServerID(strings.TrimSpace(string(serverID)))
	kind = normalizeSecretKind(kind)
	slot = normalizeSecretSlot(slot)
	if ref.ServerID != serverID {
		return fmt.Errorf("secret ref serverID %q does not match config serverID %q", ref.ServerID, serverID)
	}
	if ref.Kind != kind {
		return fmt.Errorf("secret ref kind %q does not match expected kind %q", ref.Kind, kind)
	}
	if !strings.EqualFold(ref.Slot, slot) {
		return fmt.Errorf("secret ref slot %q does not match expected slot %q", ref.Slot, slot)
	}
	return nil
}

func ParseMCPSecretRef(raw string) (spec.MCPSecretRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return spec.MCPSecretRef{}, errors.New("secret ref is empty")
	}
	if !strings.HasPrefix(raw, spec.SecretRefVersion+":") {
		return spec.MCPSecretRef{}, fmt.Errorf("secret ref %q is not a %s ref", raw, spec.SecretRefVersion)
	}

	encoded := strings.TrimPrefix(raw, spec.SecretRefVersion+":")
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return spec.MCPSecretRef{}, fmt.Errorf("secret ref %q is not valid base64: %w", raw, err)
	}

	var wire struct {
		ServerID spec.MCPServerID   `json:"serverID"`
		Kind     spec.MCPSecretKind `json:"kind"`
		Slot     string             `json:"slot"`
	}
	if err := json.Unmarshal(b, &wire); err != nil {
		return spec.MCPSecretRef{}, fmt.Errorf("secret ref %q is not valid json: %w", raw, err)
	}

	ref := spec.MCPSecretRef{
		ServerID: spec.MCPServerID(strings.TrimSpace(string(wire.ServerID))),
		Kind:     normalizeSecretKind(wire.Kind),
		Slot:     normalizeSecretSlot(wire.Slot),
	}
	if err := validateSecret(ref); err != nil {
		return spec.MCPSecretRef{}, err
	}
	return ref, nil
}

func GetMCPSecretRefStorageKey(r spec.MCPSecretRef) string {
	raw, err := canonicalSecret(r)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return spec.SecretRefVersion + ":" + hex.EncodeToString(sum[:])
}

func GetMCPSecretRefString(r spec.MCPSecretRef) string {
	raw, err := canonicalSecret(r)
	if err != nil {
		return ""
	}
	return spec.SecretRefVersion + ":" + base64.RawURLEncoding.EncodeToString(raw)
}

func canonicalSecret(r spec.MCPSecretRef) ([]byte, error) {
	if err := validateSecret(r); err != nil {
		return nil, err
	}
	wire := struct {
		ServerID spec.MCPServerID   `json:"serverID"`
		Kind     spec.MCPSecretKind `json:"kind"`
		Slot     string             `json:"slot"`
	}{
		ServerID: r.ServerID,
		Kind:     r.Kind,
		Slot:     r.Slot,
	}
	return json.Marshal(wire)
}

func validateSecret(r spec.MCPSecretRef) error {
	if strings.TrimSpace(string(r.ServerID)) == "" {
		return errors.New("secret ref serverID is empty")
	}
	switch r.Kind {
	case spec.MCPSecretKindStdioEnv,
		spec.MCPSecretKindHTTPHeader,
		spec.MCPSecretKindHTTPToken,
		spec.MCPSecretKindOAuthClientSecret,
		spec.MCPSecretKindOAuthClientCredentials:
	default:
		return fmt.Errorf("secret ref kind %q is invalid", r.Kind)
	}
	if strings.TrimSpace(r.Slot) == "" {
		return errors.New("secret ref slot is empty")
	}
	return nil
}

func normalizeSecretKind(kind spec.MCPSecretKind) spec.MCPSecretKind {
	return spec.MCPSecretKind(strings.TrimSpace(string(kind)))
}

func normalizeSecretSlot(slot string) string {
	return strings.ToLower(strings.TrimSpace(slot))
}
