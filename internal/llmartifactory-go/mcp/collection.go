package mcp

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

func (a *Service) CreateMCPPlugin(
	ctx context.Context,
	request pluginAPI.CreateRequest,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Create(ctx, request)
}

func (a *Service) EnsureMCPBaselinePlugin(
	ctx context.Context,
	rootID rootModel.RootID,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.EnsureBaseline(ctx, rootID)
}

func (a *Service) GetMCPPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Read(ctx, ref)
}

func (a *Service) SetMCPPluginEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

func (a *Service) ResolveMCPPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (pluginAPI.PluginCapabilityPlan, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginCapabilityPlan{}, spec.ErrClosed
	}
	return a.plugins.ResolveCapabilities(ctx, ref)
}

func (a *Service) ListMCPPlugins(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]pluginAPI.ListItem, error) {
	if a == nil || a.plugins == nil {
		return nil, spec.ErrClosed
	}
	return a.plugins.ListDomain(ctx, pluginAPI.ListRequest{
		RootID: rootID,
	})
}

func (a *Service) ListMCPPluginMemberships(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]pluginAPI.ArtifactMembershipView, error) {
	if a == nil || a.plugins == nil {
		return nil, spec.ErrClosed
	}
	return a.plugins.ListMembershipsForArtifact(ctx, ref)
}

func (a *Service) UpdateMCPPlugin(
	ctx context.Context,
	request pluginAPI.UpdateRequest,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Update(ctx, request)
}

func (a *Service) AddMCPPluginMember(
	ctx context.Context,
	request pluginAPI.AddMemberRequest,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.AddMember(ctx, request)
}

func (a *Service) AddMCPServerToPlugin(
	ctx context.Context,
	request pluginAPI.AddArtifactMemberRequest,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.AddArtifactMember(ctx, request)
}

func (a *Service) RemoveMCPPluginMember(
	ctx context.Context,
	request pluginAPI.RemoveMemberRequest,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.RemoveMember(ctx, request)
}

func (a *Service) DeleteMCPPlugin(
	ctx context.Context,
	request pluginAPI.DeleteRequest,
) error {
	if a == nil || a.plugins == nil {
		return spec.ErrClosed
	}
	return a.plugins.Delete(ctx, request)
}
