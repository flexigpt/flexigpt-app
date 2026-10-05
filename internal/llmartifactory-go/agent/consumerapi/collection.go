package consumerapi

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

func (a *API) CreateAgentCollection(
	ctx context.Context,
	request plugin.CreateRequest,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	if request.RootID == "" {
		rootID, err := a.ensureDefaultAgentCollectionRoot(ctx)
		if err != nil {
			return plugin.CollectionView{}, err
		}
		request.RootID = rootID
	}
	return a.collections.Create(ctx, request)
}

func (a *API) ensureAgentBaselineCollection(
	ctx context.Context,
	rootID rootModel.RootID,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.EnsureBaseline(ctx, rootID)
}

func (a *API) GetAgentCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.Read(ctx, ref)
}

func (a *API) ListAgentCollections(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]plugin.ListItem, error) {
	if a == nil || a.collections == nil {
		return nil, spec.ErrClosed
	}
	return a.collections.ListDomain(ctx, plugin.ListRequest{RootID: rootID})
}

func (a *API) UpdateAgentCollection(
	ctx context.Context,
	request plugin.UpdateRequest,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.Update(ctx, request)
}

// AddAgentCollectionMember adds one explicit named Agent relationship. It is
// useful for relationships such as a user Collection reference to a protected
// built-in Agent using scope "builtin".
func (a *API) AddAgentCollectionMember(
	ctx context.Context,
	request plugin.AddMemberRequest,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.AddMember(ctx, request)
}

// AddAgentCollectionArtifactMember adds one currently available root Agent
// Artifact to a Collection. Same-Root Artifacts are represented by an exact
// source-relative locator; cross-Root Artifact references are rejected by the
// generic Collection API.
func (a *API) AddAgentCollectionArtifactMember(
	ctx context.Context,
	request plugin.AddArtifactMemberRequest,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}

	target, err := a.getAgentRecord(ctx, request.Artifact)
	if err != nil {
		return plugin.CollectionView{}, err
	}
	if target.State != artifactModel.StateAvailable {
		return plugin.CollectionView{}, fmt.Errorf(
			"%w: Agent Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			target.ID,
		)
	}
	if target.Binding.SubresourceLocator != "" {
		return plugin.CollectionView{}, fmt.Errorf(
			"%w: contained Agent Artifacts cannot be direct Collection members",
			spec.ErrUnsupported,
		)
	}

	return a.collections.AddArtifactMember(ctx, request)
}

// RemoveAgentCollectionMember removes one direct member by the normalized
// index exposed in CollectionView.Members.
func (a *API) RemoveAgentCollectionMember(
	ctx context.Context,
	request plugin.RemoveMemberRequest,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.RemoveMember(
		ctx,
		request,
	)
}

func (a *API) SetAgentCollectionEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

func (a *API) DeleteAgentCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	if a == nil || a.collections == nil {
		return spec.ErrClosed
	}
	return a.collections.Delete(
		ctx,
		plugin.DeleteRequest{
			Collection:       ref,
			ExpectedRevision: expectedRevision,
		},
	)
}

func (a *API) ListAgentCollectionMembers(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.CollectionCapabilityPlan, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionCapabilityPlan{}, spec.ErrClosed
	}
	return a.collections.ResolveCapabilities(ctx, ref)
}

func (a *API) IsManagedAgentCollection(
	value plugin.CollectionView,
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
