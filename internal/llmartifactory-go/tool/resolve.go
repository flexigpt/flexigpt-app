package tool

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (a *Service) ResolveEnabledTool(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (ResolvedToolView, error) {
	value, err := a.getTool(ctx, ref)
	if err != nil {
		return ResolvedToolView{}, err
	}
	pluginView, err := a.pluginForTool(
		ctx,
		value.Artifact.LogicalName,
	)
	if err != nil {
		return ResolvedToolView{}, err
	}
	output := ResolvedToolView{
		Tool:   toolView(value),
		Plugin: pluginView,
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
