package consumerapi

import (
	"context"
	"fmt"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
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
	return a.collections.Create(ctx, request)
}

func (a *API) EnsureAgentBaselineCollection(
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

func (a *API) AttachAgentToCollection(
	ctx context.Context,
	request AttachAgentToCollectionRequest,
) (collection.MemberMutationResult, error) {
	if a == nil || a.collections == nil {
		return collection.MemberMutationResult{}, basespec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return collection.MemberMutationResult{}, fmt.Errorf(
			"%w: expected Agent Collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := request.Collection.Validate(); err != nil {
		return collection.MemberMutationResult{}, err
	}
	if err := request.Agent.Validate(); err != nil {
		return collection.MemberMutationResult{}, err
	}

	collectionValue, err := a.collections.Get(ctx, request.Collection)
	if err != nil {
		return collection.MemberMutationResult{}, err
	}
	if collectionValue.Artifact.Revision != request.ExpectedRevision {
		return collection.MemberMutationResult{}, basespec.ErrConflict
	}

	target, err := a.GetAgent(ctx, request.Agent)
	if err != nil {
		return collection.MemberMutationResult{}, err
	}

	member := collection.MemberReference{
		Type: declaration.TypeAgent,
		Name: target.LogicalName,
	}

	switch target.RootID {
	case collectionValue.Artifact.RootID:
		if target.Binding.SourceID != collectionValue.Artifact.Binding.SourceID {
			break
		}
		if target.Binding.SubresourceLocator != "" {
			return collection.MemberMutationResult{}, fmt.Errorf(
				"%w: a contained Agent cannot be attached as an exact managed Collection member",
				basespec.ErrUnsupported,
			)
		}

		member, err = a.collections.MemberForCollectionSource(
			ctx,
			request.Collection,
			declaration.TypeAgent,
			target.LogicalName,
			target.Binding.Locator,
		)
		if err != nil {
			return collection.MemberMutationResult{}, err
		}

	case agentBuiltinRootID():
		member.Scope = declaration.LookupScopeBuiltin

	default:
		return collection.MemberMutationResult{}, fmt.Errorf(
			"%w: Agent %q belongs to unsupported Root %q",
			basespec.ErrUnsupported,
			target.ID,
			target.RootID,
		)
	}

	result, err := a.collections.EnsureMember(
		ctx,
		collection.AddMemberRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedRevision,
			Member:           member,
		},
	)
	if err != nil {
		return collection.MemberMutationResult{}, err
	}

	return result, nil
}

func (a *API) DetachAgentFromCollection(
	ctx context.Context,
	request collection.RemoveMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.RemoveMember(ctx, request)
}

func (a *API) ResolveAgentCollection(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (collection.CollectionCapabilityPlan, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionCapabilityPlan{}, basespec.ErrClosed
	}
	return a.collections.ResolveCapabilities(ctx, ref)
}

func (a *API) ListAgentCollectionMembers(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (collection.CollectionCapabilityPlan, error) {
	return a.ResolveAgentCollection(ctx, ref)
}

func (a *API) ListDirectAgentMemberships(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]collection.ArtifactMembershipView, error) {
	if a == nil || a.collections == nil {
		return nil, basespec.ErrClosed
	}
	if _, err := a.GetAgent(ctx, ref); err != nil {
		return nil, err
	}
	return a.collections.ListMembershipsForArtifact(ctx, ref)
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
