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
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.Create(ctx, request)
}

func (a *API) ResolveSkillCollection(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (plugin.CollectionCapabilityPlan, error) {
	if a == nil ||
		a.resources == nil ||
		a.collections == nil {
		return plugin.CollectionCapabilityPlan{}, spec.ErrClosed
	}
	return resourceFlow.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) (plugin.CollectionCapabilityPlan, error) {
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
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.Read(ctx, ref)
}

func (a *API) SetSkillCollectionEnabled(
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

func (a *API) ListSkillCollections(
	ctx context.Context,
	rootID rootModel.RootID,
) ([]plugin.ListItem, error) {
	if a == nil || a.collections == nil {
		return nil, spec.ErrClosed
	}
	return a.collections.ListDomain(ctx, plugin.ListRequest{
		RootID: rootID,
	})
}

func (a *API) ListSkillCollectionMemberships(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]plugin.ArtifactMembershipView, error) {
	if a == nil || a.collections == nil {
		return nil, spec.ErrClosed
	}
	return a.collections.ListMembershipsForArtifact(ctx, ref)
}

func (a *API) UpdateSkillCollection(
	ctx context.Context,
	request plugin.UpdateRequest,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.Update(ctx, request)
}

func (a *API) AddSkillCollectionMember(
	ctx context.Context,
	request plugin.AddMemberRequest,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.AddMember(ctx, request)
}

func (a *API) AttachSkillArtifactToCollection(
	ctx context.Context,
	request plugin.AddArtifactMemberRequest,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.AddArtifactMember(ctx, request)
}

func (a *API) RemoveSkillCollectionMember(
	ctx context.Context,
	request plugin.RemoveMemberRequest,
) (plugin.CollectionView, error) {
	if a == nil || a.collections == nil {
		return plugin.CollectionView{}, spec.ErrClosed
	}
	return a.collections.RemoveMember(ctx, request)
}

func (a *API) DeleteSkillCollection(
	ctx context.Context,
	request plugin.DeleteRequest,
) error {
	if a == nil || a.collections == nil {
		return spec.ErrClosed
	}
	return a.collections.Delete(ctx, request)
}
