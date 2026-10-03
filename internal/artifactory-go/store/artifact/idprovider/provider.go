package idprovider

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

// Provider creates Store-owned Artifact IDs for source synchronization.
//
// Providers never receive this capability. Valid named source declarations
// receive IDs only from Artifact Store.
type Provider interface {
	NewArtifactID(ctx context.Context) (artifactModel.ArtifactID, error)
}

type ProviderFunc func(
	context.Context,
) (artifactModel.ArtifactID, error)

func (f ProviderFunc) NewArtifactID(
	ctx context.Context,
) (artifactModel.ArtifactID, error) {
	return f(ctx)
}

func NewUUIDProvider() Provider {
	return ProviderFunc(
		func(ctx context.Context) (artifactModel.ArtifactID, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			return artifactModel.ArtifactID(uuidutil.NewUUIDv7()), nil
		},
	)
}
