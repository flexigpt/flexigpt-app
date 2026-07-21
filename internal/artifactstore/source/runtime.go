package source

import (
	"context"
	"errors"

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
		return nil, errors.New(
			"source runtime dependencies are incomplete",
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
	return r.reader.Get(ctx, id)
}

func (r *runtime) Open(
	ctx context.Context,
	value Source,
) (Snapshot, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}
	return r.opener.Open(ctx, value)
}

var _ Runtime = (*runtime)(nil)
