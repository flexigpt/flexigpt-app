package internal

import (
	"context"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

// Repository is duplicated here only to preserve a real owner-local private
// implementation boundary; the public Install package owns construction.
type Repository interface {
	installModel.HydrationStore
	installModel.PackageHydrationStore
	PurgeTopologyRoot(ctx context.Context, rootID rootModel.RootID) error
}
