package schema

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
)

// ServerFromParsedDocument projects an Artifact Store-validated standalone
// MCP Server document into the MCP domain model.
//
// It is not a portable-document ingress path. Untrusted bytes must first pass
// through shareable.ExpectedCanonicalizer using ServerCodec{}.Key().
func ServerFromParsedDocument(
	input shareable.ParsedDocument,
) (ServerDocument, error) {
	expected := ServerCodec{}.Key()
	if err := validateParsedMCPDocument(
		input,
		expected,
		"MCP Server",
	); err != nil {
		return ServerDocument{}, err
	}

	var output ServerDocument
	if err := decodeCanonicalDocument(input.Raw, &output); err != nil {
		return ServerDocument{}, err
	}
	if output.Kind != ServerKind ||
		output.SchemaID != ServerSchemaID ||
		output.SchemaVersion != SchemaVersion {
		return ServerDocument{}, fmt.Errorf(
			"%w: canonical MCP Server output has another schema identity",
			basespec.ErrInvalid,
		)
	}
	if output.Digest != input.Digest {
		return ServerDocument{}, fmt.Errorf(
			"%w: canonical MCP Server digest does not match registry output",
			basespec.ErrDigestMismatch,
		)
	}
	return output, nil
}

// PolicyFromParsedDocument projects an Artifact Store-validated standalone
// MCP Policy document into the MCP domain model.
//
// It is not a portable-document ingress path. Untrusted bytes must first pass
// through shareable.ExpectedCanonicalizer using PolicyCodec{}.Key().
func PolicyFromParsedDocument(
	input shareable.ParsedDocument,
) (PolicyDocument, error) {
	expected := PolicyCodec{}.Key()
	if err := validateParsedMCPDocument(
		input,
		expected,
		"MCP Policy",
	); err != nil {
		return PolicyDocument{}, err
	}

	var output PolicyDocument
	if err := decodeCanonicalDocument(input.Raw, &output); err != nil {
		return PolicyDocument{}, err
	}
	if output.Kind != PolicyKind ||
		output.SchemaID != PolicySchemaID ||
		output.SchemaVersion != SchemaVersion {
		return PolicyDocument{}, fmt.Errorf(
			"%w: canonical MCP Policy output has another schema identity",
			basespec.ErrInvalid,
		)
	}
	if output.Digest != input.Digest {
		return PolicyDocument{}, fmt.Errorf(
			"%w: canonical MCP Policy digest does not match registry output",
			basespec.ErrDigestMismatch,
		)
	}
	return output, nil
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
