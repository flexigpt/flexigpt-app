package mcp

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	managepackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

func (a *Service) CreateMCPServer(
	ctx context.Context,
	request ManagedMCPCreateRequest,
) (ManagedMCPCreateResult, error) {
	if err := request.Plugin.Validate(); err != nil {
		return ManagedMCPCreateResult{}, err
	}
	if request.ExpectedPluginRevision == 0 {
		return ManagedMCPCreateResult{}, fmt.Errorf(
			"%w: expected Plugin revision is required",
			spec.ErrInvalid,
		)
	}
	if a.protection.IsProtectedRoot(request.Plugin.RootID) {
		return ManagedMCPCreateResult{}, fmt.Errorf(
			"%w: managed MCP publication is not allowed in a protected Root",
			spec.ErrProtected,
		)
	}

	definitionValue, err := serverMCPDomain.DefinitionForDocument(
		request.Document,
	)
	if err != nil {
		return ManagedMCPCreateResult{}, err
	}
	address, err := a.support.ServerPackage.Address(
		request.Document.LogicalName,
		request.Document.LogicalVersion,
	)
	if err != nil {
		return ManagedMCPCreateResult{}, err
	}
	locator, err := a.support.ServerPackage.Locator(address)
	if err != nil {
		return ManagedMCPCreateResult{}, err
	}

	membership, err := a.plugins.EnsureMemberForPluginSource(
		ctx,
		pluginAPI.EnsureMemberForPluginSourceRequest{
			Plugin:           request.Plugin,
			ExpectedRevision: request.ExpectedPluginRevision,
			Type:             declaration.TypeMCP,
			Name:             request.Document.LogicalName,
			Locator:          locator,
		},
	)
	if err != nil {
		return ManagedMCPCreateResult{}, err
	}

	result := ManagedMCPCreateResult{
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
		a.support.ServerPackage.Document.DecoderID,
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
			ExpectedKind:        mcpDomain.MCPArtifactKind,
			ExpectedLogicalName: request.Document.LogicalName,
			ExpectedDefinition:  definitionValue.Digest,
			Package: managedpackageModel.ManagedPackagePublication{
				Address: address,
				Files: []managedpackageModel.ManagedPackageFile{{
					Locator: a.support.ServerPackage.Document.Locator,
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
// its Artifact identity, managed package address, and direct Plugin
// membership. Existing installation data is preserved only when it remains
// valid for the replacement server document.
func (a *Service) UpdateMCPServer(
	ctx context.Context,
	request ManagedMCPReplaceRequest,
) (ManagedMCPReplaceResult, error) {
	if err := request.Plugin.Validate(); err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if err := request.Artifact.Validate(); err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if request.Plugin.RootID != request.Artifact.RootID {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Artifact belongs to another Root",
			spec.ErrInvalid,
		)
	}
	if request.ExpectedPluginRevision == 0 {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: expected Plugin revision is required",
			spec.ErrInvalid,
		)
	}
	if request.ExpectedArtifactRevision == 0 {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: expected MCP Artifact revision is required",
			spec.ErrInvalid,
		)
	}
	if a.protection.IsProtectedRoot(request.Plugin.RootID) {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: managed MCP replacement is not allowed in a protected Root",
			spec.ErrProtected,
		)
	}

	pluginView, err := a.plugins.Read(ctx, request.Plugin)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	if pluginView.Artifact.Revision != request.ExpectedPluginRevision {
		return ManagedMCPReplaceResult{}, spec.ErrConflict
	}
	if !pluginView.Editable &&
		!pluginView.Baseline {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Plugin is read-only",
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
	if current.Binding.SourceID != pluginView.Artifact.Binding.SourceID {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Server is not owned by this Plugin Source",
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

	memberships, err := a.plugins.ListMembershipsForArtifact(
		ctx,
		request.Artifact,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	memberFound := false
	for _, membership := range memberships {
		if membership.Plugin != request.Plugin ||
			!membership.ResolvedToArtifact {
			continue
		}
		memberFound = true
		break
	}
	if !memberFound {
		return ManagedMCPReplaceResult{}, fmt.Errorf(
			"%w: MCP Server is not a direct member of the requested Plugin",
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

	definitionValue, err := serverMCPDomain.DefinitionForDocument(
		request.Document,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}

	currentAddress, err := a.support.ServerPackage.AddressFromLocator(
		current.Binding.Locator,
	)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}
	requestedAddress, err := a.support.ServerPackage.Address(
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
	if _, err := a.plugins.EnsureManagedDeclarationDiscovery(
		ctx,
		current.RootID,
		current.Binding.SourceID,
		current.Binding.Locator,
		a.support.ServerPackage.Document.DecoderID,
	); err != nil {
		return ManagedMCPReplaceResult{}, err
	}

	published, err := a.managedArtifacts.Publish(
		ctx,
		managepackageModel.PublishRequest{
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
					Locator: a.support.ServerPackage.Document.Locator,
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

	pluginView, err = a.plugins.Read(ctx, request.Plugin)
	if err != nil {
		return ManagedMCPReplaceResult{}, err
	}

	return ManagedMCPReplaceResult{
		Artifact: updated,
		Address:  updated.Address(),
		Plugin:   pluginView,
	}, nil
}

func (a *Service) DeleteMCPServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
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
	address, err := a.support.ServerPackage.AddressFromLocator(
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
			"%w: removed MCP Artifact is not missing",
			spec.ErrConflict,
		)
	}
	if err := a.installation.Purge(ctx, ref); err != nil {
		return err
	}
	return a.artifacts.Purge(ctx, ref, missing.Revision)
}
