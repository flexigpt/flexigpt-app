package plugin

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type EnsureMemberForCollectionSourceRequest struct {
	Plugin           artifactModel.ArtifactRef `json:"plugin"`
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
		request.Plugin,
		request.Type,
		request.Name,
		request.Locator,
	)
	if err != nil {
		return MemberMutationResult{}, err
	}
	return a.EnsureMember(ctx, AddMemberRequest{
		Plugin:           request.Plugin,
		ExpectedRevision: request.ExpectedRevision,
		Member:           member,
	})
}
