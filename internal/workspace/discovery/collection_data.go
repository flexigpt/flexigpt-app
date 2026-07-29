package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

func EncodeCollectionData(value spec.CollectionData) (json.RawMessage, error) {
	if err := ValidateCollectionData(value); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func DecodeCollectionData(raw json.RawMessage) (spec.CollectionData, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return spec.CollectionData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var value spec.CollectionData
	if err := decoder.Decode(&value); err != nil {
		return spec.CollectionData{}, err
	}
	if err := ValidateCollectionData(value); err != nil {
		return spec.CollectionData{}, err
	}
	return value, nil
}

func ValidateCollectionData(value spec.CollectionData) error {
	if err := validateDiscoveryPreferences(value.Discovery); err != nil {
		return fmt.Errorf("%w: %w", spec.ErrInvalidWorkspace, err)
	}
	if err := basespec.ValidateRequiredText(
		"workspace discovery policy revision",
		value.DiscoveryPolicyRevision,
		basespec.MaxVersionBytes,
	); err != nil {
		return fmt.Errorf(
			"%w: %w",
			spec.ErrInvalidWorkspace,
			err,
		)
	}
	return nil
}
