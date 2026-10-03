package driver

import (
	"context"
	"io"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Snapshot is one generation-stable source view.
//
// Implementations must reject operations after Close and must make Confirm
// fail when source content changed since the snapshot was opened.
type Snapshot interface {
	Generation() string

	Stat(
		ctx context.Context,
		locator spec.Locator,
	) (sourceModel.Entry, error)

	ReadDir(
		ctx context.Context,
		locator spec.Locator,
	) ([]sourceModel.Entry, error)

	Open(
		ctx context.Context,
		locator spec.Locator,
	) (io.ReadCloser, error)

	Confirm(ctx context.Context) error
	Close() error
}
