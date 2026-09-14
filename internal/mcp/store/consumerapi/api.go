package consumerapi

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
)

type API struct {
	sources          compositionapi.SourceAPI
	discovery        compositionapi.DiscoveryAPI
	artifacts        compositionapi.ArtifactAPI
	resources        compositionapi.ResourceAPI
	managedArtifacts compositionapi.ManagedArtifactAPI
	protection       compositionapi.ProtectionAPI

	overlays       mcpOverlay.OverlayRepository
	secretCleaner  mcpDomainServer.SecretCleaner
	baselinePolicy mcpPolicy.MCPPolicy
}

func New(
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	overlays mcpOverlay.OverlayRepository,
	secretCleaner mcpDomainServer.SecretCleaner,
	baselinePolicy mcpPolicy.MCPPolicy,
) (*API, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil ||
		secretCleaner == nil {
		return nil, fmt.Errorf(
			"%w: MCP Store dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if err := baselinePolicy.Validate(); err != nil {
		return nil, err
	}
	return &API{
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		resources:        resources,
		managedArtifacts: managedArtifacts,
		protection:       protection,
		overlays:         overlays,
		secretCleaner:    secretCleaner,
		baselinePolicy:   baselinePolicy,
	}, nil
}

func (a *API) ListServers(
	ctx context.Context,
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	return a.listArtifacts(ctx, rootID, mcpDomain.MCPArtifactKind)
}

func (a *API) ListPolicies(
	ctx context.Context,
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	return a.listArtifacts(ctx, rootID, mcpDomain.MCPPolicyArtifactKind)
}

func (a *API) GetServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ServerInstallationView, error) {
	material, err := a.resolveServerMaterial(ctx, ref, false)
	if err != nil {
		return ServerInstallationView{}, err
	}
	return ServerInstallationView{
		Artifact:             material.Resource.Artifact.Clone(),
		Definition:           material.Resource.Definition.Clone(),
		Document:             material.Document,
		Installation:         material.Installation,
		InstallationRevision: material.InstallationRevision,
		InstallationEnabled:  material.InstallationEnabled,
		RuntimeEnabled:       material.RuntimeEnabled,
		BuiltIn:              material.BuiltIn,
	}, nil
}

func (a *API) InspectMCPPolicyForRuntime(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (PolicyView, error) {
	if a == nil {
		return PolicyView{}, basespec.ErrClosed
	}
	resolved, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resource.ResolveOptions{},
	)
	if err != nil {
		return PolicyView{}, err
	}
	if resolved.Artifact.Kind != mcpDomain.MCPPolicyArtifactKind {
		return PolicyView{}, fmt.Errorf(
			"%w: Artifact is not an MCP Policy",
			basespec.ErrReferenceUnresolved,
		)
	}
	body, err := mcpDomainPolicy.BodyFromDefinition(
		resolved.Definition,
	)
	if err != nil {
		return PolicyView{}, err
	}
	return PolicyView{
		Artifact:         resolved.Artifact.Clone(),
		Definition:       resolved.Definition.Clone(),
		Body:             body,
		EffectiveEnabled: resolved.Artifact.Enabled,
		BuiltIn: a.protection.IsProtectedRoot(
			resolved.Artifact.RootID,
		),
	}, nil
}

func (a *API) ResolveMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpDomainServer.Resolved, error) {
	return a.resolveMCPServer(ctx, ref, true)
}

func (a *API) InspectMCPServerForRuntime(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpDomainServer.Resolved, error) {
	return a.resolveMCPServer(ctx, ref, false)
}

