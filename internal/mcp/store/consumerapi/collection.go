package consumerapi

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

func (a *API) CreateMCPCollection(
	ctx context.Context,
	request collection.CreateRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.Create(ctx, request)
}

func (a *API) ensureMCPBaselineCollection(
	ctx context.Context,
	rootID rootModel.RootID,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.EnsureBaseline(ctx, rootID)
}

func (a *API) GetMCPCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.Read(ctx, ref)
}

func (a *API) SetMCPCollectionEnabled(
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

func (a *API) ResolveMCPCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (collection.CollectionCapabilityPlan, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionCapabilityPlan{}, spec.ErrClosed
	}
	return a.collections.ResolveCapabilities(ctx, ref)
}

func (a *API) ListMCPCollections(
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

func (a *API) ListMCPCollectionMemberships(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]collection.ArtifactMembershipView, error) {
	if a == nil || a.collections == nil {
		return nil, spec.ErrClosed
	}
	return a.collections.ListMembershipsForArtifact(ctx, ref)
}

func (a *API) UpdateMCPCollection(
	ctx context.Context,
	request collection.UpdateRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.Update(ctx, request)
}

func (a *API) AddMCPCollectionMember(
	ctx context.Context,
	request collection.AddMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.AddMember(ctx, request)
}

func (a *API) AddMCPServerToCollection(
	ctx context.Context,
	request collection.AddArtifactMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.AddArtifactMember(ctx, request)
}

func (a *API) RemoveMCPCollectionMember(
	ctx context.Context,
	request collection.RemoveMemberRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, spec.ErrClosed
	}
	return a.collections.RemoveMember(ctx, request)
}

func (a *API) DeleteMCPCollection(
	ctx context.Context,
	request collection.DeleteRequest,
) error {
	if a == nil || a.collections == nil {
		return spec.ErrClosed
	}
	return a.collections.Delete(ctx, request)
}
