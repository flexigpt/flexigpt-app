package consumerapi

import (
	"context"
	"fmt"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

func (a *API) CreateAgentCollection(
	ctx context.Context,
	request collection.CreateRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	if request.RootID == "" {
		rootID, err := a.ensureDefaultAgentCollectionRoot(ctx)
		if err != nil {
			return collection.CollectionView{}, err
		}
		request.RootID = rootID
	}
	return a.collections.Create(ctx, request)
}

func (a *API) ensureAgentBaselineCollection(
	ctx context.Context,
	rootID root.RootID,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.EnsureBaseline(ctx, rootID)
}

func (a *API) GetAgentCollection(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.Read(ctx, ref)
}

func (a *API) ListAgentCollections(
	ctx context.Context,
	rootID root.RootID,
) ([]collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return nil, basespec.ErrClosed
	}
	return a.collections.ListDomain(ctx, rootID)
}

func (a *API) UpdateAgentCollection(
	ctx context.Context,
	request collection.UpdateRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.Update(ctx, request)
}

// AddAgentCollectionMember adds one explicit named Agent relationship. It is
// useful for relationships such as a user Collection reference to a protected
// built-in Agent using scope "builtin".
func (a *API) AddAgentCollectionMember(
	ctx context.Context,
	request collection.AddMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.AddMember(ctx, request)
}

// AddAgentCollectionArtifactMember adds one currently available root Agent
// Artifact to a Collection. Same-Root Artifacts are represented by an exact
// source-relative locator; cross-Root Artifact references are rejected by the
// generic Collection API.
func (a *API) AddAgentCollectionArtifactMember(
	ctx context.Context,
	request collection.AddArtifactMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}

	target, err := a.GetAgent(ctx, request.Artifact)
	if err != nil {
		return collection.CollectionView{}, err
	}
	if target.State != artifact.StateAvailable {
		return collection.CollectionView{}, fmt.Errorf(
			"%w: Agent Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			target.ID,
		)
	}
	if target.Binding.SubresourceLocator != "" {
		return collection.CollectionView{}, fmt.Errorf(
			"%w: contained Agent Artifacts cannot be direct Collection members",
			basespec.ErrUnsupported,
		)
	}

	return a.collections.AddArtifactMember(ctx, request)
}

// RemoveAgentCollectionMember removes one direct member by the normalized
// index exposed in CollectionView.Members.
func (a *API) RemoveAgentCollectionMember(
	ctx context.Context,
	request collection.RemoveMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.RemoveMember(
		ctx,
		request,
	)
}

func (a *API) SetAgentCollectionEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
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
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if a == nil || a.collections == nil {
		return basespec.ErrClosed
	}
	return a.collections.Delete(
		ctx,
		collection.DeleteRequest{
			Collection:       ref,
			ExpectedRevision: expectedRevision,
		},
	)
}

func (a *API) ListAgentCollectionMembers(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (collection.CollectionCapabilityPlan, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionCapabilityPlan{}, basespec.ErrClosed
	}
	return a.collections.ResolveCapabilities(ctx, ref)
}

func (a *API) IsManagedAgentCollection(
	value collection.CollectionView,
) bool {
	if value.Artifact.Binding.SubresourceLocator != "" {
		return false
	}
	if value.Artifact.Kind != artifact.ArtifactKind(
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
	value artifact.Artifact,
) bool {
	return agentDomain.IsAgentKind(value.Kind)
}
