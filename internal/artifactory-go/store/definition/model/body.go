package model

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func EncodeBody(value any) (json.RawMessage, error) {
	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: encode definition body: %w", spec.ErrInvalid, err)
	}
	return raw, nil
}

func DecodeBody[T any](raw json.RawMessage) (T, error) {
	var output T

	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		raw,
		&output,
		spec.MaxDefinitionBodyBytes,
	); err != nil {
		return output, fmt.Errorf(
			"%w: decode definition body: %w",
			spec.ErrInvalid,
			err,
		)
	}
	return output, nil
}
