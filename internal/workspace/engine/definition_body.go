package engine

import "encoding/json"

// DecodeDefinitionBody decodes the canonical body held by a definition.
//
// Artifact adapters use this after QueryService has selected an already
// validated definition.
func DecodeDefinitionBody[T any](
	raw json.RawMessage,
) (T, error) {
	var output T
	if err := json.Unmarshal(raw, &output); err != nil {
		return output, err
	}
	return output, nil
}
