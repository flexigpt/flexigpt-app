package definition

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func EncodeBody(value any) (json.RawMessage, error) {
	raw, err := jsonutil.MarshalCanonicalObject(
		value,
		model.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: encode definition body: %w", model.ErrInvalid, err)
	}
	return raw, nil
}

func DecodeBody[T any](raw json.RawMessage) (T, error) {
	var output T

	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		raw,
		&output,
		model.MaxDefinitionBodyBytes,
	); err != nil {
		return output, fmt.Errorf(
			"%w: decode definition body: %w",
			model.ErrInvalid,
			err,
		)
	}
	return output, nil
}
