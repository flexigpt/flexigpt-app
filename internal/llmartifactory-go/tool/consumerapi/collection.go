package consumerapi

import (
	"context"
	"fmt"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
)

func (a *API) ListToolCollections(
	ctx context.Context,
) ([]plugin.ListItem, error) {
	if err := a.ready(ctx); err != nil {
		return nil, err
	}
	return a.plugins.ListDomain(ctx, plugin.ListRequest{
		RootID: a.builtinRoot,
	})
}

func (a *API) GetToolCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.PluginView, error) {
	if err := a.ready(ctx); err != nil {
		return plugin.PluginView{}, err
	}
	if err := a.requireBuiltinRef(ref); err != nil {
		return plugin.PluginView{}, err
	}

	view, err := a.plugins.Read(ctx, ref)
	if err != nil {
		return plugin.PluginView{}, err
	}
	if err := a.requireBuiltinArtifact(view.Artifact); err != nil {
		return plugin.PluginView{}, err
	}
	return view, nil
}

func (a *API) SetToolCollectionEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (plugin.PluginView, error) {
	if _, err := a.GetToolCollection(ctx, ref); err != nil {
		return plugin.PluginView{}, err
	}
	return a.plugins.SetEnabled(ctx, ref, expectedRevision, enabled)
}

func (a *API) collectionForTool(
	ctx context.Context,
	name spec.LogicalName,
) (plugin.PluginView, error) {
	collectionName, found := a.collectionByTool[name]
	if !found {
		return plugin.PluginView{}, fmt.Errorf(
			"%w: Tool %q has no generated Tool Plugin",
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
		return plugin.PluginView{}, err
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
		return plugin.PluginView{}, fmt.Errorf(
			"%w: Tool %q has no Tool Plugin",
			spec.ErrReferenceUnresolved,
			name,
		)
	default:
		return plugin.PluginView{}, fmt.Errorf(
			"%w: Tool %q belongs to %d Tool Collections",
			spec.ErrIdentityConflict,
			name,
			len(matches),
		)
	}
}
