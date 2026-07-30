package skillbundle

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type CollectionData struct {
	SchemaVersion           string `json:"schemaVersion"`
	DiscoveryPolicyRevision string `json:"discoveryPolicyRevision"`

	// BootstrapKey is retained only to locate collections written before the
	// durable Collection.IdempotencyKey was introduced. New bundle creation
	// leaves this field empty. It is never a CollectionID, ArtifactID, or
	// persistent Skill reference.
	BootstrapKey string `json:"bootstrapKey,omitempty"`
}

func EncodeCollectionData(value CollectionData) (json.RawMessage, error) {
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
	return value, nil
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
	if value.BootstrapKey != "" {
		if err := basespec.ValidateIdentifier(
			"skill bundle bootstrap key",
			value.BootstrapKey,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
	}
	return nil
}

func EmptyArtifactData() json.RawMessage {
	return json.RawMessage(jsonutil.EmptyObject)
}
