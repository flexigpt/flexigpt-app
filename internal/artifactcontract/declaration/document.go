package declaration

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// DecodeDocumentInto validates the original canonical document before
// decoding it. Validating the original bytes preserves field presence, so
// values hidden by omitempty cannot bypass their schema constraints.
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

// DecodeEntryDocumentInto validates and decodes the Entry's original
// canonical bytes without first re-marshalling its Go representation.
func DecodeEntryDocumentInto(
	entry Entry,
	compiled *jsonschema.Schema,
	target any,
) error {
	if len(entry.raw) == 0 ||
		len(entry.raw) > basespec.MaxDefinitionBodyBytes ||
		entry.raw[0] != '{' {
		return fmt.Errorf(
			"%w: canonical declaration entry must be a bounded JSON object",
			basespec.ErrInvalid,
		)
	}
	if err := jsonutil.ValidateJSONSchema(
		compiled,
		entry.raw,
		basespec.MaxDefinitionBytes,
	); err != nil {
		return err
	}
	return jsonutil.DecodeCanonicalObjectBytesInto(
		entry.raw,
		target,
		basespec.MaxDefinitionBodyBytes,
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
