package catalog

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Reader interface {
	GetCurrent(
		ctx context.Context,
		rootID artifactstore.RootID,
	) (Snapshot, error)
}
