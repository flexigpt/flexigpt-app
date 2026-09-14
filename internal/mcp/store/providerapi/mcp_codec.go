package providerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type MCPCodec struct{}

func NewMCPCodec() providerapi.SchemaCodec {
	return MCPCodec{}
}

func (MCPCodec) Key() schema.Key {
	return mcpv1.MCPSchemaKey
}

func (MCPCodec) JSONSchema() []byte {
	return mcpv1.MCPJSONSchema()
}

func (MCPCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schema.ParsedDocument, error) {
	if ctx == nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: MCP schema codec context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return schema.ParsedDocument{}, err
	}
	value, err := mcpv1.DecodeMCPJSON(raw)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	canonical, err := value.CanonicalJSON()
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	return schema.ParsedDocument{
		Key:    mcpv1.MCPSchemaKey,
		Digest: cryptoutil.DigestBytes(canonical),
		Raw:    canonical,
	}, nil
}
