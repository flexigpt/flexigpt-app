package bundle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
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
) (providerapi.ParsedDocument, error) {
	if ctx == nil {
		return providerapi.ParsedDocument{}, fmt.Errorf(
			"%w: skill collection codec context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.ParsedDocument{}, err
	}

	value, err := artifactbuiltin.ParseSkillCollectionV1(raw)
	if err != nil {
		return providerapi.ParsedDocument{}, err
	}
	canonical, err := artifactbuiltin.CanonicalizeSkillCollectionV1(value)
	if err != nil {
		return providerapi.ParsedDocument{}, err
	}
	encoded, err := artifactbuiltin.MarshalSkillCollectionV1(canonical)
	if err != nil {
		return providerapi.ParsedDocument{}, err
	}
	if canonical.Digest == nil {
		return providerapi.ParsedDocument{}, fmt.Errorf(
			"%w: canonical skill collection has no digest",
			basespec.ErrInvalid,
		)
	}

	return providerapi.ParsedDocument{
		Key:    c.Key(),
		Digest: cryptoutil.Digest(*canonical.Digest),
		Raw:    json.RawMessage(encoded),
	}, nil
}

func (skillCollectionCodec) Key() providerapi.SchemaKey {
	return artifactbuiltin.SkillCollectionV1SchemaKey
}

func (skillCollectionCodec) JSONSchema() []byte {
	return artifactbuiltin.SkillCollectionV1JSONSchema()
}
