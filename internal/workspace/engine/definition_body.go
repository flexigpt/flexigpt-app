package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// DecodeDefinitionBody decodes the canonical body held by a definition.
//
// Artifact adapters use this after QueryService has selected an already
// validated definition.
func DecodeDefinitionBody[T any](
	raw json.RawMessage,
) (T, error) {
	var output T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return output, fmt.Errorf(
			"%w: decode Workspace definition body: %w",
			ErrInvalidWorkspace,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("definition body contains trailing JSON values")
		}
		return output, fmt.Errorf(
			"%w: decode Workspace definition body: %w",
			ErrInvalidWorkspace,
			err,
		)
	}
	return output, nil
}
