package store

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type (
	BundleCodec struct{}
)

func NewBundleCodec() providerapi.SchemaCodec {
	return BundleCodec{}
}

func (BundleCodec) JSONSchema() []byte {
	return append([]byte(nil), artifactbuiltin.BundleV1JSONSchema...)
}

func (BundleCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (providerapi.ParsedDocument, error) {
	if err := artifactbuiltin.CheckCodecContext(ctx); err != nil {
		return providerapi.ParsedDocument{}, err
	}
	value, canonical, err := parseBundle(raw)
	if err != nil {
		return providerapi.ParsedDocument{}, err
	}
	return providerapi.ParsedDocument{
		Key:    BundleCodec{}.Key(),
		Digest: value.Digest,
		Raw:    canonical,
	}, nil
}

func (BundleCodec) Key() providerapi.SchemaKey {
	return artifactbuiltin.MCPBundleSchemaKey
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
