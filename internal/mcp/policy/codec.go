package policy

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type PolicyCodec struct{}

func NewPolicyCodec() shareable.Codec {
	return PolicyCodec{}
}

func (PolicyCodec) JSONSchema() []byte {
	return append([]byte(nil), artifactbuiltin.PolicyV1JSONSchema...)
}

func (PolicyCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	if err := artifactbuiltin.CheckCodecContext(ctx); err != nil {
		return shareable.ParsedDocument{}, err
	}
	value, canonical, err := parsePolicy(raw)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	return shareable.ParsedDocument{
		Key:    PolicyCodec{}.Key(),
		Digest: value.Digest,
		Raw:    canonical,
	}, nil
}

func (PolicyCodec) Key() shareable.SchemaKey {
	return artifactbuiltin.MCPPolicySchemaKey
}

func parsePolicy(
	raw []byte,
) (PolicyDocument, json.RawMessage, error) {
	value, err := jsonutil.DecodeCanonicalObject[PolicyDocument](raw, basespec.MaxDefinitionBytes)
	if err != nil {
		return PolicyDocument{}, nil, err
	}
	return CanonicalizePolicy(value)
}
