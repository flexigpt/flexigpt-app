package interpretation

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type schemaCodec struct {
	registration Registration
	jsonSchema   []byte
}

func NewSchemaCodec(
	registration Registration,
	jsonSchema []byte,
) (schema.Codec, error) {
	if err := registration.Validate(); err != nil {
		return nil, err
	}
	if len(jsonSchema) == 0 {
		return nil, fmt.Errorf(
			"%w: declaration schema is empty",
			spec.ErrInvalid,
		)
	}
	return &schemaCodec{
		registration: registration,
		jsonSchema:   append([]byte(nil), jsonSchema...),
	}, nil
}

func (c *schemaCodec) Key() schemaModel.Key {
	return c.registration.SchemaKey
}

func (c *schemaCodec) JSONSchema() []byte {
	return append([]byte(nil), c.jsonSchema...)
}

func (c *schemaCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schemaModel.ParsedDocument, error) {
	if err := declaration.ValidateSchemaHeader(
		c.registration.SchemaKey,
		raw,
	); err != nil {
		return schemaModel.ParsedDocument{}, err
	}

	// Generic expected-key schema execution has already completed before this
	// method is invoked. Family semantic interpretation occurs once in
	// Registry.DefinitionsForSchemaValidatedDocument.
	return schemaModel.ParsedDocument{
		Key:    c.registration.SchemaKey,
		Digest: cryptoutil.DigestBytes(raw),
		Raw:    append([]byte(nil), raw...),
	}, nil
}
