package schema

import (
	"context"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
)

// Codec supplies one published JSON Schema and its semantic canonicalization.
//
// Canonicalize receives an independently owned, bounded canonical JSON object
// that already satisfies this codec's published schema. It must not execute
// that same schema again. It may perform domain semantic validation or produce
// a different canonical representation.
//
// The catalog verifies returned key, canonical representation, and digest.
// Changed output is validated against the expected schema before admission.
type Codec interface {
	Key() schemaModel.Key
	JSONSchema() []byte

	Canonicalize(
		ctx context.Context,
		raw []byte,
	) (schemaModel.ParsedDocument, error)
}

// ExpectedCanonicalizer executes the schema identified by expected.
//
// It does not infer schema identity from document fields or declaration
// headers. Returned mutable data is independently owned by the caller.
type ExpectedCanonicalizer interface {
	CanonicalizeExpected(
		ctx context.Context,
		expected schemaModel.Key,
		raw []byte,
	) (schemaModel.ParsedDocument, error)
}

type API interface {
	ExpectedCanonicalizer
}

// KeyReader is an optional setup-time inspection capability. It permits an
// upper domain aggregate to verify that its selected registrations match the
// already assembled generic Store without exposing a provider implementation.
type KeyReader interface {
	Keys() []schemaModel.Key
}

// Catalog is setup-time schema access supplied to decoder registrations.
type Catalog interface {
	ExpectedCanonicalizer
	Keys() []schemaModel.Key
}

type Factory interface {
	NewCatalog(codecs ...Codec) (Catalog, error)
}

type FactoryFunc func(codecs ...Codec) (Catalog, error)

func (f FactoryFunc) NewCatalog(codecs ...Codec) (Catalog, error) {
	return f(codecs...)
}
