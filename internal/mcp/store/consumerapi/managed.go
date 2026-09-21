package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
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

	membership, err := a.collections.EnsureMemberForCollectionSource(
		ctx,
		collection.EnsureMemberForCollectionSourceRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedCollectionRevision,
			Type:             declaration.TypeMCP,
			Name:             request.Document.LogicalName,
			Locator:          locator,
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
	decoderID, err := documentTopology.DefaultDocumentDecoderID(
		documentTopology.DocumentUseManagedMCP,
	)
	if err != nil {
		return result, err
	}
	if _, err := a.collections.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceID,
		locator,
		decoderID,
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
					Locator: mcpDomain.ManagedMCPDocumentFile(),
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

// ReplaceManagedMCP replaces one complete managed MCP package while retaining
// its Artifact identity, managed package address, and direct Collection
// membership. Existing installation data is preserved only when it remains
// valid for the replacement server document.
func (a *API) ReplaceManagedMCP(
	ctx context.Context,
	request ManagedMCPReplaceRequest,
) (ManagedMCPReplaceResult, error) {
	if a == nil || a.collections == nil {
		return ManagedMCPReplaceResult{}, basespec.ErrClosed
	}
	if err := request.Collection.Validate(); err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if err := request.Artifact.Validate(); err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if request.Collection.RootID != request.Artifact.RootID {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Artifact belongs to another Root",
			basespec.ErrInvalid,
		)
	}
	if request.ExpectedCollectionRevision == 0 {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if request.ExpectedArtifactRevision == 0 {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: expected MCP Artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	if a.protection.IsProtectedRoot(request.Collection.RootID) {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: managed MCP replacement is not allowed in a protected Root",
			basespec.ErrProtected,
		)
	}

	collectionView, err := a.collections.Read(ctx, request.Collection)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if collectionView.Artifact.Revision != request.ExpectedCollectionRevision {
		return ManagedMCPReplaceResult{}, basespec.ErrConflict
	}
	if !collectionView.Editable {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Collection is read-only",
			basespec.ErrUnsupported,
		)
	}

	current, err := a.artifacts.Get(ctx, request.Artifact)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if current.Kind != mcpDomain.MCPArtifactKind {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: Artifact is not an MCP Server",
			basespec.ErrUnsupported,
		)
	}
	if current.Revision != request.ExpectedArtifactRevision {
		return ManagedMCPReplaceResult{}, basespec.ErrConflict
	}
	if current.Binding.SubresourceLocator != "" {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: contained MCP declarations cannot be replaced as managed MCP packages",
			basespec.ErrUnsupported,
		)
	}
	if current.Binding.SourceID != collectionView.Artifact.Binding.SourceID {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Server is not owned by this Collection Source",
			basespec.ErrUnsupported,
		)
	}

	sourceValue, err := a.sources.Get(
		ctx,
		current.RootID,
		current.Binding.SourceID,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if sourceValue.Kind != source.SourceKindManagedDirectory {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Server is not backed by a managed Source",
			basespec.ErrUnsupported,
		)
	}

	memberships, err := a.collections.ListMembershipsForArtifact(
		ctx,
		request.Artifact,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	memberFound := false
	for _, membership := range memberships {
		if membership.Collection != request.Collection ||
			!membership.ResolvedToArtifact {
			continue
		}
		memberFound = true
		break
	}
	if !memberFound {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Server is not a direct member of the requested Collection",
			basespec.ErrReferenceUnresolved,
		)
	}

	if request.Document.LogicalName != current.LogicalName {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: replacement MCP logical name must remain %q",
			basespec.ErrInvalid,
			current.LogicalName,
		)
	}

	definitionValue, err := mcpDomainServer.DefinitionForDocument(
		request.Document,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}

	currentAddress, err := mcpDomain.ManagedPackageAddressFromMCPLocator(
		current.Binding.Locator,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	requestedAddress, err := mcpDomain.ManagedPackageAddressForMCP(
		request.Document.LogicalName,
		request.Document.LogicalVersion,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if requestedAddress != currentAddress {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: replacement MCP cannot change managed package identity",
			basespec.ErrInvalid,
		)
	}

	currentInstallation, err := a.GetServerInstallation(
		ctx,
		request.Artifact,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if err := currentInstallation.Installation.ValidateFor(
		request.Artifact,
		request.Document,
	); err != nil {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: replacement MCP document is incompatible with current installation data: %w",
			basespec.ErrConflict,
			err,
		)
	}

	inspection, err := a.discovery.InspectSource(
		ctx,
		current.RootID,
		current.Binding.SourceID,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if !inspection.IsCurrent() {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: managed MCP Source requires refresh",
			basespec.ErrRefreshRequired,
		)
	}

	decoderID, err := documentTopology.DefaultDocumentDecoderID(
		documentTopology.DocumentUseManagedMCP,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if _, err := a.collections.EnsureManagedDeclarationDiscovery(
		ctx,
		current.RootID,
		current.Binding.SourceID,
		current.Binding.Locator,
		decoderID,
	); err != nil {
		return ManagedMCPReplaceResult{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: current.RootID,
			Binding: artifact.SourceBinding{
				SourceID: current.Binding.SourceID,
				Locator:  current.Binding.Locator,
			},
			ExpectedKind:        mcpDomain.MCPArtifactKind,
			ExpectedLogicalName: current.LogicalName,
			ExpectedDefinition:  definitionValue.Digest,
			Package: source.ManagedPackagePublication{
				Address:            currentAddress,
				ExpectedGeneration: inspection.State.SourceGeneration,
				Files: []source.ManagedPackageFile{{
					Locator: mcpDomain.ManagedMCPDocumentFile(),
					Content: append([]byte(nil), definitionValue.Body...),
				}},
			},
			AllowPackageReplacement: true,
		},
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if published.Artifact.Ref() != request.Artifact {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: replacement published another MCP Artifact",
			basespec.ErrConflict,
		)
	}

	updated := published.Artifact
	if updated.Enabled != request.Enabled {
		updated, err = a.artifacts.SetEnabled(
			ctx,
			updated.Ref(),
			updated.Revision,
			request.Enabled,
		)
		if err != nil {
			return ManagedMCPReplaceResult{}, err
		}
	}

	collectionView, err = a.collections.Read(ctx, request.Collection)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}

	return ManagedMCPReplaceResult{
		Artifact:   updated,
		Address:    updated.Address(),
		Collection: collectionView,
	}, nil
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
