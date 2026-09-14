package artifactid

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

// Provider creates Store-owned Artifact IDs for source synchronization.
//
// Providers never receive this capability. Valid named source declarations
// receive IDs only from Artifact Store.
type Provider interface {
	NewArtifactID(ctx context.Context) (artifact.ArtifactID, error)
}

type ProviderFunc func(
	context.Context,
) (artifact.ArtifactID, error)

func (f ProviderFunc) NewArtifactID(
	ctx context.Context,
) (artifact.ArtifactID, error) {
	return f(ctx)
}

func NewUUIDProvider() Provider {
	return ProviderFunc(
		func(ctx context.Context) (artifact.ArtifactID, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			return artifact.ArtifactID(uuidutil.NewUUIDv7()), nil
		},
	)
}
