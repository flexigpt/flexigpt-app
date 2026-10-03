package managepackage

import (
	"context"

	managepackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/model"
)

type API interface {
	Publish(
		ctx context.Context,
		request managepackageModel.PublishRequest,
	) (managepackageModel.PublishResult, error)

	Remove(
		ctx context.Context,
		request managepackageModel.RemoveRequest,
	) error
}
