package providerapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func EncodeBody(value any) (json.RawMessage, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode definition body: %w", err)
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: encode definition body: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return json.RawMessage(canonical), nil
}

func DecodeBody[T any](raw json.RawMessage) (T, error) {
	var output T

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return output, fmt.Errorf(
			"%w: decode definition body: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return output, fmt.Errorf(
			"%w: decode definition body: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("definition body contains trailing JSON values")
		}
		return output, fmt.Errorf(
			"%w: decode definition body: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return output, nil
}
