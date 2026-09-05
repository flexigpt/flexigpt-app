package store

import (
	"context"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpStorePolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/policy"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
)

type BundleInstallationView struct {
	Bundle             collection.CollectionRef `json:"bundle"`
	BuiltIn            bool                     `json:"builtIn"`
	CollectionRevision uint64                   `json:"collectionRevision"`
	OverlayRevision    uint64                   `json:"overlayRevision"`
	RuntimeEnabled     bool                     `json:"runtimeEnabled"`
}

type ServerInstallationView struct {
	Artifact             artifact.Artifact             `json:"artifact"`
	Collection           collection.CollectionRef      `json:"collection"`
	CatalogRevision      uint64                        `json:"catalogRevision"`
	Document             mcpStoreServer.ServerDocument `json:"document"`
	Installation         mcpStoreServer.ServerData     `json:"installation"`
	InstallationRevision uint64                        `json:"installationRevision"`
	InstallationEnabled  bool                          `json:"installationEnabled"`
	RuntimeEnabled       bool                          `json:"runtimeEnabled"`
	BuiltIn              bool                          `json:"builtIn"`
}

type PolicyView struct {
	Artifact         artifact.Artifact        `json:"artifact"`
	Collection       collection.CollectionRef `json:"collection"`
	CatalogRevision  uint64                   `json:"catalogRevision"`
	Definition       providerapi.Definition   `json:"definition"`
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
) (BundleDocument, error) {
	if a == nil {
		return BundleDocument{}, basespec.ErrClosed
	}

	bundle, err := a.Get(ctx, ref)
	if err != nil {
		return BundleDocument{}, err
	}
	snapshot, err := a.currentCatalog(ctx, bundle)
	if err != nil {
		return BundleDocument{}, err
	}

	entry, err := a.dependencies.Resources.ReadCollectionEntry(
		ctx,
		ref,
		bundle.Source.ID,
		bundle.DocumentLocator,
		basespec.MaxCandidateBytes,
	)
	if err != nil {
		return BundleDocument{}, err
	}
	if entry.CatalogRevision != snapshot.Revision {
		return BundleDocument{}, fmt.Errorf(
			"%w: MCP Bundle Catalog changed during document resolution",
			basespec.ErrCatalogStale,
		)
	}

	document, _, err := a.canonicalizeBundleBytes(ctx, entry.Content)
	if err != nil {
		return BundleDocument{}, err
	}
	if document.LogicalName != bundle.Data.LogicalName ||
		document.LogicalVersion != bundle.Data.LogicalVersion ||
		!maps.Equal(document.Labels, bundle.Data.Labels) ||
		displayName(document) != bundle.Collection.DisplayName ||
		document.Description != bundle.Collection.Description {
		return BundleDocument{}, fmt.Errorf(
			"%w: MCP Bundle document and Collection metadata differ",
			basespec.ErrCatalogStale,
		)
	}

	expected, err := definitionsForDocument(document)
	if err != nil {
		return BundleDocument{}, err
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
			return BundleDocument{}, fmt.Errorf(
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
			return BundleDocument{}, fmt.Errorf(
				"%w: MCP subresource %q differs from the current document",
				basespec.ErrCatalogStale,
				occurrence.Key.SubresourceLocator,
			)
		}
		seen[occurrence.Key.SubresourceLocator] = struct{}{}
	}

	if len(seen) != len(expected) {
		return BundleDocument{}, fmt.Errorf(
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
	return a.listArtifactsByKind(ctx, ref, artifactbuiltin.ServerKind)
}

func (a *API) ListPolicies(
	ctx context.Context,
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	return a.listArtifactsByKind(ctx, ref, artifactbuiltin.PolicyKind)
}

func (a *API) GetServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ServerInstallationView, error) {
	if a == nil {
		return ServerInstallationView{}, basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return ServerInstallationView{}, err
	}

	record, err := a.dependencies.Artifacts.Get(ctx, ref)
	if err != nil {
		return ServerInstallationView{}, err
	}
	if record.Kind != artifactbuiltin.ServerKind ||
		record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil {
		return ServerInstallationView{}, fmt.Errorf(
			"%w: Artifact is not an available MCP Server",
			basespec.ErrReferenceUnresolved,
		)
	}

	bundle, err := a.Get(ctx, collection.CollectionRef{
		RootID:       record.RootID,
		CollectionID: record.CollectionID,
	})
	if err != nil {
		return ServerInstallationView{}, err
	}
	snapshot, err := a.currentCatalog(ctx, bundle)
	if err != nil {
		return ServerInstallationView{}, err
	}
	if _, err := currentServerOccurrence(snapshot, record); err != nil {
		return ServerInstallationView{}, err
	}

	definitionValue, err := definitionForArtifact(snapshot, record)
	if err != nil {
		return ServerInstallationView{}, err
	}
	document, err := serverDocumentFromDefinition(definitionValue)
	if err != nil {
		return ServerInstallationView{}, err
	}

	data, revision, installationEnabled, runtimeEnabled, err := a.effectiveInstallation(
		ctx,
		bundle,
		record,
		document,
	)
	if err != nil {
		return ServerInstallationView{}, err
	}

	return ServerInstallationView{
		Artifact:             record.Clone(),
		Collection:           bundle.Collection.Ref(),
		CatalogRevision:      snapshot.Revision,
		Document:             document,
		Installation:         data,
		InstallationRevision: revision,
		InstallationEnabled:  installationEnabled,
		RuntimeEnabled:       runtimeEnabled,
		BuiltIn: a.dependencies.RootPolicy.IsProtectedRoot(
			record.RootID,
		),
	}, nil
}

func (a *API) InspectMCPPolicy(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (PolicyView, error) {
	if a == nil {
		return PolicyView{}, basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return PolicyView{}, err
	}

	record, err := a.dependencies.Artifacts.Get(ctx, ref)
	if err != nil {
		return PolicyView{}, err
	}
	if record.Kind != artifactbuiltin.PolicyKind ||
		record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil {
		return PolicyView{}, fmt.Errorf(
			"%w: Artifact is not an available MCP Policy",
			basespec.ErrReferenceUnresolved,
		)
	}

	bundle, err := a.Get(ctx, collection.CollectionRef{
		RootID:       record.RootID,
		CollectionID: record.CollectionID,
	})
	if err != nil {
		return PolicyView{}, err
	}
	snapshot, err := a.currentCatalog(ctx, bundle)
	if err != nil {
		return PolicyView{}, err
	}
	if err := requireCurrentPolicyOccurrence(snapshot, record); err != nil {
		return PolicyView{}, err
	}

	definitionValue, err := definitionForArtifact(snapshot, record)
	if err != nil {
		return PolicyView{}, err
	}
	body, err := mcpStorePolicy.PolicyBodyFromDefinition(definitionValue)
	if err != nil {
		return PolicyView{}, err
	}

	return PolicyView{
		Artifact:        record.Clone(),
		Collection:      bundle.Collection.Ref(),
		CatalogRevision: snapshot.Revision,
		Definition:      definitionValue,
		Body:            body,
		EffectiveEnabled: bundle.Collection.Enabled &&
			record.Enabled,
		BuiltIn: a.dependencies.RootPolicy.IsProtectedRoot(
			record.RootID,
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

	builtIn := a.dependencies.RootPolicy.IsProtectedRoot(ref.RootID)
	output := BundleInstallationView{
		Bundle:             ref,
		BuiltIn:            builtIn,
		CollectionRevision: bundle.Collection.Revision,
		RuntimeEnabled:     bundle.Collection.Enabled,
	}
	if !builtIn {
		return output, nil
	}
	if a.dependencies.Overlays == nil {
		return BundleInstallationView{}, fmt.Errorf(
			"%w: protected MCP Bundle overlay store is unavailable",
			basespec.ErrReferenceUnresolved,
		)
	}

	overlay, found, err := a.dependencies.Overlays.GetBundleOverlay(
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
	kind basespec.ArtifactKind,
) ([]artifact.Artifact, error) {
	if _, err := a.Get(ctx, ref); err != nil {
		return nil, err
	}

	records, err := a.dependencies.Artifacts.ListByCollection(ctx, ref)
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

func requireCurrentPolicyOccurrence(
	snapshot catalog.Snapshot,
	record artifact.Artifact,
) error {
	key := catalog.OccurrenceKey{
		CollectionID:       record.CollectionID,
		SourceID:           record.Binding.SourceID,
		Locator:            record.Binding.Locator,
		SubresourceLocator: record.Binding.SubresourceLocator,
	}
	for _, occurrence := range snapshot.Occurrences {
		if occurrence.Key != key {
			continue
		}
		if occurrence.State == catalog.OccurrenceValid &&
			occurrence.Kind == artifactbuiltin.PolicyKind &&
			occurrence.DefinitionDigest != nil &&
			record.ResolvedDefinition != nil &&
			*occurrence.DefinitionDigest == *record.ResolvedDefinition {
			return nil
		}
		break
	}
	return fmt.Errorf(
		"%w: MCP Policy does not match its current Catalog occurrence",
		basespec.ErrCatalogStale,
	)
}

func definitionForArtifact(
	snapshot catalog.Snapshot,
	record artifact.Artifact,
) (providerapi.Definition, error) {
	if record.ResolvedDefinition == nil {
		return providerapi.Definition{}, fmt.Errorf(
			"%w: MCP Artifact %q has no resolved definition fingerprint",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	value, err := catalog.DefinitionForOccurrence(
		snapshot,
		catalog.OccurrenceKey{
			CollectionID:       record.CollectionID,
			SourceID:           record.Binding.SourceID,
			Locator:            record.Binding.Locator,
			SubresourceLocator: record.Binding.SubresourceLocator,
		},
	)
	if err != nil {
		return providerapi.Definition{}, err
	}
	if value.Digest != *record.ResolvedDefinition {
		return providerapi.Definition{}, fmt.Errorf(
			"%w: MCP Artifact %q does not match its current catalog definition",
			basespec.ErrCatalogStale,
			record.ID,
		)
	}
	return value, nil
}
