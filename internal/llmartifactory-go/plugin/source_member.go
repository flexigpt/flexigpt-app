package plugin

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type EnsureMemberForPluginSourceRequest struct {
	Plugin           artifactModel.ArtifactRef `json:"plugin"`
	ExpectedRevision uint64                    `json:"expectedRevision"`
	Type             declaration.Type          `json:"type"`
	Name             spec.LogicalName          `json:"name"`
	Locator          spec.Locator              `json:"locator"`
}

func (a *API) EnsureMemberForPluginSource(
	ctx context.Context,
	request EnsureMemberForPluginSourceRequest,
) (MemberMutationResult, error) {
	if a == nil {
		return MemberMutationResult{}, spec.ErrClosed
	}

	member, err := a.MemberForPluginSource(
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
