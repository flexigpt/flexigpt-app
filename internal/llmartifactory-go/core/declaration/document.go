package declaration

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// DecodeDocumentInto validates the original canonical document before
// decoding it. Validating the original bytes preserves field presence, so
// values hidden by omitempty cannot bypass their schema constraints.
func DecodeDocumentInto(
	raw []byte,
	compiled jsonutil.JSONSchema,
	target any,
) error {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxDefinitionBytes,
	)
	if err != nil {
		return err
	}
	if err := jsonutil.ValidateJSONSchema(
		compiled,
		json.RawMessage(canonical),
		spec.MaxDefinitionBytes,
	); err != nil {
		return err
	}
	return jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		target,
		spec.MaxDefinitionBytes,
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
		len(entry.raw) > spec.MaxDefinitionBodyBytes ||
		entry.raw[0] != '{' {
		return fmt.Errorf(
			"%w: canonical declaration entry must be a bounded JSON object",
			spec.ErrInvalid,
		)
	}
	if err := jsonutil.ValidateJSONSchema(
		compiled,
		entry.raw,
		spec.MaxDefinitionBytes,
	); err != nil {
		return err
	}
	return jsonutil.DecodeCanonicalObjectBytesInto(
		entry.raw,
		target,
		spec.MaxDefinitionBodyBytes,
	)
}

func ValidateDocument(
	compiled jsonutil.JSONSchema,
	value any,
) error {
	return jsonutil.ValidateJSONSchema(
		compiled,
		value,
		spec.MaxDefinitionBytes,
	)
}

func CanonicalDocumentJSON(
	value any,
) ([]byte, error) {
	return jsonutil.MarshalCanonicalObject(
		value,
		spec.MaxDefinitionBytes,
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
