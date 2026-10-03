package codec

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// NewPassthrough creates a declaration SchemaCodec.
//
// Artifact Store's schema registry owns JSON canonicalization and JSON Schema
// execution. The codec receives known canonical JSON and returns it unchanged
// with the correct SchemaKey and document digest.
func NewPassthrough(
	key schemaModel.Key,
	jsonSchema []byte,
) provider.SchemaCodec {
	return passthrough{
		key:        key,
		jsonSchema: append([]byte(nil), jsonSchema...),
	}
}

type passthrough struct {
	key        schemaModel.Key
	jsonSchema []byte
}

func (c passthrough) Key() schemaModel.Key {
	return c.key
}

func (c passthrough) JSONSchema() []byte {
	return append([]byte(nil), c.jsonSchema...)
}

func (c passthrough) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schemaModel.ParsedDocument, error) {
	if ctx == nil {
		return schemaModel.ParsedDocument{}, fmt.Errorf(
			"%w: declaration schema codec context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return schemaModel.ParsedDocument{}, err
	}
	if len(raw) == 0 {
		return schemaModel.ParsedDocument{}, fmt.Errorf(
			"%w: declaration schema codec received empty canonical JSON",
			spec.ErrInvalid,
		)
	}
	return schemaModel.ParsedDocument{
		Key:    c.key,
		Digest: cryptoutil.DigestBytes(raw),
		Raw:    append([]byte(nil), raw...),
	}, nil
}
