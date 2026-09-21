package consumerapi

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
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

	overlays            mcpOverlay.OverlayRepository
	secretCleaner       mcpDomainServer.SecretCleaner
	baselinePolicy      mcpPolicy.MCPPolicy
	declarationResolver *resolve.Resolver
	collections         *collection.API
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
	options ...Option,
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
	config := apiOptions{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	output := &API{
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		resources:        resources,
		managedArtifacts: managedArtifacts,
		protection:       protection,
		overlays:         overlays,
		secretCleaner:    secretCleaner,
		baselinePolicy:   baselinePolicy,
	}
	locators, err := resolve.NewProviderLocatorResolver(
		config.locatorResolvers,
		mcpLocatorRuntime{api: output},
	)
	if err != nil {
		return nil, fmt.Errorf("bind MCP declaration locator resolvers: %w", err)
	}
	aliases, err := resolve.NewWithOptions(
		resolve.ResolverOptions{
			Artifacts:            artifacts,
			SourceArtifacts:      artifacts,
			SourceEntries:        resources,
			Locators:             locators,
			FallbackProviders:    config.fallbackProviders,
			ProtectedBuiltinRoot: documentTopology.BuiltinRootID(),
			Limits:               resolve.DefaultLimits(),
		},
	)
	if err != nil {
		return nil, err
	}
	collections, err := collection.NewWithResolver(
		sources,
		discovery,
		artifacts,
		managedArtifacts,
		aliases,
		collection.MCPDomainPolicy(),
	)
	if err != nil {
		return nil, err
	}
	output.declarationResolver = aliases
	output.collections = collections
	return output, nil
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

// SetServerEnabled changes only generic Artifact metadata. It is valid for
// both user-owned and protected built-in MCP Artifacts. Its interpretation is
// owned by the caller. MCP Store does not use it to gate installation,
// materialization, connection, or policy composition.
func (a *API) SetServerEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if a == nil {
		return artifact.Artifact{}, basespec.ErrClosed
	}
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if record.Kind != mcpDomain.MCPArtifactKind {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: Artifact is not an MCP Server",
			basespec.ErrUnsupported,
		)
	}
	return a.artifacts.SetEnabled(ctx, ref, expectedRevision, enabled)
}

// SetPolicyEnabled changes only generic Artifact metadata. Its interpretation
// is owned by the caller. MCP policy composition remains independent from this
// catalog state.
func (a *API) SetPolicyEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if a == nil {
		return artifact.Artifact{}, basespec.ErrClosed
	}
	record, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if record.Kind != mcpDomain.MCPPolicyArtifactKind {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: Artifact is not an MCP Policy",
			basespec.ErrUnsupported,
		)
	}
	return a.artifacts.SetEnabled(ctx, ref, expectedRevision, enabled)
}

func (a *API) GetServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ServerInstallationView, error) {
	material, err := a.resolveServerMaterial(ctx, ref)
	if err != nil {
		return ServerInstallationView{}, err
	}
	return ServerInstallationView{
		Artifact:             material.Resource.Artifact.Clone(),
		Document:             material.Document,
		Installation:         material.Installation,
		InstallationRevision: material.InstallationWriteRevision,
		BuiltIn:              material.BuiltIn,
	}, nil
}

