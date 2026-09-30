package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

func (a *API) purgeBuiltInServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if a == nil || a.overlays == nil {
		return fmt.Errorf(
			"%w: MCP Artifact local-state repository is unavailable",
			basespec.ErrClosed,
		)
	}
	return a.overlays.PurgeServerLocalState(ctx, ref)
}
