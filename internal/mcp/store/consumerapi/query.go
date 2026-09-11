package consumerapi

import (
	"context"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpbundlev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpserverv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomainBundle "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/bundle"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

type BundleInstallationView struct {
	Bundle             collection.CollectionRef `json:"bundle"`
	BuiltIn            bool                     `json:"builtIn"`
	CollectionRevision uint64                   `json:"collectionRevision"`
	OverlayRevision    uint64                   `json:"overlayRevision"`
	RuntimeEnabled     bool                     `json:"runtimeEnabled"`
}

type ServerInstallationView struct {
	Artifact             artifact.Artifact              `json:"artifact"`
	Collection           collection.CollectionRef       `json:"collection"`
	CatalogRevision      uint64                         `json:"catalogRevision"`
	Document             mcpDomainServer.ServerDocument `json:"document"`
	Installation         mcpDomainServer.ServerData     `json:"installation"`
	InstallationRevision uint64                         `json:"installationRevision"`
	InstallationEnabled  bool                           `json:"installationEnabled"`
	RuntimeEnabled       bool                           `json:"runtimeEnabled"`
	BuiltIn              bool                           `json:"builtIn"`
}

type PolicyView struct {
	Artifact         artifact.Artifact        `json:"artifact"`
	Collection       collection.CollectionRef `json:"collection"`
	CatalogRevision  uint64                   `json:"catalogRevision"`
	Definition       definition.Definition    `json:"definition"`
	Body             mcpPolicy.MCPPolicy      `json:"body"`
	EffectiveEnabled bool                     `json:"effectiveEnabled"`
	BuiltIn          bool                     `json:"builtIn"`
}

// GetDocument returns the current canonical source-owned MCP Bundle document.
//
// Source access remains inside Artifact Store. This method verifies current
// Catalog inputs, Source revision, Source generation, stable source bytes,
// canonical schema identity, Collection metadata, and all current valid
// subresource Definitions.
func (a *API) GetDocument(
	ctx context.Context,
	ref collection.CollectionRef,
) (mcpDomainBundle.BundleDocument, error) {
	if a == nil {
		return mcpDomainBundle.BundleDocument{}, basespec.ErrClosed
	}

	bundle, err := a.Get(ctx, ref)
	if err != nil {
		return mcpDomainBundle.BundleDocument{}, err
	}

	resolvedEntry, err := a.resources.ReadCollectionEntryWithCatalog(
		ctx,
		ref,
		bundle.Source.ID,
		bundle.DocumentLocator,
		basespec.MaxCandidateBytes,
	)
	if err != nil {
		return mcpDomainBundle.BundleDocument{}, err
	}
	snapshot := resolvedEntry.Catalog
	entry := resolvedEntry.Entry

	document, _, err := a.canonicalizeBundleBytes(ctx, entry.Content)
	if err != nil {
		return mcpDomainBundle.BundleDocument{}, err
	}
	if document.LogicalName != bundle.Data.LogicalName ||
		document.LogicalVersion != bundle.Data.LogicalVersion ||
		!maps.Equal(document.Labels, bundle.Data.Labels) ||
		displayName(document) != bundle.Collection.DisplayName ||
		document.Description != bundle.Collection.Description {
		return mcpDomainBundle.BundleDocument{}, fmt.Errorf(
			"%w: MCP Bundle document and Collection metadata differ",
			basespec.ErrCatalogStale,
		)
	}

	expected, err := mcpDomainBundle.DefinitionsForDocument(document)
	if err != nil {
		return mcpDomainBundle.BundleDocument{}, err
	}
	seen := make(map[basespec.SubresourceLocator]struct{}, len(expected))

	for _, occurrence := range snapshot.Occurrences {
		if occurrence.Key.SourceID != bundle.Source.ID ||
			occurrence.Key.Locator != bundle.DocumentLocator ||
			occurrence.State != catalog.OccurrenceValid {
			continue
		}

		expectedDefinition, wanted := expected[occurrence.Key.SubresourceLocator]
		if !wanted {
			return mcpDomainBundle.BundleDocument{}, fmt.Errorf(
				"%w: Catalog contains an unexpected valid MCP subresource %q",
				basespec.ErrCatalogStale,
				occurrence.Key.SubresourceLocator,
			)
		}
		if occurrence.Kind != expectedDefinition.Kind ||
			occurrence.DefinitionDigest == nil ||
			*occurrence.DefinitionDigest != expectedDefinition.Digest ||
			occurrence.SourceContentDigest == nil ||
			*occurrence.SourceContentDigest != entry.Digest {
			return mcpDomainBundle.BundleDocument{}, fmt.Errorf(
				"%w: MCP subresource %q differs from the current document",
				basespec.ErrCatalogStale,
				occurrence.Key.SubresourceLocator,
			)
		}
		seen[occurrence.Key.SubresourceLocator] = struct{}{}
	}

	if len(seen) != len(expected) {
		return mcpDomainBundle.BundleDocument{}, fmt.Errorf(
			"%w: Catalog does not cover every current MCP document subresource",
			basespec.ErrCatalogStale,
		)
	}
	return document, nil
}

func (a *API) ListServers(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	return a.listArtifactsByKind(ctx, ref, mcpserverv1.MCPServerKind)
}

func (a *API) ListPolicies(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	return a.listArtifactsByKind(ctx, ref, mcppolicyv1.MCPPolicyKind)
}

func (a *API) GetServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ServerInstallationView, error) {
	if a == nil {
		return ServerInstallationView{}, basespec.ErrClosed
	}
	material, err := a.resolveServerMaterial(ctx, ref, false)
	if err != nil {
		return ServerInstallationView{}, err
	}

	return ServerInstallationView{
		Artifact:             material.Resource.Artifact.Clone(),
		Collection:           material.Resource.Collection.Ref(),
		CatalogRevision:      material.Resource.CatalogRevision,
		Document:             material.Document,
		Installation:         material.Installation,
		InstallationRevision: material.InstallationRevision,
		InstallationEnabled:  material.InstallationEnabled,
		RuntimeEnabled:       material.RuntimeEnabled,
		BuiltIn: a.protection.IsProtectedRoot(
			material.Resource.Artifact.RootID,
		),
	}, nil
}

