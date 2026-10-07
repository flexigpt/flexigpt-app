package agent

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
)

func (a *Service) CreateAgentPlugin(
	ctx context.Context,
	request pluginAPI.CreateRequest,
) (pluginAPI.PluginView, error) {
	return a.plugins.Create(ctx, request)
}

func (a *Service) EnsureAgentBaselinePlugin(
	ctx context.Context,
	rootID rootModel.RootID,
) (pluginAPI.PluginView, error) {
	return a.plugins.EnsureBaseline(ctx, rootID)
}

func (a *Service) GetAgentPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (pluginAPI.PluginView, error) {
	return a.plugins.Read(ctx, ref)
}

func (a *Service) ListAgentPlugins(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]pluginAPI.ListItem, error) {
	return a.plugins.ListDomain(ctx, pluginAPI.ListRequest{RootID: rootID})
}

func (a *Service) UpdateAgentPlugin(
	ctx context.Context,
	request pluginAPI.UpdateRequest,
) (pluginAPI.PluginView, error) {
	return a.plugins.Update(ctx, request)
}

// AddAgentPluginMember adds one explicit named Agent relationship. It is
// useful for relationships such as a user Plugin reference to a protected
// built-in Agent using scope "builtin".
func (a *Service) AddAgentPluginMember(
	ctx context.Context,
	request pluginAPI.AddMemberRequest,
) (pluginAPI.PluginView, error) {
	return a.plugins.AddMember(ctx, request)
}

// AddAgentPluginArtifactMember adds one currently available root Agent
// Artifact to a Plugin. Same-Root Artifacts are represented by an exact
// source-relative locator; cross-Root Artifact references are rejected by the
// generic Plugin API.
func (a *Service) AddAgentPluginArtifactMember(
	ctx context.Context,
	request pluginAPI.AddArtifactMemberRequest,
) (pluginAPI.PluginView, error) {
	if _, err := a.getAgentRecord(ctx, request.Artifact); err != nil {
		return pluginAPI.PluginView{}, err
	}

	// Plugin owns the generic availability and root-origin requirements for
	// direct Artifact membership after Agent has established family type.
	return a.plugins.AddArtifactMember(ctx, request)
}

// RemoveAgentPluginMember removes one direct member by the normalized
// index exposed in PluginView.Members.
func (a *Service) RemoveAgentPluginMember(
	ctx context.Context,
	request pluginAPI.RemoveMemberRequest,
) (pluginAPI.PluginView, error) {
	return a.plugins.RemoveMember(
		ctx,
		request,
	)
}

func (a *Service) SetAgentPluginEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (pluginAPI.PluginView, error) {
	return a.plugins.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

func (a *Service) DeleteAgentPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return a.plugins.Delete(
		ctx,
		pluginAPI.DeleteRequest{
			Plugin:           ref,
			ExpectedRevision: expectedRevision,
		},
	)
}

func (a *Service) ListAgentPluginMembers(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (pluginAPI.DirectMembership, error) {
	return a.plugins.ResolveDirectMembers(ctx, ref)
}

// ResolveAgentPluginCapabilities is the explicit recursive graph API.
// It preserves the prior capability-plan behavior without presenting it as a
// direct member-list operation.
func (a *Service) ResolveAgentPluginCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (pluginAPI.PluginCapabilityPlan, error) {
	return a.plugins.ResolveCapabilities(ctx, ref)
}

func (a *Service) IsManagedAgentPlugin(
	value pluginAPI.PluginView,
) bool {
	if value.Artifact.Binding.SubresourceLocator != "" {
		return false
	}
	if value.Artifact.Kind != artifactModel.ArtifactKind(
		pluginv1.PluginType,
	) {
		return false
	}
	return value.Baseline ||
		value.Editable &&
			value.Artifact.LogicalName != "" &&
			value.Artifact.RootID != a.support.BuiltinRoot &&
			value.Artifact.Binding.Locator != ""
}

func (a *Service) IsAgentArtifact(
	value artifactModel.Artifact,
) bool {
	return agentDomain.IsAgentKind(value.Kind)
}
