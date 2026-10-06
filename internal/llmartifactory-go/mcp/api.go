package mcp

import (
	"context"
	"fmt"
	"sort"

	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/installation"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/overlay"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type Service struct {
	cat              catalog.API
	sources          source.API
	discovery        refreshFlow.API
	artifacts        artifact.API
	resources        resourceFlow.API
	managedArtifacts managepackageFlow.API
	protection       root.ProtectionAPI
	definitions      definition.API

	baselinePolicy      mcpPolicy.MCPPolicy
	installation        *installation.Service
	declarationResolver *composition.Resolver
	plugins             *pluginAPI.API
}

func New(
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	resources resourceFlow.API,
	managedArtifacts managepackageFlow.API,
	protection root.ProtectionAPI,
	cat catalog.API,
	definitions definition.API,
	overlays mcpOverlay.OverlayRepository,
	secretCleaner serverMCPDomain.SecretCleaner,
	baselinePolicy mcpPolicy.MCPPolicy,
	options ...Option,
) (*Service, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil ||
		secretCleaner == nil || cat == nil || definitions == nil {
		return nil, fmt.Errorf(
			"%w: MCP Store dependencies are incomplete",
			spec.ErrInvalid,
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
	aliases, err := requiredCompositionResolver(config.resolver)
	if err != nil {
		return nil, err
	}
	installations, err := installation.New(installation.Dependencies{
		Artifacts:     artifacts,
		Resources:     resources,
		Protection:    protection,
		Overlays:      overlays,
		Declarations:  aliases,
		SecretCleaner: secretCleaner,
	})
	if err != nil {
		return nil, err
	}

	output := &Service{
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		resources:        resources,
		managedArtifacts: managedArtifacts,
		protection:       protection,
		baselinePolicy:   baselinePolicy,
		cat:              cat,
		definitions:      definitions,
		installation:     installations,
	}
	plugins, err := pluginAPI.New(
		artifacts,
		cat,
		sources,
		discovery,
		managedArtifacts,
		definitions,
		aliases,
		mcpDomain.PluginProfile(),
	)
	if err != nil {
		return nil, err
	}
	output.declarationResolver = aliases
	output.plugins = plugins
	return output, nil
}

func (a *Service) ListServers(
	ctx context.Context,
	request ListServersRequest,
) ([]ServerListItem, error) {
	return a.listServers(ctx, request)
}

func (a *Service) ListPolicies(
	ctx context.Context,
	request ListPoliciesRequest,
) ([]PolicyListItem, error) {
	return a.listPolicies(ctx, request)
}

func (a *Service) GetServerSettings(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ServerInstallationView, error) {
	material, err := a.installation.Resolve(ctx, ref)
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

func (a *Service) SaveServerSettings(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedSettingsRevision uint64,
	data serverMCPDomain.ServerData,
) error {
	if a.installation == nil {
		return spec.ErrClosed
	}
	_, err := a.installation.Save(
		ctx,
		ref,
		expectedSettingsRevision,
		data,
	)
	return err
}

// ListMCPPluginServers resolves every available server once, sharing one
// resource verification session across the plugin and its dependencies.
// Aggregate owns the public projection and runtime identities.
func (a *Service) ListMCPPluginServers(
	ctx context.Context,
	pluginRef artifactModel.ArtifactRef,
) ([]ServerRead, error) {
	if a.plugins == nil ||
		a.resources == nil ||
		a.declarationResolver == nil {
		return nil, spec.ErrClosed
	}
	if err := pluginRef.Validate(); err != nil {
		return nil, err
	}

	return resourceFlow.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) ([]ServerRead, error) {
			return a.listMCPPluginServers(
				sessionCtx,
				pluginRef,
			)
		},
	)
}

func (a *Service) GetMCPPolicy(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (PolicyView, error) {
	if a == nil {
		return PolicyView{}, spec.ErrClosed
	}
	terminal, err := a.resolveDeclarationArtifact(ctx, ref)
	if err != nil {
		return PolicyView{}, err
	}
	resolved, err := a.resources.ResolveArtifact(
		ctx,
		terminal,
		resourceModel.ResolveOptions{},
	)
	if err != nil {
		return PolicyView{}, err
	}
	if resolved.Artifact.Kind != mcpDomain.MCPPolicyArtifactKind {
		return PolicyView{}, fmt.Errorf(
			"%w: Artifact is not an MCP Policy",
			spec.ErrReferenceUnresolved,
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

func (a *Service) ResolveMCPServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ServerRead, error) {
	if a.resources == nil {
		return ServerRead{}, spec.ErrClosed
	}
	return resourceFlow.WithVerificationSession(
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

func (a *Service) listMCPPluginServers(
	ctx context.Context,
	pluginRef artifactModel.ArtifactRef,
) ([]ServerRead, error) {
	if _, err := a.plugins.Read(ctx, pluginRef); err != nil {
		return nil, err
	}

	p, err := a.declarationResolver.ResolvePluginMembers(
		ctx,
		pluginRef,
	)
	if err != nil {
		return nil, err
	}
	if p == nil || p.Type != declaration.TypePlugin {
		return nil, fmt.Errorf(
			"%w: MCP Plugin did not resolve as a Plugin",
			spec.ErrReferenceUnresolved,
		)
	}

	refs := make(map[artifactModel.ArtifactRef]struct{})
	for _, relationship := range p.Relationships {
		if relationship.Declared.Header().Type != declaration.TypeMCP {
			continue
		}
		if relationship.Status == composition.ResolutionAvailable &&
			relationship.Resolved != nil {
			if ref, found := relationship.Resolved.ArtifactRef(); found {
				refs[ref] = struct{}{}
			}
		}
		if relationship.Selector == nil {
			continue
		}
		for _, match := range relationship.Selector.Matches {
			if match.Status != composition.ResolutionAvailable ||
				match.Resolved == nil {
				continue
			}
			if ref, found := match.Resolved.ArtifactRef(); found {
				refs[ref] = struct{}{}
			}
		}
	}

	ordered := make([]artifactModel.ArtifactRef, 0, len(refs))
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
				"load MCP Plugin server %q: %w",
				ref.ArtifactID,
				err,
			)
		}

		read, err := a.serverReadFromMaterial(ctx, material)
		if err != nil {
			return nil, fmt.Errorf(
				"resolve MCP Plugin server policy %q: %w",
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

func (a *Service) serverReadFromMaterial(
	ctx context.Context,
	material installation.Material,
) (ServerRead, error) {
	if material.Resource.Artifact.SourceContentDigest == nil {
		return ServerRead{}, fmt.Errorf(
			"%w: MCP Server has no source content digest",
			spec.ErrDigestMismatch,
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
		Server               artifactModel.ArtifactRef `json:"server"`
		ArtifactRevision     uint64                    `json:"artifactRevision"`
		DefinitionDigest     cryptoutil.Digest         `json:"definitionDigest"`
		SourceContentDigest  cryptoutil.Digest         `json:"sourceContentDigest"`
		SourceGeneration     string                    `json:"sourceGeneration"`
		InstallationRevision uint64                    `json:"installationRevision"`
		PolicyDigest         cryptoutil.Digest         `json:"policyDigest"`
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

	output := serverMCPDomain.Resolved{
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

func (a *Service) resolveServerMaterial(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (installation.Material, error) {
	if a.installation == nil {
		return installation.Material{}, spec.ErrClosed
	}
	return a.installation.Resolve(ctx, ref)
}

func (a *Service) resolveDeclarationArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.ArtifactRef, error) {
	if a.declarationResolver == nil {
		return artifactModel.ArtifactRef{}, spec.ErrClosed
	}
	return a.declarationResolver.ResolveTerminalArtifact(ctx, ref)
}

func resolvedMCPPolicyRelationship(
	value *composition.ResolvedEntry,
	name spec.LogicalName,
) (*composition.ResolvedRelationship, bool) {
	if value == nil {
		return nil, false
	}
	for _, relationship := range value.Relationships {
		header := relationship.Declared.Header()
		if header.Type != declaration.TypeMCPPolicy ||
			header.Name != string(name) {
			continue
		}
		copyValue := relationship
		return &copyValue, true
	}
	return nil, false
}

func (a *Service) effectivePolicy(
	ctx context.Context,
	serverRef artifactModel.ArtifactRef,
	server serverMCPDomain.ServerDocument,
	additional []artifactModel.ArtifactRef,
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
		policyResult, found := resolvedMCPPolicyRelationship(
			resolvedServer,
			reference.Name,
		)
		if !found {
			if reference.Required {
				return mcpPolicy.Effective{}, fmt.Errorf(
					"%w: required MCP Policy %q did not resolve",
					spec.ErrReferenceUnresolved,
					reference.Name,
				)
			}
		} else {
			if !policyResult.IsAvailable() {
				if reference.Required {
					return mcpPolicy.Effective{}, fmt.Errorf(
						"%w: required MCP Policy %q is %s",
						spec.ErrReferenceUnresolved,
						reference.Name,
						policyResult.Status,
					)
				}
			} else {
				if policyResult.Resolved == nil ||
					policyResult.Resolved.Target == nil ||
					policyResult.Resolved.Target.Form !=
						composition.TargetFormArtifact ||
					policyResult.Resolved.Target.Artifact == nil {
					return mcpPolicy.Effective{}, fmt.Errorf(
						"%w: MCP Policy %q did not resolve to a source-backed Artifact",
						spec.ErrReferenceUnresolved,
						reference.Name,
					)
				}
				policyRef := *policyResult.Resolved.Target.Artifact
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
				spec.ErrInvalid,
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

func (a *Service) policyBodyForArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (mcpPolicy.MCPPolicy, error) {
	resolved, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resourceModel.ResolveOptions{},
	)
	if err != nil {
		return mcpPolicy.MCPPolicy{}, err
	}
	if resolved.Artifact.Kind != mcpDomain.MCPPolicyArtifactKind {
		return mcpPolicy.MCPPolicy{}, fmt.Errorf(
			"%w: Artifact %q is not an MCP Policy",
			spec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	return a.policyBodyForResolvedArtifact(ctx, resolved)
}
