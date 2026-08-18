package mcpbundle

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func DecodeCollectionData(
	raw json.RawMessage,
) (CollectionData, error) {
	value, err := jsonutil.DecodeCanonicalObject[CollectionData](raw, basespec.MaxDefinitionBytes)
	if err != nil {
		return CollectionData{}, err
	}
	if _, err := EncodeCollectionData(value); err != nil {
		return CollectionData{}, err
	}
	return value, nil
}

func EncodeCollectionData(
	value CollectionData,
) (json.RawMessage, error) {
	if value.SchemaVersion != artifactbuiltin.MCPSchemaVersion ||
		value.DiscoveryPolicyRevision != artifactbuiltin.DecoderRevision {
		return nil, fmt.Errorf(
			"%w: invalid MCP Bundle Collection data",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateLogicalName(value.LogicalName); err != nil {
		return nil, err
	}
	if err := basespec.ValidateLogicalVersion(value.LogicalVersion, true); err != nil {
		return nil, err
	}
	if err := validateCollectionLabels(value.Labels); err != nil {
		return nil, err
	}
	if value.ManagedSourceID != "" {
		if err := basespec.ValidateSourceID(value.ManagedSourceID); err != nil {
			return nil, err
		}
	}
	return jsonutil.MarshalCanonicalObject(value, basespec.MaxDefinitionBytes)
}

func validateCollectionLabels(values map[string]string) error {
	if len(values) > basespec.MaxLabels {
		return fmt.Errorf(
			"%w: MCP Bundle labels exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxLabels,
		)
	}
	for key, value := range values {
		if err := basespec.ValidateIdentifier(
			"MCP Bundle label key",
			key,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := basespec.ValidateRequiredText(
			"MCP Bundle label value",
			value,
			basespec.MaxLabelValueBytes,
		); err != nil {
			return err
		}
	}
	return nil
}
