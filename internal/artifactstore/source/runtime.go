package source

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

// Runtime is a trusted internal capability for consumers that need an
// operational source, including its normalized adapter configuration.
//
// It is intentionally separate from Service, whose query methods return
// Summary values and do not expose opaque source configuration.
type Runtime interface {
	Get(
		ctx context.Context,
		id artifactstore.SourceID,
	) (Source, error)

	Open(
		ctx context.Context,
		value Source,
	) (Snapshot, error)
}

type runtime struct {
	reader Reader
	opener Opener
}

func NewRuntime(
	reader Reader,
	opener Opener,
) (Runtime, error) {
	if reader == nil || opener == nil {
		return nil, fmt.Errorf(
			"%w: source runtime dependencies are incomplete",
			artifactstore.ErrInvalid,
		)
	}
	return &runtime{
		reader: reader,
		opener: opener,
	}, nil
}

func (r *runtime) Get(
	ctx context.Context,
	id artifactstore.SourceID,
) (Source, error) {
	if err := artifactstore.ValidateSourceID(id); err != nil {
		return Source{}, err
	}
	value, err := r.reader.Get(ctx, id)
	if err != nil {
		return Source{}, err
	}
	if value.ID != id {
		return Source{}, fmt.Errorf(
			"%w: source reader returned %q for requested source %q",
			artifactstore.ErrInvalid,
			value.ID,
			id,
		)
	}
	if err := value.Validate(); err != nil {
		return Source{}, fmt.Errorf("invalid source returned by runtime reader: %w", err)
	}
	return value.Clone(), nil
}

func (r *runtime) Open(
	ctx context.Context,
	value Source,
) (Snapshot, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	snapshot, err := r.opener.Open(ctx, value.Clone())
	if err != nil {
		return nil, err
	}
	if err := validateSnapshot(snapshot); err != nil {
		_ = snapshot.Close()
		return nil, err
	}
	return snapshot, nil
}

func validateSnapshot(snapshot Snapshot) error {
	if snapshot == nil {
		return fmt.Errorf("%w: source opener returned a nil snapshot", artifactstore.ErrInvalid)
	}
	if err := artifactstore.ValidateSourceGeneration(snapshot.Generation()); err != nil {
		return fmt.Errorf(
			"%w: source snapshot returned an invalid generation: %w",
			artifactstore.ErrInvalid,
			err,
		)
	}
	return nil
}
