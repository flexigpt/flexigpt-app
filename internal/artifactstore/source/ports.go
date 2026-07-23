package source

import (
	"context"
	"encoding/json"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Reader interface {
	Get(ctx context.Context, id artifactstore.SourceID) (Source, error)
}

type Repository interface {
	Reader

	Create(
		ctx context.Context,
		value Source,
	) error

	List(ctx context.Context) ([]Source, error)
	Update(
		ctx context.Context,
		value Source,
		expectedRevision uint64,
	) error
	Delete(
		ctx context.Context,
		id artifactstore.SourceID,
		expectedRevision uint64,
	) error
}

type Opener interface {
	Open(
		ctx context.Context,
		value Source,
	) (Snapshot, error)
}

type Snapshot interface {
	Generation() string

	Stat(
		ctx context.Context,
		locator artifactstore.Locator,
	) (Entry, error)

	ReadDir(
		ctx context.Context,
		locator artifactstore.Locator,
	) ([]Entry, error)

	Open(
		ctx context.Context,
		locator artifactstore.Locator,
	) (io.ReadCloser, error)

	Confirm(ctx context.Context) error
	Close() error
}

// LocalPathResolver is an optional trusted-internal capability exposed by a
// source adapter when a source-relative locator has a native local filesystem
// representation.
//
// It is deliberately not part of Snapshot and is never exposed through public
// source summaries or Workspace API views. Consumers use it only after their
// own runtime policy has approved a selected record.
type LocalPathResolver interface {
	ResolveLocalPath(
		ctx context.Context,
		value Source,
		locator artifactstore.Locator,
	) (string, error)
}

type Adapter interface {
	Kind() artifactstore.SourceKind

	NormalizeConfig(
		ctx context.Context,
		raw json.RawMessage,
	) (json.RawMessage, error)

	Open(
		ctx context.Context,
		value Source,
	) (Snapshot, error)
}
