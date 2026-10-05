package codec

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type declarationCodec struct {
	key        schemaModel.Key
	jsonSchema []byte
}

func newDeclarationCodec(
	key schemaModel.Key,
	jsonSchema []byte,
) schema.Codec {
	return declarationCodec{
		key:        key,
		jsonSchema: append([]byte(nil), jsonSchema...),
	}
}

func (c declarationCodec) Key() schemaModel.Key {
	return c.key
}

func (c declarationCodec) JSONSchema() []byte {
	return append([]byte(nil), c.jsonSchema...)
}

func (c declarationCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schemaModel.ParsedDocument, error) {
	if err := ctx.Err(); err != nil {
		return schemaModel.ParsedDocument{}, err
	}
	if err := declaration.ValidateSchemaHeader(c.key, raw); err != nil {
		return schemaModel.ParsedDocument{}, err
	}
	return schemaModel.ParsedDocument{
		Key:    c.key,
		Digest: cryptoutil.DigestBytes(raw),
		Raw:    append([]byte(nil), raw...),
	}, nil
}
