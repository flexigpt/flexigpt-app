package declaration

import (
	"encoding/json"
	"fmt"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// DecodeSourceDocumentInto prepares a standalone JSON declaration and
// validates its canonical body before decoding. Validation uses the prepared
// bytes, not a re-marshalled Go value, so field presence is preserved.
func DecodeSourceDocumentInto(
	raw []byte,
	expected schemaModel.Key,
	requireAPIVersion bool,
	compiled jsonutil.JSONSchema,
	target any,
) error {
	canonical, err := PrepareSourceDocument(
		expected,
		raw,
		requireAPIVersion,
	)
	if err != nil {
		return err
	}
	if err := jsonutil.ValidateJSONSchema(
		compiled,
		json.RawMessage(canonical),
		storeSpec.MaxDefinitionBytes,
	); err != nil {
		return err
	}
	return jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		target,
		storeSpec.MaxDefinitionBytes,
	)
}

// DecodeEntryDocumentInto validates and decodes the Entry's original
// canonical bytes without first re-marshalling its Go representation.
func DecodeEntryDocumentInto(
	entry Entry,
	compiled jsonutil.JSONSchema,
	target any,
) error {
	if len(entry.raw) == 0 ||
		len(entry.raw) > storeSpec.MaxDefinitionBodyBytes ||
		entry.raw[0] != '{' {
		return fmt.Errorf(
			"%w: canonical declaration entry must be a bounded JSON object",
			storeSpec.ErrInvalid,
		)
	}
	if err := jsonutil.ValidateJSONSchema(
		compiled,
		entry.raw,
		storeSpec.MaxDefinitionBytes,
	); err != nil {
		return err
	}
	return jsonutil.DecodeCanonicalObjectBytesInto(
		entry.raw,
		target,
		storeSpec.MaxDefinitionBodyBytes,
	)
}

func ValidateDocument(
	compiled jsonutil.JSONSchema,
	value any,
) error {
	return jsonutil.ValidateJSONSchema(
		compiled,
		value,
		storeSpec.MaxDefinitionBytes,
	)
}

func CanonicalDocumentJSON(
	value any,
) ([]byte, error) {
	return jsonutil.MarshalCanonicalObject(
		value,
		storeSpec.MaxDefinitionBytes,
	)
}

func DocumentDigest(
	value any,
) (cryptoutil.Digest, error) {
	raw, err := CanonicalDocumentJSON(value)
	if err != nil {
		return "", fmt.Errorf(
			"calculate canonical declaration digest: %w",
			err,
		)
	}
	return cryptoutil.DigestBytes(raw), nil
}
