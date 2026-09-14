package providerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type PolicyCodec struct{}

func NewPolicyCodec() providerapi.SchemaCodec {
	return PolicyCodec{}
}

func (PolicyCodec) Key() schema.Key {
	return mcppolicyv1.MCPPolicySchemaKey
}

func (PolicyCodec) JSONSchema() []byte {
	return mcppolicyv1.MCPPolicyJSONSchema()
}

func (PolicyCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schema.ParsedDocument, error) {
	if ctx == nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: MCP Policy schema codec context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return schema.ParsedDocument{}, err
	}
	value, err := mcppolicyv1.DecodeMCPPolicyJSON(raw)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	canonical, err := value.CanonicalJSON()
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	return schema.ParsedDocument{
		Key:    mcppolicyv1.MCPPolicySchemaKey,
		Digest: cryptoutil.DigestBytes(canonical),
		Raw:    canonical,
	}, nil
}
