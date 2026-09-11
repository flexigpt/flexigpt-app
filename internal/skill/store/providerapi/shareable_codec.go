package providerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/skillcollectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type skillCollectionCodec struct{}

func NewShareableCodec() providerapi.SchemaCodec {
	return skillCollectionCodec{}
}

func (c skillCollectionCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schema.ParsedDocument, error) {
	if ctx == nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: skill collection codec context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return schema.ParsedDocument{}, err
	}

	value, err := artifactbuiltin.ParseSkillCollectionV1(raw)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	canonical, err := artifactbuiltin.CanonicalizeSkillCollectionV1(value)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	encoded, err := artifactbuiltin.MarshalSkillCollectionV1(canonical)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	if canonical.Digest == nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: canonical skill collection has no digest",
			basespec.ErrInvalid,
		)
	}

	return schema.ParsedDocument{
		Key:    c.Key(),
		Digest: cryptoutil.Digest(*canonical.Digest),
		Raw:    json.RawMessage(encoded),
	}, nil
}

func (skillCollectionCodec) Key() schema.Key {
	return artifactbuiltin.SkillCollectionV1SchemaKey
}

func (skillCollectionCodec) JSONSchema() []byte {
	return skillcollectionv1.SkillCollectionJSONSchema()
}
