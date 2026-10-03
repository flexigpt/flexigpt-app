package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managedpackageFlowModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managedpackage/model"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
)

func (a *API) SaveMCPPolicy(
	ctx context.Context,
	request ManagedMCPPolicyUpsertRequest,
) (ManagedMCPPolicyUpsertResult, error) {
	if a == nil {
		return ManagedMCPPolicyUpsertResult{}, spec.ErrClosed
	}
	if a.collections == nil {
		return ManagedMCPPolicyUpsertResult{}, spec.ErrClosed
	}
	if err := request.Collection.Validate(); err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return ManagedMCPPolicyUpsertResult{}, fmt.Errorf(
			"%w: expected Collection revision is required",
			spec.ErrInvalid,
		)
	}
	if err := request.Name.Validate(); err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	if a.protection.IsProtectedRoot(request.Collection.RootID) {
		return ManagedMCPPolicyUpsertResult{}, fmt.Errorf(
			"%w: managed MCP Policy publication is not allowed in a protected Root",
			spec.ErrProtected,
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

	address, err := managedpackageModel.NewManagedPackageAddress(
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
		managedpackageFlowModel.PublishRequest{
			RootID: rootID,
			Binding: artifactModel.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			ExpectedKind:        mcpDomain.MCPPolicyArtifactKind,
			ExpectedLogicalName: request.Name,
			ExpectedDefinition:  definitionValue.Digest,
			Package: managedpackageModel.ManagedPackagePublication{
				Address: address,
				Files: []managedpackageModel.ManagedPackageFile{{
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

func (a *API) DeleteMCPPolicy(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	if a == nil {
		return spec.ErrClosed
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected MCP Policy Artifact revision is required",
			spec.ErrInvalid,
		)
	}
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return err
	}
	if record.Kind != mcpDomain.MCPPolicyArtifactKind {
		return fmt.Errorf(
			"%w: Artifact is not an MCP Policy",
			spec.ErrUnsupported,
		)
	}
	if record.Revision != expectedRevision {
		return spec.ErrConflict
	}
	if a.protection.IsProtectedRoot(record.RootID) {
		return fmt.Errorf(
			"%w: protected MCP Policy deletion is not allowed",
			spec.ErrProtected,
		)
	}
	if record.Binding.SubresourceLocator != "" {
		return fmt.Errorf(
			"%w: contained MCP Policies cannot be removed as managed packages",
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
			"%w: MCP Policy is not backed by a managed Source",
			spec.ErrUnsupported,
		)
	}
	address, err := mcpDomain.ManagedPackageAddressFromMCPPolicyLocator(
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
			"%w: removed MCP Policy Artifact is not missing",
			spec.ErrConflict,
		)
	}
	return a.artifacts.Purge(ctx, ref, missing.Revision)
}

func (a *API) policyBodyForResolvedArtifact(
	ctx context.Context,
	resolved resourceModel.ResolvedArtifact,
) (mcpPolicy.MCPPolicy, error) {
	_ = ctx
	return mcpDomainPolicy.BodyFromDefinition(resolved.Definition)
}
