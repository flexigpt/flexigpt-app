package consumerapi

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

func (a *API) CreateMCPCollection(
	ctx context.Context,
	request plugin.CreateRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Create(ctx, request)
}

func (a *API) ensureMCPBaselineCollection(
	ctx context.Context,
	rootID rootModel.RootID,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.EnsureBaseline(ctx, rootID)
}

func (a *API) GetMCPCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Read(ctx, ref)
}

func (a *API) SetMCPCollectionEnabled(
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

func (a *API) ResolveMCPCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.PluginCapabilityPlan, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginCapabilityPlan{}, spec.ErrClosed
	}
	return a.plugins.ResolveCapabilities(ctx, ref)
}

func (a *API) ListMCPCollections(
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

func (a *API) ListMCPCollectionMemberships(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]plugin.ArtifactMembershipView, error) {
	if a == nil || a.plugins == nil {
		return nil, spec.ErrClosed
	}
	return a.plugins.ListMembershipsForArtifact(ctx, ref)
}

func (a *API) UpdateMCPCollection(
	ctx context.Context,
	request plugin.UpdateRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Update(ctx, request)
}

func (a *API) AddMCPCollectionMember(
	ctx context.Context,
	request plugin.AddMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.AddMember(ctx, request)
}

func (a *API) AddMCPServerToCollection(
	ctx context.Context,
	request plugin.AddArtifactMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.AddArtifactMember(ctx, request)
}

func (a *API) RemoveMCPCollectionMember(
	ctx context.Context,
	request plugin.RemoveMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.RemoveMember(ctx, request)
}

func (a *API) DeleteMCPCollection(
	ctx context.Context,
	request plugin.DeleteRequest,
) error {
	if a == nil || a.plugins == nil {
		return spec.ErrClosed
	}
	return a.plugins.Delete(ctx, request)
}
