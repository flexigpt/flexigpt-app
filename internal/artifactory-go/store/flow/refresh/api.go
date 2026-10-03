package refresh

import (
	"context"

	refreshModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
)

type API interface {
	RefreshRoot(
		ctx context.Context,
		rootID rootModel.RootID,
	) (refreshModel.RefreshRootResult, error)

	RefreshSource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) (refreshModel.RefreshSourceResult, error)

	// RefreshInspection remains in source/model during this first split.
	// It moves to flow/refresh/model in Phase 2.
	InspectSource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) (sourceModel.RefreshInspection, error)
}
