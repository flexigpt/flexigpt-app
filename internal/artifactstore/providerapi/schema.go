// Package providerapi defines Artifact Store's stable inbound provider
// contracts.
//
// Providers may depend on this package and basespec only from artifactstore.
package providerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
)

// SchemaCodec supplies one published JSON Schema and domain-specific semantic
// projection.
//
// Artifact Store owns schema registration, schema execution, registry
// dispatch, input canonicalization, and output verification. Canonicalize
// receives an already-canonical JSON object that has already passed the
// registered JSON Schema. A codec therefore does not canonicalize or execute
// the same JSON Schema again.
type SchemaCodec interface {
	Key() schema.Key
	JSONSchema() []byte

	Canonicalize(
		ctx context.Context,
		raw []byte,
	) (schema.ParsedDocument, error)
}

// EntityCanonicalizer supports dispatch by an entity type inferred from a
// caller-owned context.
type EntityCanonicalizer interface {
	CanonicalizeEntity(
		ctx context.Context,
		entity schema.EntityType,
		raw []byte,
	) (schema.ParsedDocument, error)
}

// ExpectedCanonicalizer is the narrow Artifact Store capability required by a
// provider that accepts a document with a known schema identity.
type ExpectedCanonicalizer interface {
	CanonicalizeExpected(
		ctx context.Context,
		expected schema.Key,
		raw []byte,
	) (schema.ParsedDocument, error)
}

// SchemaCatalog is the narrow setup-time capability supplied to a decoder that
// must canonicalize source documents through Artifact Store's schema registry.
type SchemaCatalog interface {
	ExpectedCanonicalizer

	Keys() []schema.Key
}
