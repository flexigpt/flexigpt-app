package catalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
)

type Reader interface {
	// GetCurrent returns the latest published snapshot. It may return a valid
	// snapshot together with an error wrapping basespec.ErrCatalogStale
	// when Collection, attachment, or Source metadata has changed.
	GetCurrent(
		ctx context.Context,
		ref collection.CollectionRef,
	) (Snapshot, error)
}

// ReadCurrent establishes the ownership, identity, and validation guarantees
// required by catalog consumers. A stale catalog remains readable because it
// is still useful for diagnostics and refresh reconciliation.
func ReadCurrent(
	ctx context.Context,
	reader Reader,
	ref collection.CollectionRef,
) (Snapshot, error) {
	if reader == nil {
		return Snapshot{}, fmt.Errorf(
			"%w: catalog reader is nil",
			basespec.ErrInvalid,
		)
	}
	if ctx == nil {
		return Snapshot{}, fmt.Errorf(
			"%w: catalog read context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if err := ref.Validate(); err != nil {
		return Snapshot{}, err
	}

	value, err := reader.GetCurrent(ctx, ref)
	if err != nil && !errors.Is(err, basespec.ErrCatalogStale) {
		return Snapshot{}, err
	}
	if value.RootID != ref.RootID || value.CollectionID != ref.CollectionID {
		return Snapshot{}, fmt.Errorf(
			"%w: catalog reader returned a catalog for another collection",
			basespec.ErrInvalid,
		)
	}
	if validateErr := value.Validate(); validateErr != nil {
		return Snapshot{}, fmt.Errorf(
			"%w: catalog reader returned an invalid catalog: %w",
			basespec.ErrInvalid,
			validateErr,
		)
	}
	return CloneSnapshot(value), err
}
