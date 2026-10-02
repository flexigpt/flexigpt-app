package consumerapi

import (
	"context"
	"fmt"

	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (a *API) purgeBuiltInServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	if a == nil || a.overlays == nil {
		return fmt.Errorf(
			"%w: MCP Artifact local-state repository is unavailable",
			spec.ErrClosed,
		)
	}
	return a.overlays.PurgeServerLocalState(ctx, ref)
}