func (a *API) UpdateServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpDomainServer.ServerData,
) (artifact.Artifact, error) {
	if expectedArtifactRevision == 0 {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: expected MCP Server Artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	material, err := a.resolveServerMaterial(ctx, ref, false)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if material.BuiltIn {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: protected MCP Server installation belongs in an overlay",
			basespec.ErrProtected,
		)
	}
	if material.Resource.Artifact.Revision != expectedArtifactRevision {
		return artifact.Artifact{}, basespec.ErrConflict
	}
	if err := data.ValidateFor(ref, material.Document); err != nil {
		return artifact.Artifact{}, err
	}
	encoded, err := mcpDomainServer.EncodeServerData(data)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if jsonutil.Equal(material.Resource.Artifact.Data, encoded) {
		return material.Resource.Artifact, nil
	}
	updated, err := a.artifacts.UpdateData(
		ctx,
		ref,
		expectedArtifactRevision,
		encoded,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if err := mcpDomainServer.CleanupUnboundServerSecrets(
		ctx,
		updated.Ref(),
		material.Document,
		data,
		a.secretCleaner,
	); err != nil {
		return updated, fmt.Errorf(
			"MCP server secret cleanup remains pending: %w",
			err,
		)
	}
	return updated, nil
}

func (a *API) UpdateProtectedServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
	data mcpDomainServer.ServerData,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if !a.protection.IsProtectedRoot(ref.RootID) {
		return fmt.Errorf(
			"%w: MCP Server is not in a protected Root",
			basespec.ErrProtected,
		)
	}
	if a.overlays == nil {
		return fmt.Errorf(
			"%w: MCP overlay store is unavailable",
			basespec.ErrReferenceUnresolved,
		)
	}
	material, err := a.resolveServerMaterial(ctx, ref, false)
	if err != nil {
		return err
	}
	if !material.BuiltIn {
		return fmt.Errorf(
			"%w: MCP Server is not a protected Artifact",
			basespec.ErrProtected,
		)
	}
	if err := data.ValidateFor(ref, material.Document); err != nil {
		return err
	}

	current, found, err := a.overlays.GetServerOverlay(ctx, ref)
	if err != nil {
		return err
	}
	if found && current.Revision != expectedOverlayRevision {
		return basespec.ErrConflict
	}
	if !found && expectedOverlayRevision != 0 {
		return basespec.ErrConflict
	}
	nextRevision := uint64(1)
	if found {
		nextRevision = current.Revision + 1
	}
	next := mcpOverlay.ServerOverlay{
		SchemaVersion:  mcpDomain.InstallationDataSchemaVersion,
		Revision:       nextRevision,
		RuntimeEnabled: runtimeEnabled,
		ServerData:     data,
	}
	if err := a.overlays.PutServerOverlay(
		ctx,
		ref,
		expectedOverlayRevision,
		next,
	); err != nil {
		return err
	}
	return mcpDomainServer.CleanupUnboundServerSecrets(
		ctx,
		ref,
		material.Document,
		data,
		a.secretCleaner,
	)
}

func (a *API) EnsureBuiltInSourceCurrent(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	inspection, err := a.discovery.InspectSource(ctx, rootID, sourceID)
	if errors.Is(err, basespec.ErrRefreshStateNotFound) {
		_, err = a.discovery.RefreshSource(ctx, rootID, sourceID)
		return err
	}
	if err != nil {
		return err
	}
	if inspection.IsCurrent() {
		return nil
	}
	_, err = a.discovery.RefreshSource(ctx, rootID, sourceID)
	return err
}

