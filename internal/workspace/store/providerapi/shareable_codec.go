package providerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/workspacecollectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type workspaceCollectionCodec struct{}

func NewCollectionCodec() providerapi.SchemaCodec {
	return workspaceCollectionCodec{}
}

func (workspaceCollectionCodec) Key() schema.Key {
	return artifactbuiltin.WorkspaceCollectionV1SchemaKey
}

func (workspaceCollectionCodec) JSONSchema() []byte {
	return workspacecollectionv1.WorkspaceCollectionJSONSchema()
}

func (workspaceCollectionCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (schema.ParsedDocument, error) {
	if ctx == nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: workspace collection codec context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return schema.ParsedDocument{}, err
	}

	value, err := artifactbuiltin.ParseWorkspaceCollectionV1(raw)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	canonical, err := artifactbuiltin.CanonicalizeWorkspaceCollectionV1(value)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	if canonical.Digest == nil {
		return schema.ParsedDocument{}, fmt.Errorf(
			"%w: canonical workspace collection has no digest",
			basespec.ErrInvalid,
		)
	}
	encoded, err := artifactbuiltin.MarshalWorkspaceCollectionV1(canonical)
	if err != nil {
		return schema.ParsedDocument{}, err
	}

	return schema.ParsedDocument{
		Key:    artifactbuiltin.WorkspaceCollectionV1SchemaKey,
		Digest: cryptoutil.Digest(*canonical.Digest),
		Raw:    json.RawMessage(encoded),
	}, nil
}
