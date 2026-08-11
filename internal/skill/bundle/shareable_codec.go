package bundle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type skillCollectionCodec struct{}

func NewShareableCodec() shareable.Codec {
	return skillCollectionCodec{}
}

func (c skillCollectionCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	if ctx == nil {
		return shareable.ParsedDocument{}, fmt.Errorf(
			"%w: skill collection codec context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return shareable.ParsedDocument{}, err
	}

	value, err := builtinSchema.ParseSkillCollectionV1(raw)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	canonical, err := builtinSchema.CanonicalizeSkillCollectionV1(value)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	encoded, err := builtinSchema.MarshalSkillCollectionV1(canonical)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	if canonical.Digest == nil {
		return shareable.ParsedDocument{}, fmt.Errorf(
			"%w: canonical skill collection has no digest",
			basespec.ErrInvalid,
		)
	}

	return shareable.ParsedDocument{
		Key:    c.Key(),
		Digest: cryptoutil.Digest(*canonical.Digest),
		Raw:    json.RawMessage(encoded),
	}, nil
}

func (skillCollectionCodec) Key() shareable.SchemaKey {
	return shareable.SchemaKey{
		Entity:        shareable.EntityCollection,
		Kind:          CollectionKind,
		SchemaID:      basespec.SchemaID(builtinSchema.SkillCollectionV1SchemaID),
		SchemaVersion: builtinSchema.SkillCollectionV1SchemaVersion,
	}
}

func (skillCollectionCodec) JSONSchema() []byte {
	return builtinSchema.SkillCollectionV1JSONSchema()
}