func (a *API) InstallBuiltInPackage(
	ctx context.Context,
	request BuiltInPackageInstallRequest,
) ([]artifact.Artifact, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return nil, err
	}
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}
	if err := request.SourceID.Validate(); err != nil {
		return nil, err
	}
	if err := request.PackageAddress.Validate(); err != nil {
		return nil, err
	}
	if err := request.DocumentFile.ValidatePortable(false); err != nil {
		return nil, err
	}
	if len(request.Expectations) == 0 {
		return nil, fmt.Errorf(
			"%w: built-in MCP package has no expected Artifacts",
			basespec.ErrInvalid,
		)
	}
	if !a.protection.IsProtectedRoot(request.RootID) {
		return nil, fmt.Errorf(
			"%w: MCP built-in Root is not protected",
			basespec.ErrProtected,
		)
	}

	sourceValue, err := a.sources.Get(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return nil, err
	}
	if sourceValue.Kind != source.SourceKindManagedDirectory {
		return nil, fmt.Errorf(
			"%w: MCP built-in Source must be managed",
			basespec.ErrInvalid,
		)
	}
	documentLocator, err := request.PackageAddress.FileLocator(
		request.DocumentFile,
	)
	if err != nil {
		return nil, err
	}

	expectations := append(
		[]BuiltInArtifactExpectation(nil),
		request.Expectations...,
	)
	sort.Slice(expectations, func(left, right int) bool {
		if expectations[left].Subresource != expectations[right].Subresource {
			return expectations[left].Subresource <
				expectations[right].Subresource
		}
		return expectations[left].Kind < expectations[right].Kind
	})
	first := expectations[0]
	if err := first.Kind.Validate(); err != nil {
		return nil, err
	}
	if err := first.LogicalName.Validate(); err != nil {
		return nil, err
	}
	if err := cryptoutil.ValidateDigest(first.DefinitionDigest); err != nil {
		return nil, err
	}

	_, err = a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: request.RootID,
			Binding: artifact.SourceBinding{
				SourceID:           request.SourceID,
				Locator:            documentLocator,
				SubresourceLocator: first.Subresource,
			},
			ExpectedKind:        first.Kind,
			ExpectedLogicalName: first.LogicalName,
			ExpectedDefinition:  first.DefinitionDigest,
			Package: source.ManagedPackagePublication{
				Address: request.PackageAddress,
				Files:   request.PackageFiles,
			},
			AllowProtected: true,
		},
	)
	if err != nil {
		return nil, err
	}

	output := make([]artifact.Artifact, 0, len(expectations))
	for _, expected := range expectations {
		if err := expected.Subresource.Validate(); err != nil {
			return nil, err
		}
		if err := expected.Kind.Validate(); err != nil {
			return nil, err
		}
		if err := expected.LogicalName.Validate(); err != nil {
			return nil, err
		}
		if err := cryptoutil.ValidateDigest(
			expected.DefinitionDigest,
		); err != nil {
			return nil, err
		}

		value, err := a.artifacts.FindByOrigin(
			ctx,
			request.RootID,
			artifact.SourceBinding{
				SourceID:           request.SourceID,
				Locator:            documentLocator,
				SubresourceLocator: expected.Subresource,
			},
			expected.Kind,
		)
		if err != nil {
			return nil, err
		}
		if value.State != artifact.StateAvailable ||
			value.LogicalName != expected.LogicalName ||
			value.ResolvedDefinition == nil ||
			*value.ResolvedDefinition != expected.DefinitionDigest {
			return nil, fmt.Errorf(
				"%w: built-in MCP Artifact %q does not match package expectation",
				basespec.ErrReferenceUnresolved,
				value.ID,
			)
		}
		if value.Enabled != expected.Enabled {
			value, err = a.artifacts.SetEnabled(
				ctx,
				value.Ref(),
				value.Revision,
				expected.Enabled,
			)
			if err != nil {
				return nil, err
			}
		}
		output = append(output, value)
	}
	return output, nil
}

func (a *API) listArtifacts(
	ctx context.Context,
	rootID root.RootID,
	kind artifact.ArtifactKind,
) ([]artifact.Artifact, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	values, err := a.artifacts.ListByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]artifact.Artifact, 0, len(values))
	for _, value := range values {
		if value.Kind == kind {
			output = append(output, value.Clone())
		}
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].LogicalName != output[right].LogicalName {
			return output[left].LogicalName <
				output[right].LogicalName
		}
		return output[left].ID < output[right].ID
	})
	return output, nil
}

type serverResolutionMaterial struct {
	Resource             resource.ResolvedArtifact
	Document             mcpDomainServer.ServerDocument
	Installation         mcpDomainServer.ServerData
	InstallationRevision uint64
	InstallationEnabled  bool
	RuntimeEnabled       bool
	BuiltIn              bool
}

