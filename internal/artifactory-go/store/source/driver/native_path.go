package driver

import (
	"context"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// LocalPathResolver is an optional trusted capability.
//
// It is intentionally not part of Snapshot. Source configuration and native
// paths must not leak through ordinary source APIs.
type LocalPathResolver interface {
	ResolveLocalPath(
		ctx context.Context,
		value sourceModel.Source,
		locator spec.Locator,
	) (string, error)
}

type LocalPathCapability interface {
	SupportsLocalPath(
		kind sourceModel.SourceKind,
	) bool
}
