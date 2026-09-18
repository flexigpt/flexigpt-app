package resolve

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const mappedIdentifierSeparator = "."

// MappedTarget is a non-Artifact target returned by a registered fallback
// provider. It has no ArtifactRef, Definition, Source binding, local data, or
// lifecycle state.
type MappedTarget struct {
	Provider   string               `json:"provider"`
	Identifier string               `json:"identifier"`
	Type       declaration.Type     `json:"type"`
	Name       basespec.LogicalName `json:"name"`
	Builtin    bool                 `json:"builtin"`
}

// Validate validates the provider-independent portion of a mapped target.
func (m MappedTarget) Validate() error {
	if err := m.Type.Validate(); err != nil {
		return err
	}
	if err := m.Name.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"mapped target provider",
		m.Provider,
		basespec.MaxKindBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"mapped target identifier",
		m.Identifier,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return nil
}

type FallbackTarget struct {
	Artifact *artifact.ArtifactRef
	Mapped   *MappedTarget
}

// Validate validates that exactly one fallback target form is present.
func (t FallbackTarget) Validate() error {
	switch {
	case t.Artifact == nil && t.Mapped == nil:
		return fmt.Errorf(
			"%w: fallback target is empty",
			basespec.ErrInvalid,
		)
	case t.Artifact != nil && t.Mapped != nil:
		return fmt.Errorf(
			"%w: fallback target contains both Artifact and mapped targets",
			basespec.ErrInvalid,
		)
	}

	if t.Artifact != nil {
		if err := t.Artifact.Validate(); err != nil {
			return fmt.Errorf("fallback Artifact target: %w", err)
		}
		return nil
	}

	return t.Mapped.Validate()
}

// EncodeMappedIdentifier serializes a versioned family-specific mapped target
// payload. Payloads are canonical JSON objects encoded as:
//
//	v1.<base64url-canonical-json>
func EncodeMappedIdentifier[T any](
	version string,
	value T,
) (string, error) {
	if err := validateMappedIdentifierVersion(version); err != nil {
		return "", err
	}

	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 || raw[0] != '{' {
		return "", fmt.Errorf(
			"%w: mapped identifier payload must be a JSON object",
			basespec.ErrInvalid,
		)
	}

	identifier := version + mappedIdentifierSeparator +
		base64.RawURLEncoding.EncodeToString(raw)

	if err := basespec.ValidateRequiredText(
		"mapped target identifier",
		identifier,
		basespec.MaxURIBytes,
	); err != nil {
		return "", err
	}
	return identifier, nil
}

// DecodeMappedIdentifier decodes a family-specific versioned mapped-target
// payload. The encoded JSON must already be canonical and must not contain
// unknown fields for the destination struct.
func DecodeMappedIdentifier[T any](
	identifier string,
	expectedVersion string,
) (T, error) {
	var zero T

	if err := validateMappedIdentifierVersion(expectedVersion); err != nil {
		return zero, err
	}
	if err := basespec.ValidateRequiredText(
		"mapped target identifier",
		identifier,
		basespec.MaxURIBytes,
	); err != nil {
		return zero, err
	}

	version, encoded, found := strings.Cut(
		identifier,
		mappedIdentifierSeparator,
	)
	if !found || version != expectedVersion || encoded == "" {
		return zero, fmt.Errorf(
			"%w: unsupported mapped identifier version",
			basespec.ErrInvalid,
		)
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return zero, fmt.Errorf(
			"%w: decode mapped identifier: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if len(raw) == 0 || len(raw) > basespec.MaxLocalDataBytes {
		return zero, fmt.Errorf(
			"%w: mapped identifier payload size is invalid",
			basespec.ErrInvalid,
		)
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return zero, fmt.Errorf(
			"%w: mapped identifier payload: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if !bytes.Equal(raw, canonical) {
		return zero, fmt.Errorf(
			"%w: mapped identifier payload is not canonical",
			basespec.ErrInvalid,
		)
	}

	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()

	var value T
	if err := decoder.Decode(&value); err != nil {
		return zero, fmt.Errorf(
			"%w: decode mapped identifier payload: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return zero, fmt.Errorf(
				"%w: mapped identifier has multiple JSON values",
				basespec.ErrInvalid,
			)
		}
		return zero, fmt.Errorf(
			"%w: mapped identifier trailing JSON: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	return value, nil
}

func validateMappedIdentifierVersion(version string) error {
	return basespec.ValidateRequiredText(
		"mapped identifier version",
		version,
		basespec.MaxKindBytes,
	)
}
