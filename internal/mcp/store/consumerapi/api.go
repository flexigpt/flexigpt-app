package consumerapi

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/composition/local/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/composition/local/consumerutil"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/resource"
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
		mcpLocatorRuntime{artifacts: artifacts},
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
			TargetMappers:        config.targetMappers,
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
	request ListServersRequest,
) ([]ServerListItem, error) {
	return a.listServers(ctx, request)
}

func (a *API) ListPolicies(
	ctx context.Context,
	request ListPoliciesRequest,
) ([]PolicyListItem, error) {
	return a.listPolicies(ctx, request)
}

func (a *API) GetServerSettings(
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
		Installation:         installationDataView(material.Installation),
		InstallationRevision: material.InstallationWriteRevision,
		BuiltIn:              material.BuiltIn,
	}, nil
}

func (a *API) SaveServerSettings(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedSettingsRevision uint64,
	data mcpDomainServer.ServerData,
) error {
	settings, err := a.GetServerSettings(ctx, ref)
	if err != nil {
		return err
	}
	if settings.BuiltIn {
		return a.saveBuiltInServerSettings(
			ctx,
			ref,
			expectedSettingsRevision,
			data,
		)
	}
	_, err = a.saveMutableServerSettings(
		ctx,
		ref,
		expectedSettingsRevision,
		data,
	)
	return err
}

// ListMCPCollectionServers resolves every available server once, sharing one
// resource verification session across the collection and its dependencies.
// Aggregate owns the public projection and runtime identities.
func (a *API) ListMCPCollectionServers(
	ctx context.Context,
	collectionRef artifact.ArtifactRef,
) ([]ServerRead, error) {
	if a == nil ||
		a.collections == nil ||
		a.resources == nil ||
		a.declarationResolver == nil {
		return nil, basespec.ErrClosed
	}
	if err := collectionRef.Validate(); err != nil {
		return nil, err
	}

	return consumerutil.WithResourceVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) ([]ServerRead, error) {
			return a.listMCPCollectionServers(
				sessionCtx,
				collectionRef,
			)
		},
	)
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

func (a *API) saveMutableServerSettings(
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

func (a *API) saveBuiltInServerSettings(
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
		material.Resource.Artifact.Revision,
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

func (a *API) listMCPCollectionServers(
	ctx context.Context,
	collectionRef artifact.ArtifactRef,
) ([]ServerRead, error) {
	if _, err := a.collections.Read(ctx, collectionRef); err != nil {
		return nil, err
	}

	plugin, err := a.declarationResolver.ResolvePluginMembers(
		ctx,
		collectionRef,
	)
	if err != nil {
		return nil, err
	}
	if plugin == nil || plugin.Type != declaration.TypePlugin {
		return nil, fmt.Errorf(
			"%w: MCP Collection did not resolve as a Plugin",
			basespec.ErrReferenceUnresolved,
		)
	}

	refs := make(map[artifact.ArtifactRef]struct{})
	for _, relationship := range plugin.MemberResults {
		if relationship.Declared.Header().Type != declaration.TypeMCP {
			continue
		}
		if relationship.Status == resolve.ResolutionAvailable &&
			relationship.Resolved != nil {
			if ref, found := relationship.Resolved.ArtifactRef(); found {
				refs[ref] = struct{}{}
			}
		}
		if relationship.Selector == nil {
			continue
		}
		for _, match := range relationship.Selector.Matches {
			if match.Status != resolve.ResolutionAvailable ||
				match.Resolved == nil {
				continue
			}
			if ref, found := match.Resolved.ArtifactRef(); found {
				refs[ref] = struct{}{}
			}
		}
	}

	ordered := make([]artifact.ArtifactRef, 0, len(refs))
	for ref := range refs {
		ordered = append(ordered, ref)
	}
	sort.Slice(ordered, func(left, right int) bool {
		if ordered[left].RootID != ordered[right].RootID {
			return ordered[left].RootID < ordered[right].RootID
		}
		return ordered[left].ArtifactID < ordered[right].ArtifactID
	})

	output := make([]ServerRead, 0, len(ordered))
	for _, ref := range ordered {
		material, err := a.resolveServerMaterial(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf(
				"load MCP Collection server %q: %w",
				ref.ArtifactID,
				err,
			)
		}

		read, err := a.serverReadFromMaterial(ctx, material)
		if err != nil {
			return nil, fmt.Errorf(
				"resolve MCP Collection server policy %q: %w",
				ref.ArtifactID,
				err,
			)
		}

		output = append(output, read)
	}

	sort.Slice(output, func(left, right int) bool {
		leftArtifact := output[left].Settings.Artifact
		rightArtifact := output[right].Settings.Artifact
		if leftArtifact.LogicalName != rightArtifact.LogicalName {
			return leftArtifact.LogicalName < rightArtifact.LogicalName
		}
		return leftArtifact.ID < rightArtifact.ID
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
) (ServerRead, error) {
	if a == nil || a.resources == nil {
		return ServerRead{}, basespec.ErrClosed
	}
	return consumerutil.WithResourceVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (ServerRead, error) {
			material, err := a.resolveServerMaterial(sessionCtx, ref)
			if err != nil {
				return ServerRead{}, err
			}
			return a.serverReadFromMaterial(sessionCtx, material)
		},
	)
}

func (a *API) serverReadFromMaterial(
	ctx context.Context,
	material serverResolutionMaterial,
) (ServerRead, error) {
	if material.Resource.Artifact.SourceContentDigest == nil {
		return ServerRead{}, fmt.Errorf(
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
		return ServerRead{}, err
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
		return ServerRead{}, err
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
		return ServerRead{}, err
	}
	return ServerRead{
		Settings: ServerInstallationView{
			Artifact:             material.Resource.Artifact.Clone(),
			Document:             material.Document,
			Installation:         installationDataView(material.Installation),
			InstallationRevision: material.InstallationWriteRevision,
			BuiltIn:              material.BuiltIn,
		},
		Resolved: output,
	}, nil
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
