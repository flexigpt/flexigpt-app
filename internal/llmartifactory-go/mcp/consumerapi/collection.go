package consumerapi

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

func (a *API) CreateMCPPlugin(
	ctx context.Context,
	request plugin.CreateRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Create(ctx, request)
}

func (a *API) ensureMCPBaselinePlugin(
	ctx context.Context,
	rootID rootModel.RootID,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.EnsureBaseline(ctx, rootID)
}

func (a *API) GetMCPPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Read(ctx, ref)
}

func (a *API) SetMCPPluginEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

func (a *API) ResolveMCPPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.PluginCapabilityPlan, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginCapabilityPlan{}, spec.ErrClosed
	}
	return a.plugins.ResolveCapabilities(ctx, ref)
}

func (a *API) ListMCPPlugins(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]plugin.ListItem, error) {
	if a == nil || a.plugins == nil {
		return nil, spec.ErrClosed
	}
	return a.plugins.ListDomain(ctx, plugin.ListRequest{
		RootID: rootID,
	})
}

func (a *API) ListMCPPluginMemberships(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]plugin.ArtifactMembershipView, error) {
	if a == nil || a.plugins == nil {
		return nil, spec.ErrClosed
	}
	return a.plugins.ListMembershipsForArtifact(ctx, ref)
}

func (a *API) UpdateMCPPlugin(
	ctx context.Context,
	request plugin.UpdateRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Update(ctx, request)
}

func (a *API) AddMCPPluginMember(
	ctx context.Context,
	request plugin.AddMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.AddMember(ctx, request)
}

func (a *API) AddMCPServerToPlugin(
	ctx context.Context,
	request plugin.AddArtifactMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.AddArtifactMember(ctx, request)
}

func (a *API) RemoveMCPPluginMember(
	ctx context.Context,
	request plugin.RemoveMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.RemoveMember(ctx, request)
}

func (a *API) DeleteMCPPlugin(
	ctx context.Context,
	request plugin.DeleteRequest,
) error {
	if a == nil || a.plugins == nil {
		return spec.ErrClosed
	}
	return a.plugins.Delete(ctx, request)
}
