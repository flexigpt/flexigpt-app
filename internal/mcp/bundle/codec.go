package bundle

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type (
	BundleCodec struct{}
)

func NewBundleCodec() shareable.Codec {
	return BundleCodec{}
}

func (BundleCodec) JSONSchema() []byte {
	return append([]byte(nil), artifactbuiltin.BundleV1JSONSchema...)
}

func (BundleCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	if err := artifactbuiltin.CheckCodecContext(ctx); err != nil {
		return shareable.ParsedDocument{}, err
	}
	value, canonical, err := parseBundle(raw)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	return shareable.ParsedDocument{
		Key:    BundleCodec{}.Key(),
		Digest: value.Digest,
		Raw:    canonical,
	}, nil
}

func (BundleCodec) Key() shareable.SchemaKey {
	return artifactbuiltin.MCPSchemaKey
}

func parseBundle(
	raw []byte,
) (BundleDocument, json.RawMessage, error) {
	value, err := jsonutil.DecodeCanonicalObject[BundleDocument](raw, basespec.MaxDefinitionBytes)
	if err != nil {
		return BundleDocument{}, nil, err
	}
	return CanonicalizeBundle(value)
}
