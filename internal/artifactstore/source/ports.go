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
