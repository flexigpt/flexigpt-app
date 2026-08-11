package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
)

type (
	BundleCodec struct{}
	ServerCodec struct{}
	PolicyCodec struct{}
)

func NewBundleCodec() shareable.Codec {
	return BundleCodec{}
}

func (BundleCodec) JSONSchema() []byte {
	return publishedJSONSchema(
		BundleSchemaURL,
		string(BundleKind),
		string(BundleSchemaID),
		[]string{
			"$schema",
			"kind",
			"schemaID",
			"schemaVersion",
			"logicalName",
			"mcpServers",
		},
		map[string]any{
			"mcpServers": map[string]any{
				"type": "object",
			},
			"bundleExtension": map[string]any{
				"type": "object",
			},
		},
	)
}

func (BundleCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	if err := checkCodecContext(ctx); err != nil {
		return shareable.ParsedDocument{}, err
	}
	value, canonical, err := ParseBundle(raw)
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
	return publishedJSONSchema(
		PolicySchemaURL,
		string(PolicyKind),
		string(PolicySchemaID),
		[]string{
			"$schema",
			"kind",
			"schemaID",
			"schemaVersion",
			"logicalName",
			"body",
		},
		map[string]any{
			"body": map[string]any{
				"type": "object",
			},
		},
	)
}

func (PolicyCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	if err := checkCodecContext(ctx); err != nil {
		return shareable.ParsedDocument{}, err
	}
	value, canonical, err := ParsePolicy(raw)
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
	return publishedJSONSchema(
		ServerSchemaURL,
		string(ServerKind),
		string(ServerSchemaID),
		[]string{
			"$schema",
			"kind",
			"schemaID",
			"schemaVersion",
			"logicalName",
			"mcpServer",
			"extension",
		},
		map[string]any{
			"mcpServer": map[string]any{
				"type": "object",
			},
			"extension": map[string]any{
				"type": "object",
			},
		},
	)
}

func (ServerCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	if err := checkCodecContext(ctx); err != nil {
		return shareable.ParsedDocument{}, err
	}
	value, canonical, err := ParseServer(raw)
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

func publishedJSONSchema(
	id string,
	kind string,
	schemaID string,
	required []string,
	additional map[string]any,
) []byte {
	properties := map[string]any{
		"$schema": map[string]any{
			"const": id,
		},
		"kind": map[string]any{
			"const": kind,
		},
		"schemaID": map[string]any{
			"const": schemaID,
		},
		"schemaVersion": map[string]any{
			"const": SchemaVersion,
		},
		"digest": map[string]any{
			"type":    "string",
			"pattern": "^sha256:[a-f0-9]{64}$",
		},
		"logicalName": map[string]any{
			"type":      "string",
			"minLength": 1,
			"maxLength": 256,
		},
		"logicalVersion": map[string]any{
			"type":      "string",
			"maxLength": 256,
		},
		"displayName": map[string]any{
			"type":      "string",
			"maxLength": 256,
		},
		"description": map[string]any{
			"type":      "string",
			"maxLength": basespec.MaxDescriptionBytes,
		},
		"labels": map[string]any{
			"type":                 "object",
			"additionalProperties": map[string]any{"type": "string"},
		},
	}
	maps.Copy(properties, additional)
	raw, err := json.Marshal(map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  id,
		"type":                 "object",
		"required":             required,
		"properties":           properties,
		"additionalProperties": false,
	})
	if err != nil {
		panic(err)
	}
	return raw
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
