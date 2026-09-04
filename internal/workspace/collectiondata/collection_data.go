package collectiondata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

// collectionDataWire accepts the former local policy-revision field while
// writing only the current Workspace data shape. Catalog invalidation is now
// owned by the registered Artifact Store collection behavior.
type collectionDataWire struct {
	DiscoveryPolicyRevision string                    `json:"discoveryPolicyRevision,omitempty"`
	Discovery               spec.DiscoveryPreferences `json:"discovery"`
}

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
	var wire collectionDataWire
	if err := decoder.Decode(&wire); err != nil {
		return spec.CollectionData{}, err
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New(
				"workspace collection data contains trailing JSON values",
			)
		}
		return spec.CollectionData{}, err
	}
	value := spec.CollectionData{
		Discovery: wire.Discovery,
	}
	if err := ValidateCollectionData(value); err != nil {
		return spec.CollectionData{}, err
	}
	return value, nil
}

func ValidateCollectionData(value spec.CollectionData) error {
	if err := spec.ValidateDiscoveryPreferences(value.Discovery); err != nil {
		return fmt.Errorf("%w: %w", spec.ErrInvalidWorkspace, err)
	}
	return nil
}
