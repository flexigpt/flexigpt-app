package source

import (
	"context"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Runtime is a trusted composition capability. It intentionally exposes the
// full normalized Source value only to Store flows and composition, never to
// ordinary Source API consumers.
type Runtime interface {
	Get(
		ctx context.Context,
		rootID rootModel.RootID,
		id sourceModel.SourceID,
	) (sourceModel.Source, error)

	Open(
		ctx context.Context,
		value sourceModel.Source,
	) (driver.Snapshot, error)

	List(
		ctx context.Context,
		rootID rootModel.RootID,
	) ([]sourceModel.Source, error)
}

// LocalPathRuntime is a trusted runtime extension. Native paths are never
// exposed through source.API or Source summaries.
type LocalPathRuntime interface {
	ResolveLocalPath(
		ctx context.Context,
		value sourceModel.Source,
		locator spec.Locator,
	) (string, error)

	SupportsLocalPath(
		kind sourceModel.SourceKind,
	) bool
}
