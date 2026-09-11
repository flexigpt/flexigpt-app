package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpbundlev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpserverv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

func (a *API) ResolveMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpDomainServer.Resolved, error) {
	return a.resolveMCPServer(ctx, ref, true)
}

// InspectMCPServerForRuntime establishes Artifact, Collection, Catalog, Definition,
// installation, and policy validity without opening a Source snapshot or
// resolving a secret. It is deliberately for read-only status and setup
// projections. Connection and explicit runtime refresh must use
// ResolveMCPServer, which verifies exact current source bytes.
func (a *API) InspectMCPServerForRuntime(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpDomainServer.Resolved, error) {
	return a.resolveMCPServer(ctx, ref, false)
}

type serverResolutionMaterial struct {
	Resource             resource.ResolvedArtifact
	Bundle               Bundle
	Document             mcpDomainServer.ServerDocument
	Installation         mcpDomainServer.ServerData
	InstallationRevision uint64
	InstallationEnabled  bool
	RuntimeEnabled       bool
}

func (a *API) resolveMCPServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
	verifySource bool,
) (mcpDomainServer.Resolved, error) {
	if a == nil {
		return mcpDomainServer.Resolved{}, basespec.ErrClosed
	}
	material, err := a.resolveServerMaterial(ctx, ref, verifySource)
	if err != nil {
		return mcpDomainServer.Resolved{}, err
	}
	if material.Resource.Occurrence.SourceContentDigest == nil {
		return mcpDomainServer.Resolved{}, fmt.Errorf(
			"%w: MCP Server Source has no current Catalog content digest",
			basespec.ErrCatalogStale,
		)
	}

	policyValue, err := a.effectivePolicy(
		ctx,
		material.Bundle,
		material.Document,
		material.Installation.AdditionalPolicies,
	)
	if err != nil {
		return mcpDomainServer.Resolved{}, err
	}

	version, err := resolvedVersion(struct {
		ServerRef            artifact.ArtifactRef     `json:"server"`
		CollectionRef        collection.CollectionRef `json:"collection"`
		ArtifactRevision     uint64                   `json:"artifactRevision"`
		CatalogRevision      uint64                   `json:"catalogRevision"`
		DefinitionDigest     cryptoutil.Digest        `json:"definitionDigest"`
		SourceContentDigest  cryptoutil.Digest        `json:"sourceContentDigest"`
		SourceGeneration     string                   `json:"sourceGeneration"`
		InstallationRevision uint64                   `json:"installationRevision"`
		PolicyDigest         cryptoutil.Digest        `json:"policyDigest"`
	}{
		ServerRef:            material.Resource.Artifact.Ref(),
		CollectionRef:        material.Resource.Collection.Ref(),
		ArtifactRevision:     material.Resource.Artifact.Revision,
		CatalogRevision:      material.Resource.CatalogRevision,
		DefinitionDigest:     material.Resource.Definition.Digest,
		SourceContentDigest:  *material.Resource.Occurrence.SourceContentDigest,
		SourceGeneration:     material.Resource.SourceGeneration,
		InstallationRevision: material.InstallationRevision,
		PolicyDigest:         policyValue.Digest,
	})
	if err != nil {
		return mcpDomainServer.Resolved{}, err
	}

	resolved := mcpDomainServer.Resolved{
		Server:               material.Resource.Artifact.Ref(),
		Collection:           material.Resource.Collection.Ref(),
		ArtifactRevision:     material.Resource.Artifact.Revision,
		CatalogRevision:      material.Resource.CatalogRevision,
		DefinitionDigest:     material.Resource.Definition.Digest,
		SourceContentDigest:  *material.Resource.Occurrence.SourceContentDigest,
		SourceGeneration:     material.Resource.SourceGeneration,
		Document:             material.Document,
		Installation:         material.Installation,
		Policy:               policyValue,
		InstallationRevision: material.InstallationRevision,
		RuntimeEnabled:       material.RuntimeEnabled,
		BuiltIn: a.protection.IsProtectedRoot(
			material.Resource.Artifact.RootID,
		),
		Version: version,
	}
	if err := resolved.Validate(); err != nil {
		return mcpDomainServer.Resolved{}, err
	}
	return resolved, nil
}

