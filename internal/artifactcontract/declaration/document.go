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
	compiled *jsonschema.Schema,
	target any,
) error {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return err
	}
	if err := jsonutil.ValidateJSONSchema(
		compiled,
		json.RawMessage(canonical),
		basespec.MaxDefinitionBytes,
	); err != nil {
		return err
	}
	return jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		target,
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
