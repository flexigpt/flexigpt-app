package providerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/skillv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type skillCodec struct{}

func NewShareableCodec() providerapi.SchemaCodec {
	return skillCodec{}
}

func (skillCodec) Key() schema.Key {
	return skillv1.SkillSchemaKey
}

func (skillCodec) JSONSchema() []byte {
	return skillv1.SkillJSONSchema()
}

func (skillCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schema.ParsedDocument, error) {
	if ctx == nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: Skill schema codec context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return schema.ParsedDocument{}, err
	}

	value, err := skillv1.DecodeSkillJSON(raw)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	canonical, err := value.CanonicalJSON()
	if err != nil {
		return schema.ParsedDocument{}, err
	}

	return schema.ParsedDocument{
		Key:    skillv1.SkillSchemaKey,
		Digest: cryptoutil.DigestBytes(canonical),
		Raw:    canonical,
	}, nil
}
