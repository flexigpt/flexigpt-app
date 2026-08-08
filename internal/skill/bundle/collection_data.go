package bundle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type CollectionData struct {
	SchemaVersion           string                  `json:"schemaVersion"`
	DiscoveryPolicyRevision string                  `json:"discoveryPolicyRevision"`
	LogicalName             basespec.LogicalName    `json:"logicalName"`
	LogicalVersion          basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	Labels                  map[string]string       `json:"labels,omitempty"`

	// ManagedSourceID identifies the managed Source provisioned exclusively
	// for this bundle. An empty value means attached Sources are externally
	// administered and are not deleted with the bundle.
	ManagedSourceID basespec.SourceID `json:"managedSourceID,omitempty"`
}

func (d CollectionData) Clone() CollectionData {
	d.Labels = maps.Clone(d.Labels)
	return d
}

func EncodeCollectionData(value CollectionData) (json.RawMessage, error) {
	value = value.Clone()
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

func DecodeCollectionData(
	raw json.RawMessage,
) (CollectionData, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return CollectionData{}, err
	}

	var value CollectionData
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return CollectionData{}, fmt.Errorf(
			"%w: decode skill bundle collection data: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	if err := ValidateCollectionData(value); err != nil {
		return CollectionData{}, err
	}
	return value.Clone(), nil
}

func ValidateCollectionData(value CollectionData) error {
	if value.SchemaVersion != CollectionSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported skill bundle schema version %q",
			basespec.ErrInvalid,
			value.SchemaVersion,
		)
	}
	if err := basespec.ValidateRequiredText(
		"skill bundle discovery policy revision",
		value.DiscoveryPolicyRevision,
		basespec.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := builtinSchema.ValidateShareableCollectionMetadata(
		string(value.LogicalName),
		string(value.LogicalVersion),
		"",
		"",
		value.Labels,
	); err != nil {
		return err
	}
	if value.ManagedSourceID != "" {
		if err := basespec.ValidateSourceID(value.ManagedSourceID); err != nil {
			return err
		}
	}

	return nil
}
