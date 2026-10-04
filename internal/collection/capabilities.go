package collection

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type CollectionCapabilityPlan struct {
	Collection  CollectionView                 `json:"collection"`
	Occurrences []resolve.CapabilityOccurrence `json:"occurrences"`
	Complete    bool                           `json:"complete"`
}

func (a *API) ResolveCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (CollectionCapabilityPlan, error) {
	if a == nil || a.resolver == nil {
		return CollectionCapabilityPlan{}, fmt.Errorf(
			"%w: Collection resolver is unavailable",
			spec.ErrUnsupported,
		)
	}
	plan, err := a.resolver.ResolvePluginCapabilities(ctx, ref)
	if err != nil {
		return CollectionCapabilityPlan{}, err
	}
	if plan.RootType != declaration.TypePlugin || plan.RootArtifact == nil {
		return CollectionCapabilityPlan{}, fmt.Errorf(
			"%w: Collection did not resolve to a source-backed Plugin",
			spec.ErrReferenceUnresolved,
		)
	}
	view, err := a.Read(ctx, *plan.RootArtifact)
	if err != nil {
		return CollectionCapabilityPlan{}, err
	}
	return CollectionCapabilityPlan{
		Collection:  view,
		Occurrences: plan.Occurrences,
		Complete:    plan.Complete,
	}, nil
}
