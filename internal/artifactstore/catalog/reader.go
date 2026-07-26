package catalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Reader interface {
	// GetCurrent returns the latest published snapshot. It may return a valid
	// snapshot together with an error wrapping artifactstore.ErrCatalogStale
	// when Collection, attachment, or Source metadata has changed.
	GetCurrent(
		ctx context.Context,
		ref artifactstore.CollectionRef,
	) (Snapshot, error)
}

// ReadCurrent establishes the ownership, identity, and validation guarantees
// required by catalog consumers. A stale catalog remains readable because it
// is still useful for diagnostics and refresh reconciliation.
func ReadCurrent(
	ctx context.Context,
	reader Reader,
	ref artifactstore.CollectionRef,
) (Snapshot, error) {
	if reader == nil {
		return Snapshot{}, fmt.Errorf(
			"%w: catalog reader is nil",
			artifactstore.ErrInvalid,
		)
	}
	if ctx == nil {
		return Snapshot{}, fmt.Errorf(
			"%w: catalog read context is nil",
			artifactstore.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if err := ref.Validate(); err != nil {
		return Snapshot{}, err
	}

	value, err := reader.GetCurrent(ctx, ref)
	if err != nil && !errors.Is(err, artifactstore.ErrCatalogStale) {
		return Snapshot{}, err
	}
	if value.RootID != ref.RootID || value.CollectionID != ref.CollectionID {
		return Snapshot{}, fmt.Errorf(
			"%w: catalog reader returned a catalog for another collection",
			artifactstore.ErrInvalid,
		)
	}
	if validateErr := value.Validate(); validateErr != nil {
		return Snapshot{}, fmt.Errorf(
			"%w: catalog reader returned an invalid catalog: %w",
			artifactstore.ErrInvalid,
			validateErr,
		)
	}
	return CloneSnapshot(value), err
}
