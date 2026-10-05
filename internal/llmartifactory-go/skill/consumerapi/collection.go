package consumerapi

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

func (a *API) CreateSkillCollection(
	ctx context.Context,
	request plugin.CreateRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Create(ctx, request)
}

func (a *API) ResolveSkillCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.PluginCapabilityPlan, error) {
	if a == nil ||
		a.resources == nil ||
		a.plugins == nil {
		return plugin.PluginCapabilityPlan{}, spec.ErrClosed
	}
	return resourceFlow.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (plugin.PluginCapabilityPlan, error) {
			return a.plugins.ResolveCapabilities(
				sessionCtx,
				ref,
			)
		},
	)
}

func (a *API) GetSkillCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Read(ctx, ref)
}

func (a *API) SetSkillCollectionEnabled(
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

func (a *API) ListSkillCollections(
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

func (a *API) ListSkillCollectionMemberships(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]plugin.ArtifactMembershipView, error) {
	if a == nil || a.plugins == nil {
		return nil, spec.ErrClosed
	}
	return a.plugins.ListMembershipsForArtifact(ctx, ref)
}

func (a *API) UpdateSkillCollection(
	ctx context.Context,
	request plugin.UpdateRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Update(ctx, request)
}

func (a *API) AddSkillCollectionMember(
	ctx context.Context,
	request plugin.AddMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.AddMember(ctx, request)
}

func (a *API) AttachSkillArtifactToCollection(
	ctx context.Context,
	request plugin.AddArtifactMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.AddArtifactMember(ctx, request)
}

func (a *API) RemoveSkillCollectionMember(
	ctx context.Context,
	request plugin.RemoveMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.RemoveMember(ctx, request)
}

func (a *API) DeleteSkillCollection(
	ctx context.Context,
	request plugin.DeleteRequest,
) error {
	if a == nil || a.plugins == nil {
		return spec.ErrClosed
	}
	return a.plugins.Delete(ctx, request)
}