func (a *API) resolveServerMaterial(
	ctx context.Context,
	ref artifact.ArtifactRef,
	verifySource bool,
) (serverResolutionMaterial, error) {
	resourceValue, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resource.ResolveOptions{
			VerifySourceContent: verifySource,
		},
	)
	if err != nil {
		return serverResolutionMaterial{}, err
	}
	if resourceValue.Artifact.Kind != mcpserverv1.MCPServerKind {
		return serverResolutionMaterial{}, fmt.Errorf(
			"%w: Artifact is not an MCP Server",
			basespec.ErrReferenceUnresolved,
		)
	}

	bundle, err := a.Get(ctx, resourceValue.Collection.Ref())
	if err != nil {
		return serverResolutionMaterial{}, err
	}
	if bundle.Collection.Ref() != resourceValue.Collection.Ref() ||
		bundle.Source.ID != resourceValue.Source.ID ||
		bundle.DocumentLocator != resourceValue.Artifact.Binding.Locator {
		return serverResolutionMaterial{}, fmt.Errorf(
			"%w: MCP Bundle topology changed during Server resolution",
			basespec.ErrCatalogStale,
		)
	}

	document, err := mcpDomainServer.ServerDocumentFromDefinition(
		resourceValue.Definition,
	)
	if err != nil {
		return serverResolutionMaterial{}, err
	}

	installation, revision, enabled, runtimeEnabled, err := a.effectiveInstallation(
		ctx,
		bundle,
		resourceValue.Artifact,
		document,
	)
	if err != nil {
		return serverResolutionMaterial{}, err
	}

	return serverResolutionMaterial{
		Resource:             resourceValue.Clone(),
		Bundle:               bundle,
		Document:             document,
		Installation:         installation,
		InstallationRevision: revision,
		InstallationEnabled:  enabled,
		RuntimeEnabled:       runtimeEnabled,
	}, nil
}

func (a *API) currentCatalog(
	ctx context.Context,
	bundle Bundle,
) (catalog.Snapshot, error) {
	return a.catalogs.CurrentCatalog(
		ctx,
		bundle.Collection.Ref(),
	)
}

func (a *API) effectiveInstallation(
	ctx context.Context,
	bundle Bundle,
	record artifact.Artifact,
	document mcpDomainServer.ServerDocument,
) (
	installationData mcpDomainServer.ServerData,
	installationRevision uint64,
	installationEnabled bool,
	runtimeEnabled bool,
	err error,
) {
	if !a.protection.IsProtectedRoot(record.RootID) {
		data, err := mcpDomainServer.DecodeServerData(record.Data)
		if err != nil {
			return mcpDomainServer.ServerData{}, 0, false, false, err
		}
		if err := data.ValidateFor(record.Ref(), document); err != nil {
			return mcpDomainServer.ServerData{}, 0, false, false, err
		}
		return data,
			record.Revision,
			record.Enabled,
			bundle.Collection.Enabled && record.Enabled,
			nil
	}

	if a.overlays == nil {
		return mcpDomainServer.ServerData{},
			0,
			false,
			false,
			fmt.Errorf(
				"%w: protected MCP installation overlay store is unavailable",
				basespec.ErrReferenceUnresolved,
			)
	}

	serverOverlay, found, err := a.overlays.GetServerOverlay(
		ctx,
		record.Ref(),
	)
	if err != nil {
		return mcpDomainServer.ServerData{}, 0, false, false, err
	}
	if !found {
		return mcpDomainServer.DefaultServerData(), 0, false, false, nil
	}
	if err := serverOverlay.ServerData.ValidateFor(
		record.Ref(),
		document,
	); err != nil {
		return mcpDomainServer.ServerData{}, 0, false, false, err
	}
	bundleOverlay, bundleFound, err := a.overlays.GetBundleOverlay(
		ctx,
		record.RootID,
		record.CollectionID,
	)
	if err != nil {
		return mcpDomainServer.ServerData{}, 0, false, false, err
	}
	if !bundleFound {
		return serverOverlay.ServerData,
			serverOverlay.Revision,
			serverOverlay.RuntimeEnabled,
			false,
			nil
	}
	return serverOverlay.ServerData,
		serverOverlay.Revision,
		serverOverlay.RuntimeEnabled,
		bundle.Collection.Enabled &&
			record.Enabled &&
			serverOverlay.RuntimeEnabled &&
			bundleOverlay.RuntimeEnabled,
		nil
}

