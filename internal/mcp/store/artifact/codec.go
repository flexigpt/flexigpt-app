package artifact

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type ServerCodec struct{}

func NewServerCodec() shareable.Codec {
	return ServerCodec{}
}

func (ServerCodec) JSONSchema() []byte {
	return append([]byte(nil), artifactbuiltin.ServerV1JSONSchema...)
}

func (ServerCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	if err := artifactbuiltin.CheckCodecContext(ctx); err != nil {
		return shareable.ParsedDocument{}, err
	}
	value, canonical, err := parseServer(raw)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	return shareable.ParsedDocument{
		Key:    ServerCodec{}.Key(),
		Digest: value.Digest,
		Raw:    canonical,
	}, nil
}

func (ServerCodec) Key() shareable.SchemaKey {
	return artifactbuiltin.MCPServerSchemaKey
}

func parseServer(
	raw []byte,
) (ServerDocument, json.RawMessage, error) {
	value, err := jsonutil.DecodeCanonicalObject[ServerDocument](raw, basespec.MaxDefinitionBytes)
	if err != nil {
		return ServerDocument{}, nil, err
	}
	return CanonicalizeServer(value)
}
