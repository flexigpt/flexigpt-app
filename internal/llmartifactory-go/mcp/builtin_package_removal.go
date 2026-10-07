package mcp

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
)

func (a *Service) purgeBuiltInServerInstallation(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	return a.installation.Purge(ctx, ref)
}