func (a *API) GetMCPPolicy(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (PolicyView, error) {
	if a == nil {
		return PolicyView{}, basespec.ErrClosed
	}
	terminal, err := a.resolveDeclarationArtifact(ctx, ref)
	if err != nil {
		return PolicyView{}, err
	}
	resolved, err := a.resources.ResolveArtifact(
		ctx,
		terminal,
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
	body, err := a.policyBodyForResolvedArtifact(
		ctx,
		resolved,
	)
	if err != nil {
		return PolicyView{}, err
	}
	return PolicyView{
		Artifact: resolved.Artifact.Clone(),
		Body:     body,
		BuiltIn: a.protection.IsProtectedRoot(
			resolved.Artifact.RootID,
		),
	}, nil
}

func (a *API) ResolveMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpDomainServer.Resolved, error) {
	return a.resolveMCPServer(ctx, ref)
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
	material, err := a.resolveServerMaterial(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	terminal := material.Resource.Artifact.Ref()
	if material.BuiltIn {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: protected MCP Server installation belongs in an overlay",
			basespec.ErrProtected,
		)
	}
	if material.Resource.Artifact.Revision != expectedArtifactRevision {
		return artifact.Artifact{}, basespec.ErrConflict
	}
	if err := data.ValidateFor(terminal, material.Document); err != nil {
		return artifact.Artifact{}, err
	}
	encoded, err := mcpDomainServer.MergeServerData(
		material.Resource.Artifact.Data,
		data,
	)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if jsonutil.Equal(material.Resource.Artifact.Data, encoded) {
		return material.Resource.Artifact, nil
	}
	updated, err := a.artifacts.UpdateData(
		ctx,
		terminal,
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
	data mcpDomainServer.ServerData,
) error {
	if a == nil {
		return basespec.ErrClosed
	}
	if a.overlays == nil {
		return fmt.Errorf(
			"%w: MCP overlay store is unavailable",
			basespec.ErrReferenceUnresolved,
		)
	}
	material, err := a.resolveServerMaterial(ctx, ref)
	if err != nil {
		return err
	}
	terminal := material.Resource.Artifact.Ref()
	if !a.protection.IsProtectedRoot(terminal.RootID) {
		return fmt.Errorf(
			"%w: MCP Server is not in a protected Root",
			basespec.ErrProtected,
		)
	}
	if !material.BuiltIn {
		return fmt.Errorf(
			"%w: MCP Server is not a protected Artifact",
			basespec.ErrProtected,
		)
	}
	if err := data.ValidateFor(terminal, material.Document); err != nil {
		return err
	}

	current, found, err := a.overlays.GetServerOverlay(ctx, terminal)
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
		SchemaVersion: mcpDomain.InstallationDataSchemaVersion,
		Revision:      nextRevision,
		ServerData:    data,
	}
	if err := a.overlays.PutServerOverlay(
		ctx,
		terminal,
		expectedOverlayRevision,
		next,
	); err != nil {
		return err
	}
	return mcpDomainServer.CleanupUnboundServerSecrets(
		ctx,
		terminal,
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
	return compositionapi.EnsureSourceCurrent(
		ctx,
		a.discovery,
		rootID,
		sourceID,
	)
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
	if !documentTopology.IsBuiltinPackageSource(
		request.RootID,
		request.SourceID,
	) {
		return nil, fmt.Errorf(
			"%w: MCP package does not target the declared built-in package Source",
			basespec.ErrInvalid,
		)
	}
	if err := request.PackageAddress.Validate(); err != nil {
		return nil, err
	}
	if request.PackageAddress.Kind != mcpDomain.MCPCollectionPackageKind {
		return nil, fmt.Errorf(
			"%w: built-in MCP package kind must be %q",
			basespec.ErrInvalid,
			mcpDomain.MCPCollectionPackageKind,
		)
	}
	if err := request.DocumentFile.ValidatePortable(false); err != nil {
		return nil, err
	}
	if !documentTopology.IsCollectionDocumentFile(
		request.DocumentFile,
	) {
		return nil, fmt.Errorf(
			"%w: built-in MCP Collection document is not declared in topology",
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

	expectations, rootExpectation, err := normalizeBuiltInMCPExpectations(
		request.DocumentFile,
		request.Expectations...,
	)
	if err != nil {
		return nil, err
	}

	previousServers, err := a.builtInPackageServers(
		ctx,
		request.RootID,
		request.SourceID,
		request.PackageAddress,
	)
	if err != nil {
		return nil, err
	}
	_, err = a.managedArtifacts.Publish(
		ctx,
		artifact.PublishArtifactRequest{
			RootID: request.RootID,
			Binding: artifact.SourceBinding{
				SourceID:           request.SourceID,
				Locator:            documentLocator,
				SubresourceLocator: rootExpectation.Subresource,
			},
			ExpectedKind:        rootExpectation.Kind,
			ExpectedLogicalName: rootExpectation.LogicalName,
			ExpectedDefinition:  rootExpectation.DefinitionDigest,
			Package: source.ManagedPackagePublication{
				Address: request.PackageAddress,
				Files:   request.PackageFiles,
			},
			AllowPackageReplacement: true,
			AllowProtected:          true,
		},
	)
	if err != nil {
		return nil, err
	}

	output := make([]artifact.Artifact, 0, len(expectations))
	for _, expected := range expectations {
		originLocator, err := request.PackageAddress.FileLocator(
			expected.Locator,
		)
		if err != nil {
			return nil, err
		}

		value, err := a.artifacts.FindByOrigin(
			ctx,
			request.RootID,
			artifact.SourceBinding{
				SourceID:           request.SourceID,
				Locator:            originLocator,
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
		output = append(output, value)
	}
	if err := a.cleanupRemovedBuiltInPackageServers(
		ctx,
		previousServers,
		output,
	); err != nil {
		return nil, err
	}
	return output, nil
}

// ResolveArtifactCapabilities exposes the complete contract capability plan
// for a declaration Artifact visible to the MCP consumer.
func (a *API) ResolveArtifactCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (resolve.CapabilityPlan, error) {
	if a == nil || a.declarationResolver == nil {
		return resolve.CapabilityPlan{}, basespec.ErrClosed
	}
	return a.declarationResolver.ResolveCapabilities(ctx, ref)
}

func normalizeBuiltInMCPExpectations(
	documentFile basespec.Locator,
	values ...BuiltInArtifactExpectation,
) (
	expectations []BuiltInArtifactExpectation,
	rootExpectation BuiltInArtifactExpectation,
	err error,
) {
	if len(values) == 0 ||
		len(values) > basespec.MaxDiscoveryEntries {
		return nil, BuiltInArtifactExpectation{}, fmt.Errorf(
			"%w: built-in MCP package has invalid expectation count",
			basespec.ErrInvalid,
		)
	}

	type origin struct {
		locator     basespec.Locator
		subresource basespec.SubresourceLocator
		kind        artifact.ArtifactKind
	}

	output := append([]BuiltInArtifactExpectation(nil), values...)
	seen := make(map[origin]struct{}, len(output))
	var rootValue BuiltInArtifactExpectation
	rootFound := false

	for index, expected := range output {
		if err := expected.Locator.ValidatePortable(false); err != nil {
			return nil, BuiltInArtifactExpectation{}, err
		}
		if err := expected.Subresource.Validate(); err != nil {
			return nil, BuiltInArtifactExpectation{}, fmt.Errorf(
				"built-in MCP expectations[%d]: %w",
				index,
				err,
			)
		}
		if err := expected.Kind.Validate(); err != nil {
			return nil, BuiltInArtifactExpectation{}, err
		}
		if err := expected.LogicalName.Validate(); err != nil {
			return nil, BuiltInArtifactExpectation{}, err
		}
		if err := cryptoutil.ValidateDigest(
			expected.DefinitionDigest,
		); err != nil {
			return nil, BuiltInArtifactExpectation{}, err
		}

		key := origin{
			locator:     expected.Locator,
			subresource: expected.Subresource,
			kind:        expected.Kind,
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, BuiltInArtifactExpectation{}, fmt.Errorf(
				"%w: built-in MCP package repeats Artifact origin %q/%q",
				basespec.ErrInvalid,
				expected.Locator,
				expected.Subresource,
			)
		}
		seen[key] = struct{}{}

		if expected.Locator != documentFile ||
			expected.Subresource != "" {
			continue
		}
		if rootFound ||
			expected.Kind != artifact.ArtifactKind(
				declaration.TypePlugin,
			) {
			return nil, BuiltInArtifactExpectation{}, fmt.Errorf(
				"%w: built-in MCP package requires exactly one root Collection",
				basespec.ErrInvalid,
			)
		}
		rootValue = expected
		rootFound = true
	}
	if !rootFound {
		return nil, BuiltInArtifactExpectation{}, fmt.Errorf(
			"%w: built-in MCP package has no root Collection",
			basespec.ErrInvalid,
		)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Locator != output[right].Locator {
			return output[left].Locator < output[right].Locator
		}
		if output[left].Kind != output[right].Kind {
			return output[left].Kind < output[right].Kind
		}
		return output[left].LogicalName <
			output[right].LogicalName
	})
	return output, rootValue, nil
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
	Resource                  resource.ResolvedArtifact
	Document                  mcpDomainServer.ServerDocument
	Installation              mcpDomainServer.ServerData
	InstallationRevision      uint64
	InstallationWriteRevision uint64
	BuiltIn                   bool
}

func (a *API) resolveMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpDomainServer.Resolved, error) {
	material, err := a.resolveServerMaterial(ctx, ref)
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
		material.Resource.Artifact.Ref(),
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
) (serverResolutionMaterial, error) {
	if a == nil {
		return serverResolutionMaterial{}, basespec.ErrClosed
	}
	terminal, err := a.resolveDeclarationArtifact(ctx, ref)
	if err != nil {
		return serverResolutionMaterial{}, err
	}
	resolved, err := a.resources.ResolveArtifact(
		ctx,
		terminal,
		resource.ResolveOptions{},
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
	document, err := mcpDomainServer.ServerDocumentFromDefinition(resolved.Definition)
	if err != nil {
		return serverResolutionMaterial{}, err
	}
	installation, effectiveRevision, writeRevision, builtIn, err := a.effectiveInstallation(
		ctx,
		resolved.Artifact,
		document,
	)
	if err != nil {
		return serverResolutionMaterial{}, err
	}
	return serverResolutionMaterial{
		Resource:                  resolved.Clone(),
		Document:                  document,
		Installation:              installation,
		InstallationRevision:      effectiveRevision,
		InstallationWriteRevision: writeRevision,
		BuiltIn:                   builtIn,
	}, nil
}

func (a *API) resolveDeclarationArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.ArtifactRef, error) {
	if a == nil || a.declarationResolver == nil {
		return artifact.ArtifactRef{}, basespec.ErrClosed
	}
	return a.declarationResolver.ResolveTerminalArtifact(ctx, ref)
}

func (a *API) effectiveInstallation(
	ctx context.Context,
	record artifact.Artifact,
	document mcpDomainServer.ServerDocument,
) (
	installation mcpDomainServer.ServerData,
	effectiveRevision uint64,
	writeRevision uint64,
	builtIn bool,
	err error,
) {
	builtIn = a.protection.IsProtectedRoot(record.RootID)
	if !builtIn {
		data, err := mcpDomainServer.DecodeServerData(record.Data)
		if err != nil {
			return mcpDomainServer.ServerData{}, 0, 0, false, err
		}
		if err := data.ValidateFor(record.Ref(), document); err != nil {
			return mcpDomainServer.ServerData{}, 0, 0, false, err
		}
		return data, record.Revision, record.Revision, false, nil
	}

	if a.overlays == nil {
		return mcpDomainServer.ServerData{},
			0,
			0,
			true,
			fmt.Errorf(
				"%w: protected MCP installation overlay store is unavailable",
				basespec.ErrReferenceUnresolved,
			)
	}
	overlay, found, err := a.overlays.GetServerOverlay(ctx, record.Ref())
	if err != nil {
		return mcpDomainServer.ServerData{}, 0, 0, true, err
	}
	if !found {
		return mcpDomainServer.DefaultServerData(), 1, 0, true, nil
	}
	if err := overlay.ServerData.ValidateFor(
		record.Ref(),
		document,
	); err != nil {
		return mcpDomainServer.ServerData{}, 0, 0, true, err
	}
	return overlay.ServerData, overlay.Revision, overlay.Revision, true, nil
}

func (a *API) effectivePolicy(
	ctx context.Context,
	serverRef artifact.ArtifactRef,
	server mcpDomainServer.ServerDocument,
	additional []artifact.ArtifactRef,
) (mcpPolicy.Effective, error) {
	values := make([]mcpPolicy.MCPPolicy, 0, 1+len(additional))
	if reference := server.Configuration.Policy; reference != nil {
		resolvedServer, err := a.declarationResolver.ResolveMCP(
			ctx,
			serverRef,
		)
		if err != nil {
			return mcpPolicy.Effective{}, err
		}
		if resolvedServer.MCP == nil ||
			resolvedServer.MCP.PolicyResult == nil {
			if reference.Required {
				return mcpPolicy.Effective{}, fmt.Errorf(
					"%w: required MCP Policy %q did not resolve",
					basespec.ErrReferenceUnresolved,
					reference.Name,
				)
			}
		} else {
			policyResult := resolvedServer.MCP.PolicyResult
			if !policyResult.IsAvailable() {
				if reference.Required {
					return mcpPolicy.Effective{}, fmt.Errorf(
						"%w: required MCP Policy %q is %s",
						basespec.ErrReferenceUnresolved,
						reference.Name,
						policyResult.Status,
					)
				}
			} else {
				policyRef, found := policyResult.Resolved.ArtifactRef()
				if !found {
					return mcpPolicy.Effective{}, fmt.Errorf(
						"%w: MCP Policy %q did not resolve to an Artifact",
						basespec.ErrReferenceUnresolved,
						reference.Name,
					)
				}
				policy, err := a.policyBodyForArtifact(ctx, policyRef)
				if err != nil {
					return mcpPolicy.Effective{}, err
				}
				values = append(values, policy)
			}
		}
	}

	for _, ref := range additional {
		if ref.RootID != serverRef.RootID {
			return mcpPolicy.Effective{}, fmt.Errorf(
				"%w: additional MCP Policy belongs to another Root",
				basespec.ErrInvalid,
			)
		}
		terminal, err := a.resolveDeclarationArtifact(ctx, ref)
		if err != nil {
			return mcpPolicy.Effective{}, err
		}
		policy, err := a.policyBodyForArtifact(ctx, terminal)
		if err != nil {
			return mcpPolicy.Effective{}, fmt.Errorf(
				"resolve additional MCP Policy %q: %w",
				ref.ArtifactID,
				err,
			)
		}
		values = append(values, policy)
	}

	return mcpPolicy.Compose(a.baselinePolicy, values...)
}

func (a *API) policyBodyForArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpPolicy.MCPPolicy, error) {
	resolved, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resource.ResolveOptions{},
	)
	if err != nil {
		return mcpPolicy.MCPPolicy{}, err
	}
	if resolved.Artifact.Kind != mcpDomain.MCPPolicyArtifactKind {
		return mcpPolicy.MCPPolicy{}, fmt.Errorf(
			"%w: Artifact %q is not an MCP Policy",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return a.policyBodyForResolvedArtifact(ctx, resolved)
}
