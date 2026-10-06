package skill

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

func (a *Service) CreateSkillPlugin(
	ctx context.Context,
	request pluginAPI.CreateRequest,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Create(ctx, request)
}

func (a *Service) ResolveSkillPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (pluginAPI.PluginCapabilityPlan, error) {
	if a == nil ||
		a.resources == nil ||
		a.plugins == nil {
		return pluginAPI.PluginCapabilityPlan{}, spec.ErrClosed
	}
	return resourceFlow.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (pluginAPI.PluginCapabilityPlan, error) {
			return a.plugins.ResolveCapabilities(
				sessionCtx,
				ref,
			)
		},
	)
}

func (a *Service) GetSkillPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Read(ctx, ref)
}

func (a *Service) SetSkillPluginEnabled(
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

func (a *Service) ListSkillPlugins(
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

func (a *Service) ListSkillPluginMemberships(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]pluginAPI.ArtifactMembershipView, error) {
	if a == nil || a.plugins == nil {
		return nil, spec.ErrClosed
	}
	return a.plugins.ListMembershipsForArtifact(ctx, ref)
}

func (a *Service) UpdateSkillPlugin(
	ctx context.Context,
	request pluginAPI.UpdateRequest,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Update(ctx, request)
}

func (a *Service) AddSkillPluginMember(
	ctx context.Context,
	request pluginAPI.AddMemberRequest,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.AddMember(ctx, request)
}

func (a *Service) AttachSkillArtifactToPlugin(
	ctx context.Context,
	request pluginAPI.AddArtifactMemberRequest,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.AddArtifactMember(ctx, request)
}

func (a *Service) RemoveSkillPluginMember(
	ctx context.Context,
	request pluginAPI.RemoveMemberRequest,
) (pluginAPI.PluginView, error) {
	if a == nil || a.plugins == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return a.plugins.RemoveMember(ctx, request)
}

func (a *Service) DeleteSkillPlugin(
	ctx context.Context,
	request pluginAPI.DeleteRequest,
) error {
	if a == nil || a.plugins == nil {
		return spec.ErrClosed
	}
	return a.plugins.Delete(ctx, request)
}
