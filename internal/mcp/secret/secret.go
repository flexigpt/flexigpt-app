package secret

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func NewMCPSecretRef(
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	kind spec.MCPSecretKind,
	slot string,
) (spec.MCPSecretRef, error) {
	serverID = spec.MCPServerID(strings.TrimSpace(string(serverID)))
	kind = normalizeSecretKind(kind)
	bundleID = bundleitemutils.BundleID(strings.TrimSpace(string(bundleID)))
	normalizedSlot, err := normalizeAndValidateSecretSlot(kind, slot)
	if err != nil {
		return spec.MCPSecretRef{}, err
	}

	ref := spec.MCPSecretRef{
		BundleID: bundleID,
		ServerID: serverID,
		Kind:     kind,
		Slot:     normalizedSlot,
	}
	if err := validateSecret(ref); err != nil {
		return spec.MCPSecretRef{}, err
	}
	return ref, nil
}

func NewMCPSecretRefString(
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	kind spec.MCPSecretKind,
	slot string,
) (string, error) {
	ref, err := NewMCPSecretRef(bundleID, serverID, kind, slot)
	if err != nil {
		return "", err
	}
	out := GetMCPSecretRefString(ref)
	if out == "" {
		return "", errors.New("could not encode secret ref")
	}
	return out, nil
}

func ValidateMCPSecretRef(
	raw string,
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	kind spec.MCPSecretKind,
	slot string,
) error {
	ref, err := ParseMCPSecretRef(raw)
	if err != nil {
		return err
	}
	bundleID = bundleitemutils.BundleID(strings.TrimSpace(string(bundleID)))
	serverID = spec.MCPServerID(strings.TrimSpace(string(serverID)))
	kind = normalizeSecretKind(kind)
	slot = normalizeSecretSlot(slot)
	if ref.BundleID != bundleID {
		return fmt.Errorf("secret ref bundleID %q does not match config bundleID %q", ref.BundleID, bundleID)
	}
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
		BundleID bundleitemutils.BundleID `json:"bundleID"`
		ServerID spec.MCPServerID         `json:"serverID"`
		Kind     spec.MCPSecretKind       `json:"kind"`
		Slot     string                   `json:"slot"`
	}
	if err := json.Unmarshal(b, &wire); err != nil {
		return spec.MCPSecretRef{}, fmt.Errorf("secret ref %q is not valid json: %w", raw, err)
	}

	ref := spec.MCPSecretRef{
		BundleID: bundleitemutils.BundleID(strings.TrimSpace(string(wire.BundleID))),
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
		BundleID bundleitemutils.BundleID `json:"bundleID"`
		ServerID spec.MCPServerID         `json:"serverID"`
		Kind     spec.MCPSecretKind       `json:"kind"`
		Slot     string                   `json:"slot"`
	}{
		BundleID: r.BundleID,
		ServerID: r.ServerID,
		Kind:     r.Kind,
		Slot:     r.Slot,
	}
	return json.Marshal(wire)
}

func validateSecret(r spec.MCPSecretRef) error {
	if strings.TrimSpace(string(r.BundleID)) == "" {
		return errors.New("secret ref bundleID is empty")
	}
	if strings.TrimSpace(string(r.ServerID)) == "" {
		return errors.New("secret ref serverID is empty")
	}
	switch r.Kind {
	case spec.MCPSecretKindStdioEnv,
		spec.MCPSecretKindOAuthClientCredentials:
	default:
		return fmt.Errorf("secret ref kind %q is invalid", r.Kind)
	}
	if strings.TrimSpace(r.Slot) == "" {
		return errors.New("secret ref slot is empty")
	}
	switch r.Kind {
	case spec.MCPSecretKindOAuthClientCredentials:
		if r.Slot != normalizeSecretSlot("clientCredentials") {
			return fmt.Errorf(
				"secret ref slot %q is invalid for kind %q",
				r.Slot,
				r.Kind,
			)
		}
	case spec.MCPSecretKindStdioEnv:
		if err := validateEnvSecretSlot(r.Slot); err != nil {
			return err
		}
	}
	return nil
}

func normalizeAndValidateSecretSlot(kind spec.MCPSecretKind, slot string) (string, error) {
	raw := strings.TrimSpace(slot)
	if raw == "" {
		return "", errors.New("secret ref slot is empty")
	}

	switch kind {
	case spec.MCPSecretKindStdioEnv:
		if err := validateEnvSecretSlot(raw); err != nil {
			return "", err
		}
		return normalizeSecretSlot(raw), nil

	case spec.MCPSecretKindOAuthClientCredentials:
		if !strings.EqualFold(raw, "clientCredentials") {
			return "", fmt.Errorf(
				"secret ref slot %q is invalid for kind %q; expected clientCredentials",
				slot,
				kind,
			)
		}
		return normalizeSecretSlot("clientCredentials"), nil

	default:
		return "", fmt.Errorf("secret ref kind %q is invalid", kind)
	}
}

func validateEnvSecretSlot(key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("env key is empty")
	}
	if strings.TrimSpace(key) != key {
		return errors.New("env key has leading/trailing whitespace")
	}
	if strings.ContainsAny(key, "=\x00") {
		return errors.New("env key must not contain '=' or NUL")
	}
	for _, c := range key {
		if c < 0x20 || c == 0x7f {
			return fmt.Errorf("env key contains control character %q", c)
		}
	}
	return nil
}

func normalizeSecretKind(kind spec.MCPSecretKind) spec.MCPSecretKind {
	return spec.MCPSecretKind(strings.TrimSpace(string(kind)))
}

func normalizeSecretSlot(slot string) string {
	return strings.ToLower(strings.TrimSpace(slot))
}
