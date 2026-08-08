package shareable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type EntityType string

const EntityCollection EntityType = "collection"

type SchemaKey struct {
	Entity        EntityType              `json:"entity"`
	Kind          basespec.CollectionKind `json:"kind"`
	SchemaID      basespec.SchemaID       `json:"schemaID"`
	SchemaVersion string                  `json:"schemaVersion"`
}

func (k SchemaKey) Validate() error {
	if k.Entity != EntityCollection {
		return fmt.Errorf(
			"%w: unsupported shareable entity type %q",
			basespec.ErrInvalid,
			k.Entity,
		)
	}
	if err := basespec.ValidateCollectionKind(k.Kind); err != nil {
		return err
	}
	if err := basespec.ValidateSchemaID(k.SchemaID); err != nil {
		return err
	}
	return basespec.ValidateRequiredText(
		"shareable schema version",
		k.SchemaVersion,
		basespec.MaxVersionBytes,
	)
}

type ParsedDocument struct {
	Key    SchemaKey
	Digest cryptoutil.Digest
	Raw    json.RawMessage
}

func (d ParsedDocument) Validate() error {
	if err := d.Key.Validate(); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(d.Digest); err != nil {
		return err
	}
	if len(d.Raw) == 0 {
		return fmt.Errorf("%w: shareable document is empty", basespec.ErrInvalid)
	}
	return nil
}

func (d ParsedDocument) Clone() ParsedDocument {
	output := d
	output.Raw = append(json.RawMessage(nil), d.Raw...)
	return output
}

type Codec interface {
	Key() SchemaKey
	JSONSchema() []byte

	// Canonicalize strictly decodes, validates, calculates or verifies the
	// optional supplied digest, and returns canonical JSON containing the
	// calculated digest. It does not persist source content.
	Canonicalize(
		ctx context.Context,
		raw []byte,
	) (ParsedDocument, error)
}

// Canonicalizer is the narrow validation capability needed by artifact-family
// hydration and linked-source descriptor readers.
type Canonicalizer interface {
	Canonicalize(
		ctx context.Context,
		raw []byte,
	) (ParsedDocument, error)
}
