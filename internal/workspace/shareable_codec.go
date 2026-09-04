package workspace

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type workspaceCollectionCodec struct{}

func NewCollectionCodec() providerapi.SchemaCodec {
	return workspaceCollectionCodec{}
}

func (workspaceCollectionCodec) Key() providerapi.SchemaKey {
	return artifactbuiltin.WorkspaceCollectionV1SchemaKey
}

func (workspaceCollectionCodec) JSONSchema() []byte {
	return artifactbuiltin.WorkspaceCollectionV1JSONSchema()
}

func (workspaceCollectionCodec) Canonicalize(
	ctx context.Context,
	raw []byte,
) (providerapi.ParsedDocument, error) {
	if ctx == nil {
		return providerapi.ParsedDocument{}, fmt.Errorf(
			"%w: workspace collection codec context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.ParsedDocument{}, err
	}

	value, err := artifactbuiltin.ParseWorkspaceCollectionV1(raw)
	if err != nil {
		return providerapi.ParsedDocument{}, err
	}
	canonical, err := artifactbuiltin.CanonicalizeWorkspaceCollectionV1(value)
	if err != nil {
		return providerapi.ParsedDocument{}, err
	}
	if canonical.Digest == nil {
		return providerapi.ParsedDocument{}, fmt.Errorf(
			"%w: canonical workspace collection has no digest",
			basespec.ErrInvalid,
		)
	}
	encoded, err := artifactbuiltin.MarshalWorkspaceCollectionV1(canonical)
	if err != nil {
		return providerapi.ParsedDocument{}, err
	}

	return providerapi.ParsedDocument{
		Key:    artifactbuiltin.WorkspaceCollectionV1SchemaKey,
		Digest: cryptoutil.Digest(*canonical.Digest),
		Raw:    json.RawMessage(encoded),
	}, nil
}
