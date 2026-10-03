package root

import (
	"context"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

type API interface {
	Create(
		ctx context.Context,
		draft rootModel.RootDraft,
	) (rootModel.Root, error)

	Get(
		ctx context.Context,
		id rootModel.RootID,
	) (rootModel.Root, error)

	List(ctx context.Context) ([]rootModel.Root, error)

	Update(
		ctx context.Context,
		id rootModel.RootID,
		update rootModel.RootUpdate,
	) (rootModel.Root, error)

	Retire(
		ctx context.Context,
		id rootModel.RootID,
		expectedRevision uint64,
	) (rootModel.Root, error)

	Purge(
		ctx context.Context,
		id rootModel.RootID,
		expectedRevision uint64,
	) error
}
