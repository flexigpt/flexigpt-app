package providerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/skillcollectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
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

	value, err := jsonutil.DecodeJSONRaw[skillcollectionv1.SkillCollectionDocument](
		json.RawMessage(raw),
	)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	canonical, err := value.Canonicalize()
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	digest, err := canonical.CalculatedDigest()
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	digestValue := string(digest)
	canonical.Digest = &digestValue

	encoded, err := canonical.CanonicalJSON()
	if err != nil {
		return schema.ParsedDocument{}, err
	}

	return schema.ParsedDocument{
		Key:    c.Key(),
		Digest: cryptoutil.Digest(*canonical.Digest),
		Raw:    json.RawMessage(encoded),
	}, nil
}

func (skillCollectionCodec) Key() schema.Key {
	return skillcollectionv1.SkillCollectionSchemaKey
}

func (skillCollectionCodec) JSONSchema() []byte {
	return skillcollectionv1.SkillCollectionJSONSchema()
}
