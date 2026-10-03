package schema

import (
	"context"
	"fmt"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// PassthroughCodec is useful for schema-backed documents whose canonical
// representation is the validated canonical JSON object itself.
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
	if len(jsonSchema) == 0 ||
		len(jsonSchema) > spec.MaxDefinitionBytes {
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
	if c == nil {
		return schemaModel.Key{}
	}
	return c.key
}

func (c *PassthroughCodec) JSONSchema() []byte {
	if c == nil {
		return nil
	}
	return append([]byte(nil), c.jsonSchema...)
}

func (c *PassthroughCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schemaModel.ParsedDocument, error) {
	if c == nil {
		return schemaModel.ParsedDocument{}, spec.ErrClosed
	}
	if ctx == nil {
		return schemaModel.ParsedDocument{}, fmt.Errorf(
			"%w: passthrough schema context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return schemaModel.ParsedDocument{}, err
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxDefinitionBytes,
	)
	if err != nil {
		return schemaModel.ParsedDocument{}, err
	}

	return schemaModel.ParsedDocument{
		Key:    c.key,
		Digest: cryptoutil.DigestBytes(canonical),
		Raw:    canonical,
	}, nil
}
