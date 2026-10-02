package rootimpl

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
)

type Repository interface {
	Create(
		ctx context.Context,
		value root.Root,
	) error

	Get(
		ctx context.Context,
		id root.RootID,
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
		id root.RootID,
		expectedRevision uint64,
	) error
}
