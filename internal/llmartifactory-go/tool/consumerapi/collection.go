package consumerapi

import (
	"context"
	"fmt"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/domain"
)

func toolCollectionPolicy() plugin.DomainPolicy {
	return plugin.DomainPolicy{
		Name:        "tool",
		ReadOnly:    true,
		PackageKind: toolDomain.ToolCollectionPackageKind,
		DocumentUse: topology.DocumentUseToolCollection,
		AllowedMemberTypes: []declaration.Type{
			declaration.TypeTool,
		},
		AllowedMemberForms: []declaration.MemberForm{
			declaration.MemberNamed,
		},
		ValidateDocument: func(document pluginv1.PluginDocument) error {
			_, err := toolDomain.ValidateToolCollectionDocument(document)
			return err
		},
	}
}

func (a *API) ListToolCollections(
	ctx context.Context,
) ([]plugin.ListItem, error) {
	if err := a.ready(ctx); err != nil {
		return nil, err
	}
	return a.collections.ListDomain(ctx, plugin.ListRequest{
		RootID: a.builtinRoot,
	})
}

func (a *API) GetToolCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.CollectionView, error) {
	if err := a.ready(ctx); err != nil {
		return plugin.CollectionView{}, err
	}
	if err := a.requireBuiltinRef(ref); err != nil {
		return plugin.CollectionView{}, err
	}

	view, err := a.collections.Read(ctx, ref)
	if err != nil {
		return plugin.CollectionView{}, err
	}
	if err := a.requireBuiltinArtifact(view.Artifact); err != nil {
		return plugin.CollectionView{}, err
	}
	return view, nil
}

func (a *API) SetToolCollectionEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (plugin.CollectionView, error) {
	if _, err := a.GetToolCollection(ctx, ref); err != nil {
		return plugin.CollectionView{}, err
	}
	return a.collections.SetEnabled(ctx, ref, expectedRevision, enabled)
}

func (a *API) collectionForTool(
	ctx context.Context,
	name spec.LogicalName,
) (plugin.CollectionView, error) {
	collectionName, found := a.collectionByTool[name]
	if !found {
		return plugin.CollectionView{}, fmt.Errorf(
			"%w: Tool %q has no generated Tool Collection",
			spec.ErrReferenceUnresolved,
			name,
		)
	}

	entries, err := a.cat.FindByIdentity(
		ctx,
		a.builtinRoot,
		artifactModel.ArtifactKind(pluginv1.PluginType),
		collectionName,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return plugin.CollectionView{}, err
	}

	matches := make([]artifactModel.ArtifactRef, 0, 1)
	for _, entry := range entries {
		if entry.State != artifactModel.StateAvailable ||
			entry.Binding.SourceID != a.builtinSource ||
			entry.Binding.SubresourceLocator != "" {
			continue
		}
		matches = append(matches, entry.Ref())
	}
	switch len(matches) {
	case 1:
		return a.GetToolCollection(ctx, matches[0])
	case 0:
		return plugin.CollectionView{}, fmt.Errorf(
			"%w: Tool %q has no Tool Collection",
			spec.ErrReferenceUnresolved,
			name,
		)
	default:
		return plugin.CollectionView{}, fmt.Errorf(
			"%w: Tool %q belongs to %d Tool Collections",
			spec.ErrIdentityConflict,
			name,
			len(matches),
		)
	}
}
