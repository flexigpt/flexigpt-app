package managedpackage

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
)

type API interface {
	Publish(
		ctx context.Context,
		request artifactModel.PublishArtifactRequest,
	) (artifactModel.PublishArtifactResult, error)

	Remove(
		ctx context.Context,
		request artifactModel.RemoveArtifactRequest,
	) error
}
