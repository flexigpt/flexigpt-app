package root

import (
	"context"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

type Repository interface {
	Create(
		ctx context.Context,
		value rootModel.Root,
	) error

	Get(
		ctx context.Context,
		id rootModel.RootID,
	) (rootModel.Root, error)

	List(ctx context.Context) ([]rootModel.Root, error)

	Update(
		ctx context.Context,
		value rootModel.Root,
		expectedRevision uint64,
	) error

	Retire(
		ctx context.Context,
		value rootModel.Root,
		expectedRevision uint64,
	) error

	Purge(
		ctx context.Context,
		id rootModel.RootID,
		expectedRevision uint64,
	) error
}
