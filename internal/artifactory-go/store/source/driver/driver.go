package driver

import (
	"context"
	"encoding/json"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
)

// Driver owns physical source configuration normalization and snapshot opening.
//
// It does not own Source persistence, Source lifecycle, discovery policy,
// package orchestration, or Artifact synchronization.
type Driver interface {
	Kind() sourceModel.SourceKind

	NormalizeConfig(
		ctx context.Context,
		raw json.RawMessage,
	) (json.RawMessage, error)

	Open(
		ctx context.Context,
		value sourceModel.Source,
	) (Snapshot, error)
}

type Opener interface {
	Open(
		ctx context.Context,
		value sourceModel.Source,
	) (Snapshot, error)
}
