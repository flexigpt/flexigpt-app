package rootimpl

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
)

type Repository interface {
	Create(
		ctx context.Context,
		value root.Root,
	) error

	Get(
		ctx context.Context,
		id basespec.RootID,
	) (root.Root, error)

	List(ctx context.Context) ([]root.Root, error)

	Update(
		ctx context.Context,
		value root.Root,
		expectedRevision uint64,
	) error

	Retire(
		ctx context.Context,
		value root.Root,
		expectedRevision uint64,
	) error

	Purge(
		ctx context.Context,
		id basespec.RootID,
		expectedRevision uint64,
	) error
}
