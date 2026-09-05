// Package providerapi defines Artifact Store's stable inbound provider
// contracts.
//
// Providers may depend on this package and basespec only from artifactstore.
package providerapi

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

// SchemaKind is the entity-neutral kind portion of a schema key.
//
// A SchemaKind intentionally does not imply either CollectionKind or
// ArtifactKind. The Entity field establishes which validation rule applies.
type SchemaKind string

type SchemaKey struct {
	Entity        EntityType        `json:"entity"`
	Kind          SchemaKind        `json:"kind"`
	SchemaID      basespec.SchemaID `json:"schemaID"`
	SchemaVersion string            `json:"schemaVersion"`
}

func CollectionSchemaKey(
	kind basespec.CollectionKind,
	schemaID basespec.SchemaID,
	schemaVersion string,
) SchemaKey {
	return SchemaKey{
		Entity:        EntityCollection,
		Kind:          SchemaKind(kind),
		SchemaID:      schemaID,
		SchemaVersion: schemaVersion,
	}
}

func ArtifactSchemaKey(
	kind basespec.ArtifactKind,
	schemaID basespec.SchemaID,
	schemaVersion string,
) SchemaKey {
	return SchemaKey{
		Entity:        EntityArtifact,
		Kind:          SchemaKind(kind),
		SchemaID:      schemaID,
		SchemaVersion: schemaVersion,
	}
}

func (k SchemaKey) Validate() error {
	switch k.Entity {
	case EntityCollection:
		if err := basespec.ValidateCollectionKind(
			basespec.CollectionKind(k.Kind),
		); err != nil {
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
			"%w: unsupported provider schema entity %q",
			basespec.ErrInvalid,
			k.Entity,
		)
	}

	if err := basespec.ValidateSchemaID(k.SchemaID); err != nil {
		return err
	}
	return basespec.ValidateRequiredText(
		"provider schema version",
		k.SchemaVersion,
		basespec.MaxVersionBytes,
	)
}

// ParsedDocument is a canonical document accepted by a registered schema
// codec.
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
		return fmt.Errorf(
			"%w: provider parsed document is empty",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func (d ParsedDocument) Clone() ParsedDocument {
	output := d
	output.Raw = append(json.RawMessage(nil), d.Raw...)
	return output
}

// SchemaCodec supplies one published JSON Schema and domain-specific semantic
// canonicalization.
//
// Artifact Store owns schema registration, schema execution, registry
// dispatch, canonical JSON checks, and output verification. The provider owns
// only its schema semantics and canonicalization rules.
type SchemaCodec interface {
	Key() SchemaKey
	JSONSchema() []byte

	Canonicalize(
		ctx context.Context,
		raw []byte,
	) (ParsedDocument, error)
}

// EntityCanonicalizer supports dispatch by an entity type inferred from a
// caller-owned context.
type EntityCanonicalizer interface {
	CanonicalizeEntity(
		ctx context.Context,
		entity EntityType,
		raw []byte,
	) (ParsedDocument, error)
}

// ExpectedCanonicalizer is the narrow Artifact Store capability required by a
// provider that accepts a document with a known schema identity.
type ExpectedCanonicalizer interface {
	CanonicalizeExpected(
		ctx context.Context,
		expected SchemaKey,
		raw []byte,
	) (ParsedDocument, error)
}

// SchemaCatalog is the narrow setup-time capability supplied to a decoder that
// must canonicalize source documents through Artifact Store's schema registry.
//
// It deliberately exposes only known keys and expected-schema
// canonicalization. It does not expose concrete registry implementation,
// database access, or Artifact Store services.
type SchemaCatalog interface {
	ExpectedCanonicalizer

	Keys() []SchemaKey
}