func (a *API) resolveMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
	verifySource bool,
) (mcpDomainServer.Resolved, error) {
	material, err := a.resolveServerMaterial(ctx, ref, verifySource)
	if err != nil {
		return mcpDomainServer.Resolved{}, err
	}
	if material.Resource.Artifact.SourceContentDigest == nil {
		return mcpDomainServer.Resolved{}, fmt.Errorf(
			"%w: MCP Server has no source content digest",
			basespec.ErrDigestMismatch,
		)
	}

	policyValue, err := a.effectivePolicy(
		ctx,
		material.Resource.Artifact.RootID,
		material.Document,
		material.Installation.AdditionalPolicies,
	)
	if err != nil {
		return mcpDomainServer.Resolved{}, err
	}

	version, err := cryptoutil.CanonicalDigest(struct {
		Server               artifact.ArtifactRef `json:"server"`
		ArtifactRevision     uint64               `json:"artifactRevision"`
		DefinitionDigest     cryptoutil.Digest    `json:"definitionDigest"`
		SourceContentDigest  cryptoutil.Digest    `json:"sourceContentDigest"`
		SourceGeneration     string               `json:"sourceGeneration"`
		InstallationRevision uint64               `json:"installationRevision"`
		PolicyDigest         cryptoutil.Digest    `json:"policyDigest"`
	}{
		Server:               material.Resource.Artifact.Ref(),
		ArtifactRevision:     material.Resource.Artifact.Revision,
		DefinitionDigest:     material.Resource.Definition.Digest,
		SourceContentDigest:  *material.Resource.Artifact.SourceContentDigest,
		SourceGeneration:     material.Resource.RefreshState.SourceGeneration,
		InstallationRevision: material.InstallationRevision,
		PolicyDigest:         policyValue.Digest,
	})
	if err != nil {
		return mcpDomainServer.Resolved{}, err
	}

	output := mcpDomainServer.Resolved{
		Server:               material.Resource.Artifact.Ref(),
		ArtifactRevision:     material.Resource.Artifact.Revision,
		DefinitionDigest:     material.Resource.Definition.Digest,
		SourceContentDigest:  *material.Resource.Artifact.SourceContentDigest,
		SourceGeneration:     material.Resource.RefreshState.SourceGeneration,
		Document:             material.Document,
		Installation:         material.Installation,
		Policy:               policyValue,
		InstallationRevision: material.InstallationRevision,
		RuntimeEnabled:       material.RuntimeEnabled,
		BuiltIn:              material.BuiltIn,
		Version:              version,
	}
	if err := output.Validate(); err != nil {
		return mcpDomainServer.Resolved{}, err
	}
	return output, nil
}

func (a *API) resolveServerMaterial(
	ctx context.Context,
	ref artifact.ArtifactRef,
	verifySource bool,
) (serverResolutionMaterial, error) {
	if a == nil {
		return serverResolutionMaterial{}, basespec.ErrClosed
	}
	resolved, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resource.ResolveOptions{
			VerifySourceContent: verifySource,
		},
	)
	if err != nil {
		return serverResolutionMaterial{}, err
	}
	if resolved.Artifact.Kind != mcpDomain.MCPArtifactKind {
		return serverResolutionMaterial{}, fmt.Errorf(
			"%w: Artifact is not an MCP Server",
			basespec.ErrReferenceUnresolved,
		)
	}
	document, err := a.serverDocumentForResolvedArtifact(ctx, resolved)
	if err != nil {
		return serverResolutionMaterial{}, err
	}

	installation, revision, enabled, runtimeEnabled, builtIn, err := a.effectiveInstallation(
		ctx,
		resolved.Artifact,
		document,
	)
	if err != nil {
		return serverResolutionMaterial{}, err
	}
	return serverResolutionMaterial{
		Resource:             resolved.Clone(),
		Document:             document,
		Installation:         installation,
		InstallationRevision: revision,
		InstallationEnabled:  enabled,
		RuntimeEnabled:       runtimeEnabled,
		BuiltIn:              builtIn,
	}, nil
}

