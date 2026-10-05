package consumerapi

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (a *API) ResolveEnabledTool(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
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
		Tool:   toolView(value),
		Plugin: collectionView,
	}
	if !output.Enabled() {
		return ResolvedToolView{}, fmt.Errorf(
			"%w: Tool %q or its Plugin is disabled",
			spec.ErrReferenceUnresolved,
			value.Artifact.LogicalName,
		)
	}
	return output, nil
}