func (a *API) effectivePolicy(
	ctx context.Context,
	bundle Bundle,
	serverDocument mcpDomainServer.ServerDocument,
	additional []artifact.ArtifactRef,
) (mcpPolicy.Effective, error) {
	values := make([]mcpPolicy.MCPPolicy, 0, 1+len(additional))

	if reference := serverDocument.Extension.Policy; reference != nil {
		matches, err := a.policyBodiesByLogicalName(
			ctx,
			bundle.Collection.Ref(),
			reference.Ref,
		)
		if err != nil {
			return mcpPolicy.Effective{}, err
		}
		switch len(matches) {
		case 0:
			if reference.Required {
				return mcpPolicy.Effective{}, fmt.Errorf(
					"%w: required MCP policy %q is unavailable",
					basespec.ErrReferenceUnresolved,
					reference.Ref,
				)
			}
		case 1:
			values = append(values, matches[0])
		default:
			return mcpPolicy.Effective{}, fmt.Errorf(
				"%w: MCP policy %q is ambiguous",
				basespec.ErrConflict,
				reference.Ref,
			)
		}
	}

	for _, ref := range sortedArtifactRefs(additional) {
		if ref.RootID != bundle.Collection.RootID {
			return mcpPolicy.Effective{}, fmt.Errorf(
				"%w: additional MCP policy belongs to another Root",
				basespec.ErrInvalid,
			)
		}
		resolvedResource, err := a.resources.ResolveArtifact(
			ctx,
			ref,
			resource.ResolveOptions{},
		)
		if err != nil {
			return mcpPolicy.Effective{}, err
		}
		record := resolvedResource.Artifact
		if record.Kind != mcppolicyv1.MCPPolicyKind ||
			resolvedResource.Collection.Kind != mcpbundlev1.MCPBundleKind ||
			resolvedResource.Collection.Ref() != bundle.Collection.Ref() ||
			!record.Enabled {
			return mcpPolicy.Effective{}, fmt.Errorf(
				"%w: additional MCP policy %q is unavailable",
				basespec.ErrReferenceUnresolved,
				ref.ArtifactID,
			)
		}
		body, err := mcpDomainPolicy.BodyFromDefinition(
			resolvedResource.Definition,
		)
		if err != nil {
			return mcpPolicy.Effective{}, err
		}
		values = append(values, body)
	}

	// BaselinePolicy is a fallback, not a mandatory policy floor. A primary
	// policy already replaces it. Apply the same rule when the installation
	// explicitly selects only additional policy Artifacts, otherwise allow,
	// trusted, auto, and Apps-enabled effects can never become effective.
	baseline := a.baselinePolicy
	if len(values) != 0 {
		baseline = values[0]
		values = values[1:]
	}
	return mcpPolicy.Compose(baseline, values...)
}

func (a *API) policyBodiesByLogicalName(
	ctx context.Context,
	ref collection.CollectionRef,
	name basespec.LogicalName,
) ([]mcpPolicy.MCPPolicy, error) {
	inspection, err := a.resources.InspectCollectionResources(ctx, ref)
	if err != nil {
		return nil, err
	}
	if !inspection.Catalog.IsCurrent() {
		return nil, fmt.Errorf(
			"%w: MCP Bundle Catalog is stale",
			basespec.ErrCatalogStale,
		)
	}

	output := make([]mcpPolicy.MCPPolicy, 0)
	for _, value := range inspection.Resources {
		if value.Artifact.Kind != mcppolicyv1.MCPPolicyKind ||
			!value.Artifact.Enabled ||
			!value.CatalogCurrent ||
			value.Resolved == nil {
			continue
		}
		if value.Definition.LogicalName != name {
			continue
		}
		body, err := mcpDomainPolicy.BodyFromDefinition(
			value.Definition,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, body)
	}
	return output, nil
}

func resolvedVersion(value any) (cryptoutil.Digest, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(canonical), nil
}

func sortedArtifactRefs(
	values []artifact.ArtifactRef,
) []artifact.ArtifactRef {
	output := append([]artifact.ArtifactRef(nil), values...)
	sort.Slice(output, func(left, right int) bool {
		if output[left].RootID != output[right].RootID {
			return output[left].RootID < output[right].RootID
		}
		return output[left].ArtifactID < output[right].ArtifactID
	})
	return output
}
