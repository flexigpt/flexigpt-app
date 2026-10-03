package managedpackage

import (
	"context"

	managedpackageFlowModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managedpackage/model"
)

type API interface {
	Publish(
		ctx context.Context,
		request managedpackageFlowModel.PublishRequest,
	) (managedpackageFlowModel.PublishResult, error)

	Remove(
		ctx context.Context,
		request managedpackageFlowModel.RemoveRequest,
	) error
}
