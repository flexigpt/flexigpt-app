package source

import (
	"context"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
)

type API interface {
	Create(ctx context.Context, rootID rootModel.RootID, draft sourceModel.Draft) (sourceModel.Summary, error)
	Ensure(ctx context.Context, rootID rootModel.RootID, draft sourceModel.Draft) (sourceModel.Summary, bool, error)
	Discard(ctx context.Context, rootID rootModel.RootID, sourceID sourceModel.SourceID, expectedRevision uint64) error
	Get(ctx context.Context, rootID rootModel.RootID, sourceID sourceModel.SourceID) (sourceModel.Summary, error)
	List(ctx context.Context, rootID rootModel.RootID) ([]sourceModel.Summary, error)

	// PrepareDiscovery performs a revision-checked additive declaration
	// discovery update. It does not refresh Source content.
	PrepareDiscovery(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		preparation sourceModel.DiscoveryPreparation,
	) (sourceModel.Summary, error)

	// Update atomically publishes Source lifecycle invalidation when an enabled
	// Source becomes disabled or loses its final discovery configuration.
	Update(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		update sourceModel.Update,
	) (sourceModel.Summary, error)

	// Retire atomically invalidates source-backed Artifacts with retirement
	// evidence before the Source becomes retired.
	Retire(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		expectedRevision uint64,
	) (sourceModel.Summary, error)
	Purge(ctx context.Context, rootID rootModel.RootID, sourceID sourceModel.SourceID, expectedRevision uint64) error
}
