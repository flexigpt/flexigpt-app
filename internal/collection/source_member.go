package collection

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

type EnsureMemberForCollectionSourceRequest struct {
	Collection       artifact.ArtifactRef `json:"collection"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	Type             declaration.Type     `json:"type"`
	Name             basespec.LogicalName `json:"name"`
	Locator          basespec.Locator     `json:"locator"`
}

func (a *API) EnsureMemberForCollectionSource(
	ctx context.Context,
	request EnsureMemberForCollectionSourceRequest,
) (MemberMutationResult, error) {
	if a == nil {
		return MemberMutationResult{}, basespec.ErrClosed
	}

	member, err := a.MemberForCollectionSource(
		ctx,
		request.Collection,
		request.Type,
		request.Name,
		request.Locator,
	)
	if err != nil {
		return MemberMutationResult{}, err
	}
	return a.EnsureMember(ctx, AddMemberRequest{
		Collection:       request.Collection,
		ExpectedRevision: request.ExpectedRevision,
		Member:           member,
	})
}
