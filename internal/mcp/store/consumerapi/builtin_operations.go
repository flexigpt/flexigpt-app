package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpbundlev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpserverv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpDomainBundle "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/bundle"
)

type EnsureBuiltInRequest struct {
	RootID         root.RootID
	CollectionID   collection.CollectionID
	SourceID       source.SourceID
	PackageAddress source.ManagedPackageAddress

	PackageFiles  []source.ManagedPackageFile
	Document      schema.ParsedDocument
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
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return Bundle{}, err
	}
	if err := request.RootID.Validate(); err != nil {
		return Bundle{}, err
	}
	if err := request.CollectionID.Validate(); err != nil {
		return Bundle{}, err
	}
	if err := request.SourceID.Validate(); err != nil {
		return Bundle{}, err
	}
	if _, err := mcpDomainBundle.DocumentLocatorForPackage(request.PackageAddress); err != nil {
		return Bundle{}, err
	}
	if !a.protection.IsProtectedRoot(request.RootID) {
		return Bundle{}, fmt.Errorf(
			"%w: MCP built-in Root %q is not protected",
			basespec.ErrProtected,
			request.RootID,
		)
	}

	document, err := mcpDomainBundle.BundleFromParsedDocument(
		request.Document,
	)
	if err != nil {
		return Bundle{}, err
	}

	expectedAddress, err := mcpDomainBundle.PackageAddressForBundle(
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

	sourceValue, err := a.sources.Get(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return Bundle{}, err
	}
	if sourceValue.Kind != source.SourceKindManagedDirectory ||
		!sourceValue.Enabled {
		return Bundle{}, fmt.Errorf(
			"%w: protected MCP Bundle requires an enabled managed Source",
			basespec.ErrConflict,
		)
	}

	collectionData, err := mcpDomainBundle.EncodeCollectionData(mcpDomainBundle.CollectionData{
		SchemaVersion:           artifactbuiltin.MCPSchemaVersion,
		DiscoveryPolicyRevision: artifactbuiltin.DecoderRevision,
		LogicalName:             document.LogicalName,
		LogicalVersion:          document.LogicalVersion,
		Labels:                  maps.Clone(document.Labels),
	})
	if err != nil {
		return Bundle{}, err
	}
	attachmentData, err := mcpDomainBundle.EncodeAttachmentData(mcpDomainBundle.AttachmentData{
		SchemaVersion:  artifactbuiltin.MCPSchemaVersion,
		PackageAddress: request.PackageAddress,
	})
	if err != nil {
		return Bundle{}, err
	}

	created, _, err := a.collections.Create(
		ctx,
		request.RootID,
		collection.Draft{
			ID:          request.CollectionID,
			Kind:        mcpbundlev1.MCPBundleKind,
			DisplayName: displayName(document),
			Description: document.Description,
			Enabled:     true,
			Data:        collectionData,
		},
		[]collection.AttachmentDraft{{
			SourceID: request.SourceID,
			Role:     artifactbuiltin.BuiltInAttachmentRole,
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
		document,
		request.Document.Raw,
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
	document mcpDomainBundle.BundleDocument,
	collectionData json.RawMessage,
	attachmentData json.RawMessage,
) error {
	if bundle.Collection.RootID != request.RootID ||
		bundle.Collection.ID != request.CollectionID ||
		bundle.Collection.Kind != mcpbundlev1.MCPBundleKind ||
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
		bundle.Attachment.Role != artifactbuiltin.BuiltInAttachmentRole ||
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

func isMCPKind(kind artifact.ArtifactKind) bool {
	return kind == mcpserverv1.MCPServerKind ||
		kind == mcppolicyv1.MCPPolicyKind
}
