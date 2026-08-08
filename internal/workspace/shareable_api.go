package workspace

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

// GetShareableWorkspaceDocument returns the immutable shareable document
// explicitly bound to this local Workspace Collection.
//
// It does not read the mutable source-owned .flexigpt/workspace.json file.
func (a *API) GetShareableWorkspaceDocument(
	ctx context.Context,
	request *GetShareableWorkspaceDocumentRequest,
) (*GetShareableWorkspaceDocumentResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, invalidAPIRequest(
			"shareable Workspace document request is required",
		)
	}
	if _, err := a.workspace.service.Get(ctx, request.Workspace); err != nil {
		return nil, err
	}

	document, err := a.dependencies.Shareables.GetCollection(
		ctx,
		request.Workspace,
	)
	if err != nil {
		return nil, err
	}
	view, err := workspaceShareableDocumentViewOf(document)
	if err != nil {
		return nil, err
	}
	return &GetShareableWorkspaceDocumentResponse{Body: &view}, nil
}

// StoreShareableWorkspaceDocument validates and binds an initial immutable
// workspace.collection document to an existing local Workspace Collection.
//
// This operation is intentionally explicit. Refreshing a Workspace only reads
// its mutable source-owned workspace.json descriptor and must never overwrite
// an immutable imported or exported shareable-document binding.
func (a *API) StoreShareableWorkspaceDocument(
	ctx context.Context,
	request *StoreShareableWorkspaceDocumentRequest,
) (*StoreShareableWorkspaceDocumentResponse, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if request == nil || request.Body == nil {
		return nil, invalidAPIRequest(
			"shareable Workspace document body is required",
		)
	}
	if len(request.Body.Document) == 0 {
		return nil, invalidAPIRequest(
			"shareable Workspace document is required",
		)
	}
	if _, err := a.workspace.service.Get(ctx, request.Workspace); err != nil {
		return nil, err
	}

	parsed, err := a.dependencies.Shareables.Canonicalize(
		ctx,
		request.Body.Document,
	)
	if err != nil {
		return nil, err
	}
	if parsed.Key != workspaceShareableSchemaKey() {
		return nil, fmt.Errorf(
			"%w: shareable document schema does not identify a Workspace collection",
			spec.ErrInvalidWorkspace,
		)
	}

	document, err := a.dependencies.Shareables.StoreCollection(
		ctx,
		request.Workspace,
		parsed.Raw,
	)
	if err != nil {
		return nil, err
	}
	view, err := workspaceShareableDocumentViewOf(document)
	if err != nil {
		return nil, err
	}
	return &StoreShareableWorkspaceDocumentResponse{Body: &view}, nil
}

func workspaceShareableDocumentViewOf(
	document shareable.CollectionDocument,
) (WorkspaceShareableDocumentView, error) {
	if document.Binding.Key != workspaceShareableSchemaKey() {
		return WorkspaceShareableDocumentView{}, fmt.Errorf(
			"%w: stored shareable document schema does not identify a Workspace collection",
			spec.ErrInvalidWorkspace,
		)
	}
	if _, err := builtinSchema.ParseWorkspaceCollectionV1(document.Raw); err != nil {
		return WorkspaceShareableDocumentView{}, fmt.Errorf(
			"%w: stored Workspace shareable document is invalid: %w",
			spec.ErrInvalidWorkspace,
			err,
		)
	}
	return WorkspaceShareableDocumentView{
		Workspace: document.Binding.Collection,
		Schema:    document.Binding.Key,
		Digest:    document.Binding.Digest,
		Document:  append([]byte(nil), document.Raw...),
	}, nil
}
