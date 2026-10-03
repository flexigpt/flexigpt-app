package install

import (
	"context"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

type Repository interface {
	installModel.HydrationStore
	installModel.PackageHydrationStore

	PurgeTopologyRoot(
		ctx context.Context,
		rootID rootModel.RootID,
	) error
}
