package providerapi

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
)

type PolicyCodec struct{}

func NewPolicyCodec() providerapi.SchemaCodec {
	return PolicyCodec{}
}

func (PolicyCodec) JSONSchema() []byte {
	return mcppolicyv1.MCPPolicyJSONSchema()
}

func (PolicyCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schema.ParsedDocument, error) {
	value, canonical, err := parsePolicy(raw)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	return schema.ParsedDocument{
		Key:    PolicyCodec{}.Key(),
		Digest: value.Digest,
		Raw:    canonical,
	}, nil
}

func (PolicyCodec) Key() schema.Key {
	return mcppolicyv1.MCPPolicySchemaKey
}

func parsePolicy(
	raw []byte,
) (mcpDomainPolicy.PolicyDocument, json.RawMessage, error) {
	value, err := jsonutil.DecodeJSONRaw[mcpDomainPolicy.PolicyDocument](json.RawMessage(raw))
	if err != nil {
		return mcpDomainPolicy.PolicyDocument{}, nil, err
	}
	return mcpDomainPolicy.CanonicalizePolicy(value)
}