func (a *API) effectiveInstallation(
	ctx context.Context,
	record artifact.Artifact,
	document mcpDomainServer.ServerDocument,
) (
	installation mcpDomainServer.ServerData,
	revision uint64,
	enabled bool,
	runtimeEnabled bool,
	builtIn bool,
	err error,
) {
	builtIn = a.protection.IsProtectedRoot(record.RootID)
	if !builtIn {
		data, err := mcpDomainServer.DecodeServerData(record.Data)
		if err != nil {
			return mcpDomainServer.ServerData{}, 0, false, false, false, err
		}
		if err := data.ValidateFor(record.Ref(), document); err != nil {
			return mcpDomainServer.ServerData{}, 0, false, false, false, err
		}
		return data,
			record.Revision,
			record.Enabled,
			record.Enabled,
			false,
			nil
	}

	if a.overlays == nil {
		return mcpDomainServer.ServerData{},
			0,
			false,
			false,
			true,
			fmt.Errorf(
				"%w: protected MCP installation overlay store is unavailable",
				basespec.ErrReferenceUnresolved,
			)
	}
	overlay, found, err := a.overlays.GetServerOverlay(ctx, record.Ref())
	if err != nil {
		return mcpDomainServer.ServerData{}, 0, false, false, true, err
	}
	if !found {
		return mcpDomainServer.DefaultServerData(),
			1,
			false,
			false,
			true,
			nil
	}
	if err := overlay.ServerData.ValidateFor(
		record.Ref(),
		document,
	); err != nil {
		return mcpDomainServer.ServerData{}, 0, false, false, true, err
	}
	return overlay.ServerData,
		overlay.Revision,
		overlay.RuntimeEnabled,
		record.Enabled && overlay.RuntimeEnabled,
		true,
		nil
}

func (a *API) effectivePolicy(
	ctx context.Context,
	rootID root.RootID,
	server mcpDomainServer.ServerDocument,
	additional []artifact.ArtifactRef,
) (mcpPolicy.Effective, error) {
	values := make([]mcpPolicy.MCPPolicy, 0, 1+len(additional))
	if reference := server.Extension.Policy; reference != nil {
		matches, err := a.policyBodiesByLogicalName(
			ctx,
			rootID,
			reference.Ref,
		)
		if err != nil {
			return mcpPolicy.Effective{}, err
		}
		switch len(matches) {
		case 0:
			if reference.Required {
				return mcpPolicy.Effective{}, fmt.Errorf(
					"%w: required MCP Policy %q is unavailable",
					basespec.ErrReferenceUnresolved,
					reference.Ref,
				)
			}
		case 1:
			values = append(values, matches[0])
		default:
			return mcpPolicy.Effective{}, fmt.Errorf(
				"%w: MCP Policy %q is ambiguous in Root",
				basespec.ErrIdentityConflict,
				reference.Ref,
			)
		}
	}

	for _, ref := range additional {
		if ref.RootID != rootID {
			return mcpPolicy.Effective{}, fmt.Errorf(
				"%w: additional MCP Policy belongs to another Root",
				basespec.ErrInvalid,
			)
		}
		resolved, err := a.resources.ResolveArtifact(
			ctx,
			ref,
			resource.ResolveOptions{},
		)
		if err != nil {
			return mcpPolicy.Effective{}, err
		}
		if resolved.Artifact.Kind != mcpDomain.MCPPolicyArtifactKind ||
			!resolved.Artifact.Enabled {
			return mcpPolicy.Effective{}, fmt.Errorf(
				"%w: additional MCP Policy %q is unavailable",
				basespec.ErrReferenceUnresolved,
				ref.ArtifactID,
			)
		}
		body, err := mcpDomainPolicy.BodyFromDefinition(
			resolved.Definition,
		)
		if err != nil {
			return mcpPolicy.Effective{}, err
		}
		values = append(values, body)
	}

	baseline := a.baselinePolicy
	if len(values) != 0 {
		baseline = values[0]
		values = values[1:]
	}
	return mcpPolicy.Compose(baseline, values...)
}

func (a *API) policyBodiesByLogicalName(
	ctx context.Context,
	rootID root.RootID,
	name basespec.LogicalName,
) ([]mcpPolicy.MCPPolicy, error) {
	records, err := a.artifacts.FindByIdentity(
		ctx,
		rootID,
		mcpDomain.MCPPolicyArtifactKind,
		name,
	)
	if err != nil {
		return nil, err
	}
	output := make([]mcpPolicy.MCPPolicy, 0, len(records))
	for _, record := range records {
		if !record.Enabled ||
			record.State != artifact.StateAvailable {
			continue
		}
		resolved, err := a.resources.ResolveArtifact(
			ctx,
			record.Ref(),
			resource.ResolveOptions{},
		)
		if err != nil {
			return nil, err
		}
		body, err := mcpDomainPolicy.BodyFromDefinition(
			resolved.Definition,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, body)
	}
	return output, nil
}
