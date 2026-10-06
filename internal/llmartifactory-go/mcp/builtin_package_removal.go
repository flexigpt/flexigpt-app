package mcp

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (a *Service) purgeBuiltInServerInstallation(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	if a == nil || a.installation == nil {
		return spec.ErrClosed
	}
	return a.installation.Purge(ctx, ref)
}
