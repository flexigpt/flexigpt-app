package policy

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type PolicyCodec struct{}

func NewPolicyCodec() providerapi.SchemaCodec {
	return PolicyCodec{}
}

func (PolicyCodec) JSONSchema() []byte {
	return append([]byte(nil), artifactbuiltin.PolicyV1JSONSchema...)
}

func (PolicyCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (providerapi.ParsedDocument, error) {
	if err := artifactbuiltin.CheckCodecContext(ctx); err != nil {
		return providerapi.ParsedDocument{}, err
	}
	value, canonical, err := parsePolicy(raw)
	if err != nil {
		return providerapi.ParsedDocument{}, err
	}
	return providerapi.ParsedDocument{
		Key:    PolicyCodec{}.Key(),
		Digest: value.Digest,
		Raw:    canonical,
	}, nil
}

func (PolicyCodec) Key() providerapi.SchemaKey {
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
