package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
)

func (a *API) purgeBuiltInServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if a == nil || a.overlays == nil {
		return fmt.Errorf(
			"%w: MCP Artifact local-state repository is unavailable",
			model.ErrClosed,
		)
	}
	return a.overlays.PurgeServerLocalState(ctx, ref)
}
