package workspace

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type workspaceCollectionCodec struct{}

func NewShareableCodec() shareable.Codec {
	return workspaceCollectionCodec{}
}

func workspaceShareableSchemaKey() shareable.SchemaKey {
	return shareable.SchemaKey{
		Entity:        shareable.EntityCollection,
		Kind:          spec.CollectionKind,
		SchemaID:      spec.WorkspaceDescriptorSchemaID,
		SchemaVersion: spec.WorkspaceDescriptorSchemaVersion,
	}
}

func (workspaceCollectionCodec) Key() shareable.SchemaKey {
	return workspaceShareableSchemaKey()
}

func (workspaceCollectionCodec) JSONSchema() []byte {
	return builtinSchema.WorkspaceCollectionV1JSONSchema()
}

func (workspaceCollectionCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (shareable.ParsedDocument, error) {
	if ctx == nil {
		return shareable.ParsedDocument{}, fmt.Errorf(
			"%w: workspace collection codec context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return shareable.ParsedDocument{}, err
	}

	value, err := builtinSchema.ParseWorkspaceCollectionV1(raw)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	canonical, err := builtinSchema.CanonicalizeWorkspaceCollectionV1(value)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}
	if canonical.Digest == nil {
		return shareable.ParsedDocument{}, fmt.Errorf(
			"%w: canonical workspace collection has no digest",
			basespec.ErrInvalid,
		)
	}
	encoded, err := builtinSchema.MarshalWorkspaceCollectionV1(canonical)
	if err != nil {
		return shareable.ParsedDocument{}, err
	}

	return shareable.ParsedDocument{
		Key:    workspaceShareableSchemaKey(),
		Digest: cryptoutil.Digest(*canonical.Digest),
		Raw:    json.RawMessage(encoded),
	}, nil
}
