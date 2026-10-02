package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
)

func (a *API) ResolveEnabledTool(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedToolView, error) {
	if err := a.ready(ctx); err != nil {
		return ResolvedToolView{}, err
	}

	value, err := a.getTool(ctx, ref)
	if err != nil {
		return ResolvedToolView{}, err
	}
	collectionView, err := a.collectionForTool(
		ctx,
		value.Artifact.LogicalName,
	)
	if err != nil {
		return ResolvedToolView{}, err
	}
	output := ResolvedToolView{
		Tool:       toolView(value),
		Collection: collectionView,
	}
	if !output.Enabled() {
		return ResolvedToolView{}, fmt.Errorf(
			"%w: Tool %q or its Collection is disabled",
			model.ErrReferenceUnresolved,
			value.Artifact.LogicalName,
		)
	}
	return output, nil
}
