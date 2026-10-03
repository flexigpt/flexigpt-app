package jsonschema

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"

type Factory struct{}

func (Factory) NewCatalog(
	codecs ...schema.Codec,
) (schema.Catalog, error) {
	return NewRegistry(codecs...)
}

func NewFactory() schema.Factory {
	return Factory{}
}
