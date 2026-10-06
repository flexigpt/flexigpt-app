package schema

import (
	"context"
	"fmt"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// PassthroughCodec retains the canonical object already validated by Catalog.
// It performs no declaration-language dispatch or repeated normalization.
type PassthroughCodec struct {
	key        schemaModel.Key
	jsonSchema []byte
}

func NewPassthroughCodec(
	key schemaModel.Key,
	jsonSchema []byte,
) (*PassthroughCodec, error) {
	if err := key.Validate(); err != nil {
		return nil, err
	}
	if len(jsonSchema) == 0 || len(jsonSchema) > spec.MaxDefinitionBytes {
		return nil, fmt.Errorf(
			"%w: passthrough JSON Schema is empty or exceeds the size limit",
			spec.ErrInvalid,
		)
	}
	return &PassthroughCodec{
		key:        key,
		jsonSchema: append([]byte(nil), jsonSchema...),
	}, nil
}

func (c *PassthroughCodec) Key() schemaModel.Key {
	return c.key
}

func (c *PassthroughCodec) JSONSchema() []byte {
	return append([]byte(nil), c.jsonSchema...)
}

func (c *PassthroughCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schemaModel.ParsedDocument, error) {
	return schemaModel.ParsedDocument{
		Key:    c.key,
		Digest: cryptoutil.DigestBytes(raw),
		Raw:    append([]byte(nil), raw...),
	}, nil
}
