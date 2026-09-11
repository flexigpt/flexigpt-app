package providerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/workspacecollectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type workspaceCollectionCodec struct{}

func NewCollectionCodec() providerapi.SchemaCodec {
	return workspaceCollectionCodec{}
}

func (workspaceCollectionCodec) Key() schema.Key {
	return workspacecollectionv1.WorkspaceCollectionSchemaKey
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

	value, err := jsonutil.DecodeJSONRaw[workspacecollectionv1.WorkspaceCollectionDocument](
		json.RawMessage(raw),
	)
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	canonical, err := value.Canonicalize()
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	digest, err := canonical.CalculatedDigest()
	if err != nil {
		return schema.ParsedDocument{}, err
	}
	digestValue := string(digest)
	canonical.Digest = &digestValue

	encoded, err := canonical.CanonicalJSON()
	if err != nil {
		return schema.ParsedDocument{}, err
	}

	return schema.ParsedDocument{
		Key:    workspacecollectionv1.WorkspaceCollectionSchemaKey,
		Digest: digest,
		Raw:    json.RawMessage(encoded),
	}, nil
}
