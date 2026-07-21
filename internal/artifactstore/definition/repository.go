package definition

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Reader interface {
	Get(
		ctx context.Context,
		digest artifactstore.Digest,
	) (Definition, error)
}

type Writer interface {
	Put(
		ctx context.Context,
		value Definition,
	) (Definition, error)
}

type Repository interface {
	Reader
	Writer
}
