package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
)

func (a *API) UpsertManagedMCPPolicy(
	ctx context.Context,
	request ManagedMCPPolicyUpsertRequest,
) (ManagedMCPPolicyUpsertResult, error) {
	if a == nil {
		return ManagedMCPPolicyUpsertResult{}, basespec.ErrClosed
	}
	if a.collections == nil {
		return ManagedMCPPolicyUpsertResult{}, basespec.ErrClosed
	}
	if err := request.Collection.Validate(); err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return ManagedMCPPolicyUpsertResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := request.Name.Validate(); err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	if a.protection.IsProtectedRoot(request.Collection.RootID) {
		return ManagedMCPPolicyUpsertResult{}, fmt.Errorf(
			"%w: managed MCP Policy publication is not allowed in a protected Root",
			basespec.ErrProtected,
		)
	}

	document, err := mcpDomainPolicy.DocumentFromPolicy(
		request.Name,
		request.Description,
		request.Policy,
	)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	definitionValue, err := mcpDomainPolicy.DefinitionForDocument(document)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	raw, err := document.CanonicalJSON()
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}

	address, err := source.NewManagedPackageAddress(
		mcpDomain.ManagedMCPPolicyPackageKind,
		request.Name,
		documentTopology.UnversionedPackageVersion(),
	)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	locator, err := address.FileLocator(
		mcpDomain.ManagedMCPPolicyDocumentFile(),
	)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}

	membership, err := a.collections.EnsureMemberForCollectionSource(
		ctx,
		collection.EnsureMemberForCollectionSourceRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedCollectionRevision,
			Type:             mcppolicyv1.MCPPolicyType,
			Name:             request.Name,
			Locator:          locator,
		},
	)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}

	result := ManagedMCPPolicyUpsertResult{
		Collection:        membership.Collection,
		MembershipCreated: membership.Created,
	}
	rootID := membership.Collection.Artifact.RootID
	sourceID := membership.Collection.Artifact.Binding.SourceID
	decoderID, err := documentTopology.DefaultDocumentDecoderID(
		documentTopology.DocumentUseManagedMCPPolicy,
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
			ExpectedKind:        mcpDomain.MCPPolicyArtifactKind,
			ExpectedLogicalName: request.Name,
			ExpectedDefinition:  definitionValue.Digest,
			Package: source.ManagedPackagePublication{
				Address: address,
				Files: []source.ManagedPackageFile{{
					Locator: mcpDomain.ManagedMCPPolicyDocumentFile(),
					Content: raw,
				}},
			},
			AllowPackageReplacement: true,
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

func (a *API) PurgeManagedMCPPolicy(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected MCP Policy Artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	if record.Kind != mcpDomain.MCPPolicyArtifactKind {
		return fmt.Errorf(
			"%w: Artifact is not an MCP Policy",
			basespec.ErrUnsupported,
		)
	}
	if record.Revision != expectedRevision {
		return basespec.ErrConflict
	}
	if a.protection.IsProtectedRoot(record.RootID) {
		return fmt.Errorf(
			"%w: protected MCP Policy deletion is not allowed",
			basespec.ErrProtected,
		)
	}
	if record.Binding.SubresourceLocator != "" {
		return fmt.Errorf(
			"%w: contained MCP Policies cannot be removed as managed packages",
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
			"%w: MCP Policy is not backed by a managed Source",
			basespec.ErrUnsupported,
		)
	}
	address, err := mcpDomain.ManagedPackageAddressFromMCPPolicyLocator(
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
			"%w: removed MCP Policy Artifact is not missing",
			basespec.ErrConflict,
		)
	}
	return a.artifacts.Purge(ctx, ref, missing.Revision)
}

func (a *API) policyBodyForResolvedArtifact(
	ctx context.Context,
	resolved resource.ResolvedArtifact,
) (mcpPolicy.MCPPolicy, error) {
	_ = ctx
	return mcpDomainPolicy.BodyFromDefinition(resolved.Definition)
}
