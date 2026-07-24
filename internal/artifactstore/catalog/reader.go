package catalog

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Reader interface {
	// GetCurrent returns the latest published snapshot. It may return a valid
	// snapshot together with an error wrapping artifactstore.ErrCatalogStale
	// when the publication no longer matches the current root or attached
	// source metadata.
	GetCurrent(
		ctx context.Context,
		rootID artifactstore.RootID,
	) (Snapshot, error)
}
