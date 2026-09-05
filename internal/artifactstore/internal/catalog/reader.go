package catalogimpl

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
)

// Reader is the internal current-Catalog persistence port.
type Reader interface {
	// GetCurrent returns the latest published snapshot. It may return a valid
	// snapshot together with an error wrapping basespec.ErrCatalogStale when
	// Collection, attachment, or Source metadata has changed.
	GetCurrent(
		ctx context.Context,
		ref collection.CollectionRef,
	) (catalog.Snapshot, error)
}

// ReadCurrent establishes the ownership, identity, and validation guarantees
// required by Store internals. A stale Catalog remains readable for diagnostics
// and refresh reconciliation.
func ReadCurrent(
	ctx context.Context,
	reader Reader,
	ref collection.CollectionRef,
) (catalog.Snapshot, error) {
	if reader == nil {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: catalog reader is nil",
			basespec.ErrInvalid,
		)
	}
	if ctx == nil {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: catalog read context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return catalog.Snapshot{}, err
	}
	if err := ref.Validate(); err != nil {
		return catalog.Snapshot{}, err
	}

	value, err := reader.GetCurrent(ctx, ref)
	if err != nil && !errors.Is(err, basespec.ErrCatalogStale) {
		return catalog.Snapshot{}, err
	}
	if value.RootID != ref.RootID ||
		value.CollectionID != ref.CollectionID {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: catalog reader returned a catalog for another collection",
			basespec.ErrInvalid,
		)
	}
	if validateErr := value.Validate(); validateErr != nil {
		return catalog.Snapshot{}, fmt.Errorf(
			"%w: catalog reader returned an invalid catalog: %w",
			basespec.ErrInvalid,
			validateErr,
		)
	}
	return catalog.CloneSnapshot(value), err
}
