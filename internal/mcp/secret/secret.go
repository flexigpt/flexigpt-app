package secret

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func NewMCPSecretRefString(
	server artifact.ArtifactRef,
	kind spec.MCPSecretKind,
	slot string,
) (string, error) {
	ref, err := NewMCPSecretRef(server, kind, slot)
	if err != nil {
		return "", err
	}
	out := GetMCPSecretRefString(ref)
	if out == "" {
		return "", errors.New("could not encode secret ref")
	}
	return out, nil
}

func NewMCPSecretRef(
	server artifact.ArtifactRef,
	kind spec.MCPSecretKind,
	slot string,
) (spec.MCPSecretRef, error) {
	kind = normalizeSecretKind(kind)
	normalizedSlot, err := normalizeAndValidateSecretSlot(kind, slot)
	if err != nil {
		return spec.MCPSecretRef{}, err
	}

	ref := spec.MCPSecretRef{
		Server: server,
		Kind:   kind,
		Slot:   normalizedSlot,
	}
	if err := validateSecret(ref); err != nil {
		return spec.MCPSecretRef{}, err
	}
	return ref, nil
}

func ValidateMCPSecretRef(
	raw string,
	server artifact.ArtifactRef,
	kind spec.MCPSecretKind,
	slot string,
) error {
	ref, err := ParseMCPSecretRef(raw)
	if err != nil {
		return err
	}
	kind = normalizeSecretKind(kind)
	slot = normalizeSecretSlot(slot)
	if ref.Server != server {
		return errors.New("secret ref server does not match requested Artifact")
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
		Server artifact.ArtifactRef `json:"server"`
		Kind   spec.MCPSecretKind   `json:"kind"`
		Slot   string               `json:"slot"`
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return spec.MCPSecretRef{}, fmt.Errorf("secret ref %q is not valid json: %w", raw, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("secret ref contains trailing JSON")
		}
		return spec.MCPSecretRef{}, fmt.Errorf(
			"secret ref %q is not valid json: %w",
			raw,
			err,
		)
	}

	ref := spec.MCPSecretRef{
		Server: wire.Server,
		Kind:   normalizeSecretKind(wire.Kind),
		Slot:   normalizeSecretSlot(wire.Slot),
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
		Server artifact.ArtifactRef `json:"server"`
		Kind   spec.MCPSecretKind   `json:"kind"`
		Slot   string               `json:"slot"`
	}{
		Server: r.Server,
		Kind:   r.Kind,
		Slot:   r.Slot,
	}
	return json.Marshal(wire)
}

func validateSecret(r spec.MCPSecretRef) error {
	if err := r.Server.Validate(); err != nil {
		return fmt.Errorf("secret ref server: %w", err)
	}
	switch r.Kind {
	case spec.MCPSecretKindStdioEnv,
		spec.MCPSecretKindHTTPHeader,
		spec.MCPSecretKindOAuthClientCredentials,
		spec.MCPSecretKindOAuthToken:
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
	case spec.MCPSecretKindOAuthToken:
		if r.Slot != normalizeSecretSlot("token") {
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
	case spec.MCPSecretKindHTTPHeader:
		if err := validateHTTPHeaderSecretSlot(r.Slot); err != nil {
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
	case spec.MCPSecretKindHTTPHeader:
		if err := validateHTTPHeaderSecretSlot(raw); err != nil {
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
	case spec.MCPSecretKindOAuthToken:
		if !strings.EqualFold(raw, "token") {
			return "", fmt.Errorf(
				"secret ref slot %q is invalid for kind %q; expected token",
				slot,
				kind,
			)
		}
		return normalizeSecretSlot("token"), nil

	default:
		return "", fmt.Errorf("secret ref kind %q is invalid", kind)
	}
}

func validateHTTPHeaderSecretSlot(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("http header name is empty")
	}
	if strings.TrimSpace(name) != name {
		return errors.New("http header name has leading/trailing whitespace")
	}

	for _, r := range name {
		if !isHTTPTokenRune(r) {
			return fmt.Errorf("http header name contains invalid character %q", r)
		}
	}

	return nil
}

func isHTTPTokenRune(r rune) bool {
	if r >= 'A' && r <= 'Z' ||
		r >= 'a' && r <= 'z' ||
		r >= '0' && r <= '9' {
		return true
	}

	switch r {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	default:
		return false
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
