// Package provider defines Artifact Store's stable inbound provider
// contracts.
//
// Providers may depend on this package and basespec only from artifactory-go.
package provider

import (
	"context"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
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
	Key() schemaModel.Key
	JSONSchema() []byte

	Canonicalize(
		ctx context.Context,
		raw []byte,
	) (schemaModel.ParsedDocument, error)
}

// EntityCanonicalizer supports dispatch by an entity type inferred from a
// caller-owned context.
type EntityCanonicalizer interface {
	CanonicalizeEntity(
		ctx context.Context,
		entity schemaModel.EntityType,
		raw []byte,
	) (schemaModel.ParsedDocument, error)
}

// ExpectedCanonicalizer is the narrow Artifact Store capability required by a
// provider that accepts a document with a known schema identity.
type ExpectedCanonicalizer interface {
	CanonicalizeExpected(
		ctx context.Context,
		expected schemaModel.Key,
		raw []byte,
	) (schemaModel.ParsedDocument, error)
}

// SchemaCatalog is the narrow setup-time capability supplied to a decoder that
// must canonicalize source documents through Artifact Store's schema registry.
type SchemaCatalog interface {
	ExpectedCanonicalizer

	Keys() []schemaModel.Key
}
