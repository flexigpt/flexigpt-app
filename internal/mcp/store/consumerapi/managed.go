package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managedpackageFlowModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managedpackage/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

func (a *API) CreateMCPServer(
	ctx context.Context,
	request ManagedMCPCreateRequest,
) (ManagedMCPCreateResult, error) {
	if a == nil || a.collections == nil {
		return ManagedMCPCreateResult{}, spec.ErrClosed
	}
	if err := request.Collection.Validate(); err != nil {
		return ManagedMCPCreateResult{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return ManagedMCPCreateResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			spec.ErrInvalid,
		)
	}
	if a.protection.IsProtectedRoot(request.Collection.RootID) {
		return ManagedMCPCreateResult{}, fmt.Errorf(
			"%w: managed MCP publication is not allowed in a protected Root",
			spec.ErrProtected,
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
		managedpackageFlowModel.PublishRequest{
			RootID: rootID,
			Binding: artifactModel.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			ExpectedKind:        mcpDomain.MCPArtifactKind,
			ExpectedLogicalName: request.Document.LogicalName,
			ExpectedDefinition:  definitionValue.Digest,
			Package: managedpackageModel.ManagedPackagePublication{
				Address: address,
				Files: []managedpackageModel.ManagedPackageFile{{
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

// UpdateMCPServer replaces one complete managed MCP package while retaining
// its Artifact identity, managed package address, and direct Collection
// membership. Existing installation data is preserved only when it remains
// valid for the replacement server document.
func (a *API) UpdateMCPServer(
	ctx context.Context,
	request ManagedMCPReplaceRequest,
) (ManagedMCPReplaceResult, error) {
	if a == nil || a.collections == nil {
		return ManagedMCPReplaceResult{}, spec.ErrClosed
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
			spec.ErrInvalid,
		)
	}
	if request.ExpectedCollectionRevision == 0 {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			spec.ErrInvalid,
		)
	}
	if request.ExpectedArtifactRevision == 0 {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: expected MCP Artifact revision is required",
			spec.ErrInvalid,
		)
	}
	if a.protection.IsProtectedRoot(request.Collection.RootID) {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: managed MCP replacement is not allowed in a protected Root",
			spec.ErrProtected,
		)
	}

	collectionView, err := a.collections.Read(ctx, request.Collection)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if collectionView.Artifact.Revision != request.ExpectedCollectionRevision {
		return ManagedMCPReplaceResult{}, spec.ErrConflict
	}
	if !collectionView.Editable &&
		!collectionView.Baseline {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Collection is read-only",
			spec.ErrUnsupported,
		)
	}

	current, err := a.artifacts.Get(ctx, request.Artifact)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if current.Kind != mcpDomain.MCPArtifactKind {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: Artifact is not an MCP Server",
			spec.ErrUnsupported,
		)
	}
	if current.Revision != request.ExpectedArtifactRevision {
		return ManagedMCPReplaceResult{}, spec.ErrConflict
	}
	if current.Binding.SubresourceLocator != "" {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: contained MCP declarations cannot be replaced as managed MCP packages",
			spec.ErrUnsupported,
		)
	}
	if current.Binding.SourceID != collectionView.Artifact.Binding.SourceID {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Server is not owned by this Collection Source",
			spec.ErrUnsupported,
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
	if sourceValue.Kind != managedfs.Kind {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Server is not backed by a managed Source",
			spec.ErrUnsupported,
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
			spec.ErrReferenceUnresolved,
		)
	}

	if request.Document.LogicalName != current.LogicalName {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: replacement MCP logical name must remain %q",
			spec.ErrInvalid,
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
			spec.ErrInvalid,
		)
	}

	currentMaterial, err := a.resolveServerMaterial(
		ctx,
		request.Artifact,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if err := currentMaterial.Installation.ValidateFor(
		request.Artifact,
		request.Document,
	); err != nil {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: replacement MCP document is incompatible with current installation data: %w",
			spec.ErrConflict,
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
			spec.ErrRefreshRequired,
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
		managedpackageFlowModel.PublishRequest{
			RootID: current.RootID,
			Binding: artifactModel.SourceBinding{
				SourceID: current.Binding.SourceID,
				Locator:  current.Binding.Locator,
			},
			ExpectedKind:        mcpDomain.MCPArtifactKind,
			ExpectedLogicalName: current.LogicalName,
			ExpectedDefinition:  definitionValue.Digest,
			Package: managedpackageModel.ManagedPackagePublication{
				Address:            currentAddress,
				ExpectedGeneration: inspection.State.SourceGeneration,
				Files: []managedpackageModel.ManagedPackageFile{{
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
			spec.ErrConflict,
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

func (a *API) DeleteMCPServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	if a == nil {
		return spec.ErrClosed
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected MCP Artifact revision is required",
			spec.ErrInvalid,
		)
	}

	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	if record.Kind != mcpDomain.MCPArtifactKind {
		return fmt.Errorf(
			"%w: Artifact is not an MCP Server",
			spec.ErrUnsupported,
		)
	}
	if record.Revision != expectedRevision {
		return spec.ErrConflict
	}
	if a.protection.IsProtectedRoot(record.RootID) {
		return fmt.Errorf(
			"%w: protected MCP Server deletion is not allowed",
			spec.ErrProtected,
		)
	}
	if record.Binding.SubresourceLocator != "" {
		return fmt.Errorf(
			"%w: contained MCP declarations cannot be removed as managed MCP packages",
			spec.ErrUnsupported,
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
	if sourceValue.Kind != managedfs.Kind {
		return fmt.Errorf(
			"%w: MCP Server is not backed by a managed Source",
			spec.ErrUnsupported,
		)
	}
	address, err := mcpDomain.ManagedPackageAddressFromMCPLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return err
	}

	removeRequest := managedpackageFlowModel.RemoveRequest{
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
	if missing.State != artifactModel.StateMissing {
		return fmt.Errorf(
			"%w: removed MCP Artifact is not missing",
			spec.ErrConflict,
		)
	}
	if err := a.overlays.PurgeServerLocalState(ctx, ref); err != nil {
		return err
	}
	return a.artifacts.Purge(ctx, ref, missing.Revision)
}
