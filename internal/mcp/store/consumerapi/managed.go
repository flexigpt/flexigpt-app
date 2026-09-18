package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

func (a *API) CreateManagedMCP(
	ctx context.Context,
	request ManagedMCPCreateRequest,
) (ManagedMCPCreateResult, error) {
	if a == nil || a.collections == nil {
		return ManagedMCPCreateResult{}, basespec.ErrClosed
	}
	if err := request.Collection.Validate(); err != nil {
		return ManagedMCPCreateResult{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return ManagedMCPCreateResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if a.protection.IsProtectedRoot(request.Collection.RootID) {
		return ManagedMCPCreateResult{}, fmt.Errorf(
			"%w: managed MCP publication is not allowed in a protected Root",
			basespec.ErrProtected,
		)
	}

	definitionValue, err := mcpDomainServer.DefinitionForDocument(
		request.Document,
	)
	if err != nil {
		return ManagedMCPCreateResult{}, err
	}
	address, err := mcpDomain.ManagedPackageAddressForMCP(
		request.Document.LogicalName,
		request.Document.LogicalVersion,
	)
	if err != nil {
		return ManagedMCPCreateResult{}, err
	}
	locator, err := mcpDomain.ManagedPackageLocatorForMCP(address)
	if err != nil {
		return ManagedMCPCreateResult{}, err
	}

	member, err := a.collections.MemberForCollectionSource(
		ctx,
		request.Collection,
		declaration.TypeMCP,
		request.Document.LogicalName,
		locator,
	)
	if err != nil {
		return ManagedMCPCreateResult{}, err
	}
	membership, err := a.collections.EnsureMember(
		ctx,
		collection.AddMemberRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedCollectionRevision,
			Member:           member,
		},
	)
	if err != nil {
		return ManagedMCPCreateResult{}, err
	}

	result := ManagedMCPCreateResult{
		Collection:        membership.Collection,
		MembershipCreated: membership.Created,
	}
	rootID := membership.Collection.Artifact.RootID
	sourceID := membership.Collection.Artifact.Binding.SourceID
	if _, err := a.collections.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceID,
		locator,
		decoder.JSONDecoderID,
	); err != nil {
		return result, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: rootID,
			Binding: artifact.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			ExpectedKind:        mcpDomain.MCPArtifactKind,
			ExpectedLogicalName: request.Document.LogicalName,
			ExpectedDefinition:  definitionValue.Digest,
			Package: source.ManagedPackagePublication{
				Address: address,
				Files: []source.ManagedPackageFile{{
					Locator: mcpDomain.ManagedMCPDocumentFile,
					Content: append([]byte(nil), definitionValue.Body...),
				}},
			},
		},
	)
	if err != nil {
		return result, err
	}

	record := published.Artifact
	result.Artifact = record
	result.Address = record.Address()
	if record.Enabled != request.Enabled {
		record, err = a.artifacts.SetEnabled(
			ctx,
			record.Ref(),
			record.Revision,
			request.Enabled,
		)
		if err != nil {
			return result, err
		}
		result.Artifact = record
		result.Address = record.Address()
	}
	return result, nil
}

func (a *API) PurgeManagedMCP(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected MCP Artifact revision is required",
			basespec.ErrInvalid,
		)
	}

	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	if record.Kind != mcpDomain.MCPArtifactKind {
		return fmt.Errorf(
			"%w: Artifact is not an MCP Server",
			basespec.ErrUnsupported,
		)
	}
	if record.Revision != expectedRevision {
		return basespec.ErrConflict
	}
	if a.protection.IsProtectedRoot(record.RootID) {
		return fmt.Errorf(
			"%w: protected MCP Server deletion is not allowed",
			basespec.ErrProtected,
		)
	}
	if record.Binding.SubresourceLocator != "" {
		return fmt.Errorf(
			"%w: contained MCP declarations cannot be removed as managed MCP packages",
			basespec.ErrUnsupported,
		)
	}

	sourceValue, err := a.sources.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return err
	}
	if sourceValue.Kind != source.SourceKindManagedDirectory {
		return fmt.Errorf(
			"%w: MCP Server is not backed by a managed Source",
			basespec.ErrUnsupported,
		)
	}
	address, err := mcpDomain.ManagedPackageAddressFromMCPLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return err
	}

	removeRequest := artifact.RemoveArtifactRequest{
		RootID:           record.RootID,
		SourceID:         record.Binding.SourceID,
		Package:          address,
		ExpectedArtifact: &ref,
	}
	if sourceValue.StorageKey == collection.MCPManagedSourceStorageKey {
		locator := record.Binding.Locator
		removeRequest.PruneDiscoveryLocator = &locator
	}
	if err := a.managedArtifacts.Remove(ctx, removeRequest); err != nil {
		return err
	}
	missing, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	if missing.State != artifact.StateMissing {
		return fmt.Errorf(
			"%w: removed MCP Artifact is not missing",
			basespec.ErrConflict,
		)
	}
	installation, err := mcpDomainServer.DecodeServerData(missing.Data)
	if err != nil {
		return err
	}
	if err := a.cleanupServerSecretReferences(
		ctx,
		ref,
		installation,
	); err != nil {
		return err
	}

	return a.artifacts.Purge(ctx, ref, missing.Revision)
}
