package shareable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type EntityType string

const (
	EntityCollection EntityType = "collection"
	EntityArtifact   EntityType = "artifact"
)

type SchemaKey struct {
	Entity EntityType `json:"entity"`
	// Kind retains the existing storage type to avoid breaking Collection
	// codecs. For EntityArtifact it carries the string representation of an
	// ArtifactKind and is validated accordingly.
	Kind          basespec.CollectionKind `json:"kind"`
	SchemaID      basespec.SchemaID       `json:"schemaID"`
	SchemaVersion string                  `json:"schemaVersion"`
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

func (k SchemaKey) Validate() error {
	switch k.Entity {
	case EntityCollection:
		if err := basespec.ValidateCollectionKind(k.Kind); err != nil {
			return err
		}
	case EntityArtifact:
		if err := basespec.ValidateArtifactKind(
			basespec.ArtifactKind(k.Kind),
		); err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"%w: unsupported shareable entity type %q",
			basespec.ErrInvalid,
			k.Entity,
		)
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

// EntityCanonicalizer supports both shareable Collection and Artifact
// documents without implying local entity creation or persistence.
type EntityCanonicalizer interface {
	CanonicalizeEntity(
		ctx context.Context,
		entity EntityType,
		raw []byte,
	) (ParsedDocument, error)
}

// ExpectedCanonicalizer is the narrow Artifact Store boundary for consumers
// that accept one known shareable schema. It ensures that callers do not
// bypass registry dispatch and validate an arbitrary schema before checking
// its expected identity.
type ExpectedCanonicalizer interface {
	CanonicalizeExpected(
		ctx context.Context,
		expected SchemaKey,
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
