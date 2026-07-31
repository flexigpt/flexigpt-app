package skillbundle

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// ArtifactIDProvider belongs to Skill Bundle policy. It is used only for
// feature-driven automatic adoption during bundle refresh.
//
// Explicit managed, adopted, and pinned Skill requests already carry their
// caller-supplied ArtifactID directly.
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
