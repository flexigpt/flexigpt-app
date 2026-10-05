package consumerapi

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
)

func (a *API) CreateAgentPlugin(
	ctx context.Context,
	request plugin.CreateRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	if request.RootID == "" {
		rootID, err := a.ensureDefaultAgentCollectionRoot(ctx)
		if err != nil {
			return plugin.PluginView{}, err
		}
		request.RootID = rootID
	}
	return a.plugins.Create(ctx, request)
}

func (a *API) ensureAgentBaselineCollection(
	ctx context.Context,
	rootID rootModel.RootID,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.EnsureBaseline(ctx, rootID)
}

func (a *API) GetAgentPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Read(ctx, ref)
}

func (a *API) ListAgentCollections(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]plugin.ListItem, error) {
	if a == nil || a.plugins == nil {
		return nil, spec.ErrClosed
	}
	return a.plugins.ListDomain(ctx, plugin.ListRequest{RootID: rootID})
}

func (a *API) UpdateAgentCollection(
	ctx context.Context,
	request plugin.UpdateRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.Update(ctx, request)
}

// AddAgentPluginMember adds one explicit named Agent relationship. It is
// useful for relationships such as a user Plugin reference to a protected
// built-in Agent using scope "builtin".
func (a *API) AddAgentPluginMember(
	ctx context.Context,
	request plugin.AddMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.AddMember(ctx, request)
}

// AddAgentPluginArtifactMember adds one currently available root Agent
// Artifact to a Plugin. Same-Root Artifacts are represented by an exact
// source-relative locator; cross-Root Artifact references are rejected by the
// generic Plugin API.
func (a *API) AddAgentPluginArtifactMember(
	ctx context.Context,
	request plugin.AddArtifactMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}

	target, err := a.getAgentRecord(ctx, request.Artifact)
	if err != nil {
		return plugin.PluginView{}, err
	}
	if target.State != artifactModel.StateAvailable {
		return plugin.PluginView{}, fmt.Errorf(
			"%w: Agent Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			target.ID,
		)
	}
	if target.Binding.SubresourceLocator != "" {
		return plugin.PluginView{}, fmt.Errorf(
			"%w: contained Agent Artifacts cannot be direct Plugin members",
			spec.ErrUnsupported,
		)
	}

	return a.plugins.AddArtifactMember(ctx, request)
}

// RemoveAgentPluginMember removes one direct member by the normalized
// index exposed in PluginView.Members.
func (a *API) RemoveAgentPluginMember(
	ctx context.Context,
	request plugin.RemoveMemberRequest,
) (plugin.PluginView, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginView{}, spec.ErrClosed
	}
	return a.plugins.RemoveMember(
		ctx,
		request,
	)
}

func (a *API) SetAgentCollectionEnabled(
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

func (a *API) DeleteAgentPlugin(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	if a == nil || a.plugins == nil {
		return spec.ErrClosed
	}
	return a.plugins.Delete(
		ctx,
		plugin.DeleteRequest{
			Plugin:           ref,
			ExpectedRevision: expectedRevision,
		},
	)
}

func (a *API) ListAgentPluginMembers(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.DirectMembership, error) {
	if a == nil || a.plugins == nil {
		return plugin.DirectMembership{}, spec.ErrClosed
	}
	return a.plugins.ResolveDirectMembers(ctx, ref)
}

// ResolveAgentPluginCapabilities is the explicit recursive graph API.
// It preserves the prior capability-plan behavior without presenting it as a
// direct member-list operation.
func (a *API) ResolveAgentPluginCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.PluginCapabilityPlan, error) {
	if a == nil || a.plugins == nil {
		return plugin.PluginCapabilityPlan{}, spec.ErrClosed
	}
	return a.plugins.ResolveCapabilities(ctx, ref)
}

func (a *API) IsManagedAgentCollection(
	value plugin.PluginView,
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
			value.Artifact.RootID != agentBuiltinRootID() &&
			value.Artifact.Binding.Locator != ""
}

func (a *API) IsAgentArtifact(
	value artifactModel.Artifact,
) bool {
	return agentDomain.IsAgentKind(value.Kind)
}
