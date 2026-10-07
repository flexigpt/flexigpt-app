package mcp

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managepackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/model"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain"
	policyMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/policy"
	mcppolicyv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcppolicy/contract/v1"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	"github.com/flexigpt/flexigpt-app/internal/mcppolicy"
)

func (a *Service) SaveMCPPolicy(
	ctx context.Context,
	request ManagedMCPPolicyUpsertRequest,
) (ManagedMCPPolicyUpsertResult, error) {
	if err := request.Plugin.Validate(); err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	if request.ExpectedPluginRevision == 0 {
		return ManagedMCPPolicyUpsertResult{}, fmt.Errorf(
			"%w: expected Plugin revision is required",
			spec.ErrInvalid,
		)
	}
	if err := request.Name.Validate(); err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	if a.protection.IsProtectedRoot(request.Plugin.RootID) {
		return ManagedMCPPolicyUpsertResult{}, fmt.Errorf(
			"%w: managed MCP Policy publication is not allowed in a protected Root",
			spec.ErrProtected,
		)
	}

	document, err := policyMCPDomain.DocumentFromPolicy(
		request.Name,
		request.Description,
		request.Policy,
	)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	definitionValue, err := policyMCPDomain.DefinitionForDocument(document)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	raw, err := document.CanonicalJSON()
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}

	address, err := a.support.PolicyPackage.Address(
		request.Name,
		"",
	)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}
	locator, err := a.support.PolicyPackage.Locator(address)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}

	membership, err := a.plugins.EnsureMemberForPluginSource(
		ctx,
		pluginAPI.EnsureMemberForPluginSourceRequest{
			Plugin:           request.Plugin,
			ExpectedRevision: request.ExpectedPluginRevision,
			Type:             mcppolicyv1.MCPPolicyType,
			Name:             request.Name,
			Locator:          locator,
		},
	)
	if err != nil {
		return ManagedMCPPolicyUpsertResult{}, err
	}

	result := ManagedMCPPolicyUpsertResult{
		Plugin:            membership.Plugin,
		MembershipCreated: membership.Created,
	}
	rootID := membership.Plugin.Artifact.RootID
	sourceID := membership.Plugin.Artifact.Binding.SourceID
	if _, err := a.plugins.EnsureManagedDeclarationDiscovery(
		ctx,
		rootID,
		sourceID,
		locator,
		a.support.PolicyPackage.Document.DecoderID,
	); err != nil {
		return result, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		managepackageModel.PublishRequest{
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
					Locator: a.support.PolicyPackage.Document.Locator,
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

func (a *Service) DeleteMCPPolicy(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
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
	address, err := a.support.PolicyPackage.AddressFromLocator(
		record.Binding.Locator,
	)
	if err != nil {
		return err
	}
	removeRequest := managepackageModel.RemoveRequest{
		RootID:           record.RootID,
		SourceID:         record.Binding.SourceID,
		Package:          address,
		ExpectedArtifact: &ref,
	}
	if a.support.PluginProfile.Source != nil &&
		sourceValue.StorageKey == a.support.PluginProfile.Source.StorageKey {
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

func (a *Service) policyBodyForResolvedArtifact(
	ctx context.Context,
	resolved resourceModel.ResolvedArtifact,
) (mcppolicy.MCPPolicy, error) {
	_ = ctx
	return policyMCPDomain.BodyFromDefinition(resolved.Definition)
}
