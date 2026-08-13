package bundle

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
	"github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type EnsureBuiltInRequest struct {
	RootID         basespec.RootID
	CollectionID   basespec.CollectionID
	SourceID       basespec.SourceID
	PackageAddress source.ManagedPackageAddress

	PackageFiles  []source.ManagedPackageFile
	Document      shareable.ParsedDocument
	Registrations []Registration
}

// EnsureBuiltIn creates or verifies one protected MCP Bundle topology and
// installs its complete canonical document through the protected managed
// source path. It is only callable from trusted hydration composition.
func (a *API) EnsureBuiltIn(
	ctx context.Context,
	request EnsureBuiltInRequest,
) (Bundle, error) {
	if a == nil {
		return Bundle{}, basespec.ErrClosed
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return Bundle{}, err
	}
	if err := basespec.ValidateRootID(request.RootID); err != nil {
		return Bundle{}, err
	}
	if err := basespec.ValidateCollectionID(request.CollectionID); err != nil {
		return Bundle{}, err
	}
	if err := basespec.ValidateSourceID(request.SourceID); err != nil {
		return Bundle{}, err
	}
	if err := validateBundlePackageAddress(request.PackageAddress); err != nil {
		return Bundle{}, err
	}
	if !a.dependencies.RootPolicy.IsProtectedRoot(request.RootID) {
		return Bundle{}, fmt.Errorf(
			"%w: MCP built-in Root %q is not protected",
			basespec.ErrProtected,
			request.RootID,
		)
	}

	document, err := BundleFromParsedDocument(
		request.Document,
	)
	if err != nil {
		return Bundle{}, err
	}

	expectedAddress, err := PackageAddressForBundle(
		document.LogicalName,
		document.LogicalVersion,
	)
	if err != nil {
		return Bundle{}, err
	}
	if expectedAddress != request.PackageAddress {
		return Bundle{}, fmt.Errorf(
			"%w: MCP built-in package address differs from document identity",
			basespec.ErrConflict,
		)
	}

	sourceValue, err := a.dependencies.Sources.Get(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return Bundle{}, err
	}
	if sourceValue.Kind != managed.Kind || !sourceValue.Enabled {
		return Bundle{}, fmt.Errorf(
			"%w: protected MCP Bundle requires an enabled managed Source",
			basespec.ErrConflict,
		)
	}

	collectionData, err := EncodeCollectionData(CollectionData{
		SchemaVersion:           CollectionDataSchemaVersion,
		DiscoveryPolicyRevision: DiscoveryPolicyRevision,
		LogicalName:             document.LogicalName,
		LogicalVersion:          document.LogicalVersion,
		Labels:                  maps.Clone(document.Labels),
	})
	if err != nil {
		return Bundle{}, err
	}
	attachmentData, err := EncodeAttachmentData(AttachmentData{
		SchemaVersion:  AttachmentDataSchemaVersion,
		PackageAddress: request.PackageAddress,
	})
	if err != nil {
		return Bundle{}, err
	}

	created, _, err := a.dependencies.Collections.Create(
		ctx,
		request.RootID,
		collection.Draft{
			ID:          request.CollectionID,
			Kind:        schema.BundleKind,
			DisplayName: displayName(document),
			Description: document.Description,
			Enabled:     true,
			Data:        collectionData,
		},
		[]collection.AttachmentDraft{{
			SourceID: request.SourceID,
			Role:     RoleBuiltIn,
			Enabled:  true,
			Data:     attachmentData,
		}},
	)
	if err != nil {
		return Bundle{}, err
	}

	bundle, err := a.Get(ctx, created.Ref())
	if err != nil {
		return Bundle{}, err
	}
	if err := ensureBuiltInTopologyMatches(
		bundle,
		request,
		document,
		collectionData,
		attachmentData,
	); err != nil {
		return Bundle{}, err
	}

	updated, err := a.replaceCanonicalDocument(
		ctx,
		ReplaceDocumentRequest{
			Bundle:                     bundle.Collection.Ref(),
			ExpectedCollectionRevision: bundle.Collection.Revision,
			Document:                   request.Document.Raw,
			Registrations:              request.Registrations,
			AllowProtected:             true,
		},
		request.Document,
		request.PackageFiles,
	)
	if err != nil {
		return Bundle{}, err
	}
	return updated, nil
}

func ensureBuiltInTopologyMatches(
	bundle Bundle,
	request EnsureBuiltInRequest,
	document BundleDocument,
	collectionData json.RawMessage,
	attachmentData json.RawMessage,
) error {
	if bundle.Collection.RootID != request.RootID ||
		bundle.Collection.ID != request.CollectionID ||
		bundle.Collection.Kind != schema.BundleKind ||
		bundle.Collection.DisplayName != displayName(document) ||
		bundle.Collection.Description != document.Description ||
		!bundle.Collection.Enabled ||
		!jsonutil.Equal(bundle.Collection.Data, collectionData) {
		return fmt.Errorf(
			"%w: protected MCP Bundle differs from its static registry",
			basespec.ErrConflict,
		)
	}

	if bundle.Attachment.SourceID != request.SourceID ||
		bundle.Attachment.Role != RoleBuiltIn ||
		!bundle.Attachment.Enabled ||
		!jsonutil.Equal(bundle.Attachment.Data, attachmentData) ||
		bundle.PackageAddress != request.PackageAddress {
		return fmt.Errorf(
			"%w: protected MCP Bundle attachment differs from its static registry",
			basespec.ErrConflict,
		)
	}

	return nil
}

func isMCPKind(kind basespec.ArtifactKind) bool {
	return kind == schema.ServerKind || kind == schema.PolicyKind
}
