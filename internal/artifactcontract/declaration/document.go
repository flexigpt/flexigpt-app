package declaration

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// DecodeDocumentInto performs common strict canonical JSON and JSON Schema
// handling for an independently versioned declaration contract.
func DecodeDocumentInto(
	raw []byte,
	_ *jsonschema.Schema,
	target any,
) error {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return err
	}

	// Each concrete Decode function validates its decoded document
	// immediately afterward. Keep schema execution there so direct decoding
	// and source decoding do not validate the same document twice.
	return jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		target,
		basespec.MaxDefinitionBytes,
	)
}

// ValidateEntryDocument validates the original canonical Entry bytes against
// a concrete declaration schema. This preserves field presence for nested
// declarations, including values such as maxIterations: 0 that would be lost
// when an omitempty Go struct is marshaled again.
func ValidateEntryDocument(
	entry Entry,
	compiled *jsonschema.Schema,
) error {
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return err
	}
	return jsonutil.ValidateJSONSchema(
		compiled,
		json.RawMessage(raw),
		basespec.MaxDefinitionBytes,
	)
}

func ValidateDocument(
	compiled *jsonschema.Schema,
	value any,
) error {
	return jsonutil.ValidateJSONSchema(
		compiled,
		value,
		basespec.MaxDefinitionBytes,
	)
}

func CanonicalDocumentJSON(
	value any,
) ([]byte, error) {
	return jsonutil.MarshalCanonicalObject(
		value,
		basespec.MaxDefinitionBytes,
	)
}

func DocumentDigest(
	value any,
) (cryptoutil.Digest, error) {
	digest, err := cryptoutil.CanonicalDigest(value)
	if err != nil {
		return "", fmt.Errorf(
			"calculate canonical declaration digest: %w",
			err,
		)
	}
	return digest, nil
}
