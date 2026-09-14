package artifact

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// DecodeDataObject decodes generic Artifact.Data without assigning ownership
// of the object shape to one consumer domain.
func DecodeDataObject(
	raw json.RawMessage,
) (map[string]json.RawMessage, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &values); err != nil {
		return nil, fmt.Errorf(
			"%w: decode Artifact local data: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	output := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		output[key] = append(json.RawMessage(nil), value...)
	}
	return output, nil
}

// EncodeDataObject canonicalizes generic Artifact.Data while preserving
// independently owned namespaced consumer values.
func EncodeDataObject(
	values map[string]json.RawMessage,
) (json.RawMessage, error) {
	output := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		canonical, err := jsonutil.Canonicalize(value)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: Artifact local data field %q: %w",
				basespec.ErrInvalid,
				key,
				err,
			)
		}
		output[key] = json.RawMessage(canonical)
	}
	return jsonutil.MarshalCanonicalObject(
		output,
		basespec.MaxLocalDataBytes,
	)
}
