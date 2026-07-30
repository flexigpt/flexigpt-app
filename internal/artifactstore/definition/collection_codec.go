package definition

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func MarshalCollectionDefinition(
	input CollectionDefinition,
) ([]byte, error) {
	canonical, err := CanonicalizeCollectionDefinition(input)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("marshal portable collection definition: %w", err)
	}
	return jsonutil.Canonicalize(raw)
}

func ParseCollectionDefinition(
	raw []byte,
) (CollectionDefinition, error) {
	canonicalJSON, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return CollectionDefinition{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(canonicalJSON))
	decoder.DisallowUnknownFields()
	var value CollectionDefinition
	if err := decoder.Decode(&value); err != nil {
		return CollectionDefinition{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("portable collection JSON has trailing values")
		}
		return CollectionDefinition{}, fmt.Errorf(
			"%w: decode portable collection definition: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return CanonicalizeCollectionDefinition(value)
}
