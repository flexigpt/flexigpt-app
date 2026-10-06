package artifact

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

// IDProvider allocates Store-owned Artifact identities during synchronization.
type IDProvider interface {
	NewArtifactID(ctx context.Context) (artifactModel.ArtifactID, error)
}

type IDProviderFunc func(context.Context) (artifactModel.ArtifactID, error)

func (f IDProviderFunc) NewArtifactID(ctx context.Context) (artifactModel.ArtifactID, error) {
	return f(ctx)
}

func NewUUIDIDProvider() IDProvider {
	return IDProviderFunc(func(ctx context.Context) (artifactModel.ArtifactID, error) {
		return artifactModel.ArtifactID(uuidutil.NewUUIDv7()), nil
	})
}
