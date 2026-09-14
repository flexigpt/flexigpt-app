package codec

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// NewPassthrough creates a declaration SchemaCodec.
//
// Artifact Store's schema registry owns JSON canonicalization and JSON Schema
// execution. The codec receives known canonical JSON and returns it unchanged
// with the correct SchemaKey and document digest.
func NewPassthrough(
	key schema.Key,
	jsonSchema []byte,
) providerapi.SchemaCodec {
	return passthrough{
		key:        key,
		jsonSchema: append([]byte(nil), jsonSchema...),
	}
}

type passthrough struct {
	key        schema.Key
	jsonSchema []byte
}

func (c passthrough) Key() schema.Key {
	return c.key
}

func (c passthrough) JSONSchema() []byte {
	return append([]byte(nil), c.jsonSchema...)
}

func (c passthrough) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schema.ParsedDocument, error) {
	if ctx == nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: declaration schema codec context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return schema.ParsedDocument{}, err
	}
	if len(raw) == 0 {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: declaration schema codec received empty canonical JSON",
			basespec.ErrInvalid,
		)
	}
	return schema.ParsedDocument{
		Key:    c.key,
		Digest: cryptoutil.DigestBytes(raw),
		Raw:    append([]byte(nil), raw...),
	}, nil
}
