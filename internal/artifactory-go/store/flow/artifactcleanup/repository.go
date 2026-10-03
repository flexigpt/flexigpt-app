package artifactcleanup

import (
	"context"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
)

type Repository interface {
	PurgeArtifactLocalState(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		now time.Time,
	) error
}
