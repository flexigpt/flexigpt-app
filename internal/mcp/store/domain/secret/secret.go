package secret

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type mcpSecretRefWire struct {
	Server artifact.ArtifactRef `json:"server"`
	Kind   MCPSecretKind        `json:"kind"`
	Slot   string               `json:"slot"`
}

func NewMCPSecretRefString(
	server artifact.ArtifactRef,
	kind MCPSecretKind,
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
	kind MCPSecretKind,
	slot string,
) (MCPSecretRef, error) {
	kind = kind.normalized()
	normalizedSlot, err := kind.NormalizeSlot(slot)
	if err != nil {
		return MCPSecretRef{}, err
	}

	ref := MCPSecretRef{
		Server: server,
		Kind:   kind,
		Slot:   normalizedSlot,
	}
	if err := ref.Validate(); err != nil {
		return MCPSecretRef{}, err
	}
	return ref, nil
}

func ParseMCPSecretRef(raw string) (MCPSecretRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return MCPSecretRef{}, errors.New("secret ref is empty")
	}
	if !strings.HasPrefix(raw, SecretRefVersion+":") {
		return MCPSecretRef{}, fmt.Errorf("secret ref %q is not a %s ref", raw, SecretRefVersion)
	}

	encoded := strings.TrimPrefix(raw, SecretRefVersion+":")
	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return MCPSecretRef{}, fmt.Errorf("secret ref %q is not valid base64: %w", raw, err)
	}

	wire, err := jsonutil.DecodeCanonicalObjectExact[mcpSecretRefWire](
		b,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return MCPSecretRef{}, fmt.Errorf("secret ref %q is not valid json: %w", raw, err)
	}

	ref := MCPSecretRef{
		Server: wire.Server,
		Kind:   wire.Kind.normalized(),
		Slot:   normalizeSecretSlot(wire.Slot),
	}
	if err := ref.Validate(); err != nil {
		return MCPSecretRef{}, err
	}
	return ref, nil
}

func (ref MCPSecretRef) Matches(
	server artifact.ArtifactRef,
	kind MCPSecretKind,
	slot string,
) error {
	if err := server.Validate(); err != nil {
		return err
	}
	normalizedSlot, err := kind.NormalizeSlot(slot)
	if err != nil {
		return err
	}
	if ref.Server != server {
		return errors.New("secret ref server does not match requested Artifact")
	}
	if ref.Kind != kind.normalized() {
		return fmt.Errorf("secret ref kind %q does not match expected kind %q", ref.Kind, kind)
	}
	if ref.Slot != normalizedSlot {
		return fmt.Errorf("secret ref slot %q does not match expected slot %q", ref.Slot, normalizedSlot)
	}
	return nil
}

func GetMCPSecretRefStorageKey(r MCPSecretRef) string {
	raw, err := canonicalSecret(r)
	if err != nil {
		return ""
	}
	digest := cryptoutil.DigestBytes(raw)
	return SecretRefVersion + ":" + strings.TrimPrefix(
		string(digest),
		cryptoutil.DigestSHA256Prefix,
	)
}

func GetMCPSecretRefString(r MCPSecretRef) string {
	raw, err := canonicalSecret(r)
	if err != nil {
		return ""
	}
	return SecretRefVersion + ":" + base64.RawURLEncoding.EncodeToString(raw)
}

func canonicalSecret(r MCPSecretRef) ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	wire := struct {
		Server artifact.ArtifactRef `json:"server"`
		Kind   MCPSecretKind        `json:"kind"`
		Slot   string               `json:"slot"`
	}{
		Server: r.Server,
		Kind:   r.Kind,
		Slot:   r.Slot,
	}
	canonical, err := jsonutil.MarshalCanonicalObject(
		wire,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), canonical...), nil
}

func validateSecret(r MCPSecretRef) error {
	if err := r.Server.Validate(); err != nil {
		return fmt.Errorf("secret ref server: %w", err)
	}
	switch r.Kind {
	case MCPSecretKindStdioEnv,
		MCPSecretKindHTTPHeader,
		MCPSecretKindOAuthClientCredentials,
		MCPSecretKindOAuthToken:
	default:
		return fmt.Errorf("secret ref kind %q is invalid", r.Kind)
	}
	if strings.TrimSpace(r.Slot) == "" {
		return errors.New("secret ref slot is empty")
	}
	switch r.Kind {
	case MCPSecretKindOAuthClientCredentials:
		if r.Slot != normalizeSecretSlot("clientCredentials") {
			return fmt.Errorf(
				"secret ref slot %q is invalid for kind %q",
				r.Slot,
				r.Kind,
			)
		}
	case MCPSecretKindOAuthToken:
		if r.Slot != normalizeSecretSlot("token") {
			return fmt.Errorf(
				"secret ref slot %q is invalid for kind %q",
				r.Slot,
				r.Kind,
			)
		}
	case MCPSecretKindStdioEnv:
		if err := validateEnvSecretSlot(r.Slot); err != nil {
			return err
		}
	case MCPSecretKindHTTPHeader:
		if err := validateHTTPHeaderSecretSlot(r.Slot); err != nil {
			return err
		}
	}
	return nil
}

func normalizeAndValidateSecretSlot(kind MCPSecretKind, slot string) (string, error) {
	raw := strings.TrimSpace(slot)
	if raw == "" {
		return "", errors.New("secret ref slot is empty")
	}

	switch kind {
	case MCPSecretKindStdioEnv:
		if err := validateEnvSecretSlot(raw); err != nil {
			return "", err
		}
		return normalizeSecretSlot(raw), nil
	case MCPSecretKindHTTPHeader:
		if err := validateHTTPHeaderSecretSlot(raw); err != nil {
			return "", err
		}
		return normalizeSecretSlot(raw), nil
	case MCPSecretKindOAuthClientCredentials:
		if !strings.EqualFold(raw, "clientCredentials") {
			return "", fmt.Errorf(
				"secret ref slot %q is invalid for kind %q; expected clientCredentials",
				slot,
				kind,
			)
		}
		return normalizeSecretSlot("clientCredentials"), nil
	case MCPSecretKindOAuthToken:
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

func (kind MCPSecretKind) NormalizeSlot(slot string) (string, error) {
	return normalizeAndValidateSecretSlot(kind.normalized(), slot)
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

func (kind MCPSecretKind) normalized() MCPSecretKind {
	return MCPSecretKind(strings.TrimSpace(string(kind)))
}

func normalizeSecretSlot(slot string) string {
	return strings.ToLower(strings.TrimSpace(slot))
}
