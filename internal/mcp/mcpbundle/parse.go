package mcpbundle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// BundleFromParsedDocument projects an Artifact Store-validated canonical MCP
// Bundle document into the MCP domain model.
//
// It is intentionally not a second schema-validation entry point. Untrusted
// bytes must first pass through shareable.ExpectedCanonicalizer with
// BundleCodec{}.Key(). This helper verifies the registry output identity,
// canonical-byte invariant, and document digest before MCP lifecycle code
// consumes the typed projection.
func BundleFromParsedDocument(
	input shareable.ParsedDocument,
) (BundleDocument, error) {
	expected := artifactbuiltin.MCPBundleSchemaKey
	if err := validateParsedMCPDocument(
		input,
		expected,
		"MCP Bundle",
	); err != nil {
		return BundleDocument{}, err
	}

	var output BundleDocument
	if err := decodeCanonicalDocument(input.Raw, &output); err != nil {
		return BundleDocument{}, err
	}
	if output.Kind != artifactbuiltin.BundleKind ||
		output.SchemaID != artifactbuiltin.BundleSchemaID ||
		output.SchemaVersion != artifactbuiltin.MCPSchemaVersion {
		return BundleDocument{}, fmt.Errorf(
			"%w: canonical MCP Bundle output has another schema identity",
			basespec.ErrInvalid,
		)
	}
	if output.Digest != input.Digest {
		return BundleDocument{}, fmt.Errorf(
			"%w: canonical MCP Bundle digest does not match registry output",
			basespec.ErrDigestMismatch,
		)
	}
	return output, nil
}

func decodeCanonicalDocument(
	raw json.RawMessage,
	target any,
) error {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return fmt.Errorf(
			"%w: canonical MCP document output is invalid: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if !bytes.Equal(canonical, raw) {
		return fmt.Errorf(
			"%w: Artifact Store returned non-canonical MCP document JSON",
			basespec.ErrInvalid,
		)
	}

	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf(
			"%w: decode canonical MCP document output: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("canonical MCP document contains trailing JSON")
		}
		return fmt.Errorf(
			"%w: decode canonical MCP document output: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return nil
}

func validateParsedMCPDocument(
	input shareable.ParsedDocument,
	expected shareable.SchemaKey,
	subject string,
) error {
	if input.Key != expected {
		return fmt.Errorf(
			"%w: expected canonical %s schema %q/%q/%q",
			basespec.ErrInvalid,
			subject,
			expected.Kind,
			expected.SchemaID,
			expected.SchemaVersion,
		)
	}
	if err := input.Validate(); err != nil {
		return fmt.Errorf(
			"%w: invalid canonical %s registry output: %w",
			basespec.ErrInvalid,
			subject,
			err,
		)
	}
	return nil
}
