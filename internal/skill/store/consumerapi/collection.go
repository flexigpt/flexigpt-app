package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/consumerutil"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

func (a *API) CreateSkillCollection(
	ctx context.Context,
	request collection.CreateRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.Create(ctx, request)
}

func (a *API) ResolveSkillCollection(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (collection.CollectionCapabilityPlan, error) {
	if a == nil ||
		a.resources == nil ||
		a.collections == nil {
		return collection.CollectionCapabilityPlan{}, basespec.ErrClosed
	}
	return consumerutil.WithResourceVerificationSession(
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
	ref artifact.ArtifactRef,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.Read(ctx, ref)
}

func (a *API) SetSkillCollectionEnabled(
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

func (a *API) ListSkillCollections(
	ctx context.Context,
	rootID root.RootID,
) ([]collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return nil, basespec.ErrClosed
	}
	return a.collections.ListDomain(ctx, rootID)
}

func (a *API) ListSkillCollectionMemberships(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]collection.ArtifactMembershipView, error) {
	if a == nil || a.collections == nil {
		return nil, basespec.ErrClosed
	}
	return a.collections.ListMembershipsForArtifact(ctx, ref)
}

func (a *API) UpdateSkillCollection(
	ctx context.Context,
	request collection.UpdateRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.Update(ctx, request)
}

func (a *API) AddSkillCollectionMember(
	ctx context.Context,
	request collection.AddMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.AddMember(ctx, request)
}

func (a *API) AttachSkillArtifactToCollection(
	ctx context.Context,
	request collection.AddArtifactMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.AddArtifactMember(ctx, request)
}

func (a *API) RemoveSkillCollectionMember(
	ctx context.Context,
	request collection.RemoveMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.RemoveMember(ctx, request)
}

func (a *API) DeleteSkillCollection(
	ctx context.Context,
	request collection.DeleteRequest,
) error {
	if a == nil || a.collections == nil {
		return basespec.ErrClosed
	}
	return a.collections.Delete(ctx, request)
}
