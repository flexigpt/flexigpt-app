package plugin

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type EnsureMemberForCollectionSourceRequest struct {
	Collection       artifactModel.ArtifactRef `json:"collection"`
	ExpectedRevision uint64                    `json:"expectedRevision"`
	Type             declaration.Type          `json:"type"`
	Name             spec.LogicalName          `json:"name"`
	Locator          spec.Locator              `json:"locator"`
}

func (a *API) EnsureMemberForCollectionSource(
	ctx context.Context,
	request EnsureMemberForCollectionSourceRequest,
) (MemberMutationResult, error) {
	if a == nil {
		return MemberMutationResult{}, spec.ErrClosed
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
