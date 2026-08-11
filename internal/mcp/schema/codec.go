package schema

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
)

//go:embed mcp-bundle-v1.schema.json
var bundleV1JSONSchema []byte

//go:embed mcp-server-v1.schema.json
var serverV1JSONSchema []byte

//go:embed mcp-policy-v1.schema.json
var policyV1JSONSchema []byte

type (
	BundleCodec struct{}
	ServerCodec struct{}
	PolicyCodec struct{}
)

func NewBundleCodec() shareable.Codec {
	return BundleCodec{}
}

func (BundleCodec) JSONSchema() []byte {
	return append([]byte(nil), bundleV1JSONSchema...)
}

func (BundleCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	if err := checkCodecContext(ctx); err != nil {
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
	return shareable.SchemaKey{
		Entity:        shareable.EntityCollection,
		Kind:          BundleKind,
		SchemaID:      BundleSchemaID,
		SchemaVersion: SchemaVersion,
	}
}

func NewPolicyCodec() shareable.Codec {
	return PolicyCodec{}
}

func (PolicyCodec) JSONSchema() []byte {
	return append([]byte(nil), policyV1JSONSchema...)
}

func (PolicyCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	if err := checkCodecContext(ctx); err != nil {
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
	return shareable.SchemaKey{
		Entity:        shareable.EntityArtifact,
		Kind:          basespec.CollectionKind(PolicyKind),
		SchemaID:      PolicySchemaID,
		SchemaVersion: SchemaVersion,
	}
}

func NewServerCodec() shareable.Codec {
	return ServerCodec{}
}

func (ServerCodec) JSONSchema() []byte {
	return append([]byte(nil), serverV1JSONSchema...)
}

func (ServerCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	if err := checkCodecContext(ctx); err != nil {
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
	return shareable.SchemaKey{
		Entity:        shareable.EntityArtifact,
		Kind:          basespec.CollectionKind(ServerKind),
		SchemaID:      ServerSchemaID,
		SchemaVersion: SchemaVersion,
	}
}

func checkCodecContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP schema codec context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}
