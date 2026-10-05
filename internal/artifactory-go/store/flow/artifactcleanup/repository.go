package artifactcleanup

import (
	"context"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
)

type Repository interface {
	CleanupArtifactLocalState(
		ctx context.Context,
		request PurgeRequest,
		now time.Time,
	) (artifactModel.Artifact, error)
}
