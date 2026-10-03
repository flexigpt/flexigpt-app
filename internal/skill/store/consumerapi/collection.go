package consumerapi

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

func (a *API) CreateSkillCollection(
	ctx context.Context,
	request collection.CreateRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.Create(ctx, request)
}

func (a *API) ResolveSkillCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (collection.CollectionCapabilityPlan, error) {
	if a == nil ||
		a.resources == nil ||
		a.collections == nil {
		return collection.CollectionCapabilityPlan{}, spec.ErrClosed
	}
	return resource.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (collection.CollectionCapabilityPlan, error) {
			return a.collections.ResolveCapabilities(
				sessionCtx,
				ref,
			)
		},
	)
}

func (a *API) GetSkillCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.Read(ctx, ref)
}

func (a *API) SetSkillCollectionEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

func (a *API) ListSkillCollections(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]collection.ListItem, error) {
	if a == nil || a.collections == nil {
		return nil, spec.ErrClosed
	}
	return a.collections.ListDomain(ctx, collection.ListRequest{
		RootID: rootID,
	})
}

func (a *API) ListSkillCollectionMemberships(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]collection.ArtifactMembershipView, error) {
	if a == nil || a.collections == nil {
		return nil, spec.ErrClosed
	}
	return a.collections.ListMembershipsForArtifact(ctx, ref)
}

func (a *API) UpdateSkillCollection(
	ctx context.Context,
	request collection.UpdateRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.Update(ctx, request)
}

func (a *API) AddSkillCollectionMember(
	ctx context.Context,
	request collection.AddMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.AddMember(ctx, request)
}

func (a *API) AttachSkillArtifactToCollection(
	ctx context.Context,
	request collection.AddArtifactMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.AddArtifactMember(ctx, request)
}

func (a *API) RemoveSkillCollectionMember(
	ctx context.Context,
	request collection.RemoveMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.RemoveMember(ctx, request)
}

func (a *API) DeleteSkillCollection(
	ctx context.Context,
	request collection.DeleteRequest,
) error {
	if a == nil || a.collections == nil {
		return spec.ErrClosed
	}
	return a.collections.Delete(ctx, request)
}
