package artifactcleanup

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
)

type API interface {
	PurgeArtifactLocalState(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) error
}
