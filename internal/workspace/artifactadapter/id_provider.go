package artifactadapter

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// ArtifactIDProvider belongs to the Workspace feature. It is used only when
// Workspace policy elects to automatically adopt an observed occurrence.
//
// Generic Artifact Store accepts the resulting artifact.Draft.ID but never
// allocates it.
type ArtifactIDProvider interface {
	NewArtifactID(ctx context.Context) (basespec.ArtifactID, error)
}

type ArtifactIDProviderFunc func(
	context.Context,
) (basespec.ArtifactID, error)

func (f ArtifactIDProviderFunc) NewArtifactID(
	ctx context.Context,
) (basespec.ArtifactID, error) {
	return f(ctx)
}