func (a *API) InspectMCPPolicyForRuntime(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (PolicyView, error) {
	if a == nil {
		return PolicyView{}, basespec.ErrClosed
	}
	resolvedResource, err := a.resources.ResolveArtifact(
		ctx,
		ref,
		resource.ResolveOptions{},
	)
	if err != nil {
		return PolicyView{}, err
	}
	if resolvedResource.Artifact.Kind != mcppolicyv1.MCPPolicyKind ||
		resolvedResource.Collection.Kind != mcpbundlev1.MCPBundleKind {
		return PolicyView{}, fmt.Errorf(
			"%w: Artifact is not an available MCP Policy",
			basespec.ErrReferenceUnresolved,
		)
	}
	body, err := mcpDomainPolicy.BodyFromDefinition(
		resolvedResource.Definition,
	)
	if err != nil {
		return PolicyView{}, err
	}

	return PolicyView{
		Artifact:        resolvedResource.Artifact.Clone(),
		Collection:      resolvedResource.Collection.Ref(),
		CatalogRevision: resolvedResource.CatalogRevision,
		Definition:      resolvedResource.Definition.Clone(),
		Body:            body,
		EffectiveEnabled: resolvedResource.Collection.Enabled &&
			resolvedResource.Artifact.Enabled,
		BuiltIn: a.protection.IsProtectedRoot(
			resolvedResource.Artifact.RootID,
		),
	}, nil
}

func (a *API) GetBundleInstallation(
	ctx context.Context,
	ref collection.CollectionRef,
) (BundleInstallationView, error) {
	if a == nil {
		return BundleInstallationView{}, basespec.ErrClosed
	}

	bundle, err := a.Get(ctx, ref)
	if err != nil {
		return BundleInstallationView{}, err
	}

	builtIn := a.protection.IsProtectedRoot(ref.RootID)
	output := BundleInstallationView{
		Bundle:             ref,
		BuiltIn:            builtIn,
		CollectionRevision: bundle.Collection.Revision,
		RuntimeEnabled:     bundle.Collection.Enabled,
	}
	if !builtIn {
		return output, nil
	}
	if a.overlays == nil {
		return BundleInstallationView{}, fmt.Errorf(
			"%w: protected MCP Bundle overlay store is unavailable",
			basespec.ErrReferenceUnresolved,
		)
	}

	overlay, found, err := a.overlays.GetBundleOverlay(
		ctx,
		ref.RootID,
		ref.CollectionID,
	)
	if err != nil {
		return BundleInstallationView{}, err
	}
	if !found {
		output.RuntimeEnabled = false
		return output, nil
	}

	output.OverlayRevision = overlay.Revision
	output.RuntimeEnabled = bundle.Collection.Enabled &&
		overlay.RuntimeEnabled
	return output, nil
}

func (a *API) listArtifactsByKind(
	ctx context.Context,
	ref collection.CollectionRef,
	kind artifact.ArtifactKind,
) ([]artifact.Artifact, error) {
	if _, err := a.Get(ctx, ref); err != nil {
		return nil, err
	}

	records, err := a.artifacts.ListByCollection(ctx, ref)
	if err != nil {
		return nil, err
	}
	output := make([]artifact.Artifact, 0, len(records))
	for _, record := range records {
		if !isMCPKind(record.Kind) {
			return nil, fmt.Errorf(
				"%w: non-MCP Artifact %q exists in MCP Bundle %q",
				basespec.ErrConflict,
				record.ID,
				ref.CollectionID,
			)
		}
		if record.Kind == kind {
			output = append(output, record.Clone())
		}
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].ID < output[right].ID
	})
	return output, nil
}
