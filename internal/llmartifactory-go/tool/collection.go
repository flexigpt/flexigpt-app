package tool

import (
	"context"
	"fmt"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
)

func (a *Service) ListToolPlugins(
	ctx context.Context,
) ([]pluginAPI.ListItem, error) {
	return a.plugins.ListDomain(ctx, pluginAPI.ListRequest{
		RootID: a.builtinRoot,
	})
}

func (a *Service) GetToolPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (pluginAPI.PluginView, error) {
	if err := a.requireBuiltinRef(ref); err != nil {
		return pluginAPI.PluginView{}, err
	}

	view, err := a.plugins.Read(ctx, ref)
	if err != nil {
		return pluginAPI.PluginView{}, err
	}
	if err := a.requireBuiltinArtifact(view.Artifact); err != nil {
		return pluginAPI.PluginView{}, err
	}
	return view, nil
}

func (a *Service) SetToolPluginEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (pluginAPI.PluginView, error) {
	if _, err := a.GetToolPlugin(ctx, ref); err != nil {
		return pluginAPI.PluginView{}, err
	}
	return a.plugins.SetEnabled(ctx, ref, expectedRevision, enabled)
}

func (a *Service) pluginForTool(
	ctx context.Context,
	name spec.LogicalName,
) (pluginAPI.PluginView, error) {
	pluginName, found := a.pluginByTool[name]
	if !found {
		return pluginAPI.PluginView{}, fmt.Errorf(
			"%w: Tool %q has no generated Tool Plugin",
			spec.ErrReferenceUnresolved,
			name,
		)
	}

	entries, err := a.cat.FindByIdentity(
		ctx,
		a.builtinRoot,
		artifactModel.ArtifactKind(pluginv1.PluginType),
		pluginName,
		catalogModel.ListOptions{},
	)
	if err != nil {
		return pluginAPI.PluginView{}, err
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
		return a.GetToolPlugin(ctx, matches[0])
	case 0:
		return pluginAPI.PluginView{}, fmt.Errorf(
			"%w: Tool %q has no Tool Plugin",
			spec.ErrReferenceUnresolved,
			name,
		)
	default:
		return pluginAPI.PluginView{}, fmt.Errorf(
			"%w: Tool %q belongs to %d Tool Plugins",
			spec.ErrIdentityConflict,
			name,
			len(matches),
		)
	}
}
