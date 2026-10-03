package schema

import (
	"context"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
)

// Codec supplies one published JSON Schema and domain-specific semantic
// projection.
type Codec interface {
	Key() schemaModel.Key
	JSONSchema() []byte

	Canonicalize(
		ctx context.Context,
		raw []byte,
	) (schemaModel.ParsedDocument, error)
}

type EntityCanonicalizer interface {
	CanonicalizeEntity(
		ctx context.Context,
		entity schemaModel.EntityType,
		raw []byte,
	) (schemaModel.ParsedDocument, error)
}

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

// Catalog is the setup-time schema capability supplied to decoders.
type Catalog interface {
	ExpectedCanonicalizer

	Keys() []schemaModel.Key
}

type Factory interface {
	NewCatalog(codecs ...Codec) (Catalog, error)
}

type FactoryFunc func(codecs ...Codec) (Catalog, error)

func (f FactoryFunc) NewCatalog(
	codecs ...Codec,
) (Catalog, error) {
	return f(codecs...)
}
