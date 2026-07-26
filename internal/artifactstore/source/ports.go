package source

import (
	"context"
	"encoding/json"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Reader interface {
	Get(
		ctx context.Context,
		rootID artifactstore.RootID,
		id artifactstore.SourceID,
	) (Source, error)
}

type Repository interface {
	Reader

	Create(
		ctx context.Context,
		value Source,
	) error

	List(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]Source, error)

	Update(
		ctx context.Context,
		value Source,
		expectedRevision uint64,
	) error

	Retire(
		ctx context.Context,
		value Source,
		expectedRevision uint64,
	) error

	Discard(
		ctx context.Context,
		rootID artifactstore.RootID,
		id artifactstore.SourceID,
		expectedRevision uint64,
	) error

	Purge(
		ctx context.Context,
		rootID artifactstore.RootID,
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

// LocalPathResolver is an optional trusted internal capability exposed by a
// Source adapter when a source-relative locator has a native local filesystem
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

// LocalPathCapability advertises which Source kinds can provide trusted
// native paths. It avoids consumer hard-coding of concrete adapter kinds.
type LocalPathCapability interface {
	SupportsLocalPath(
		kind artifactstore.SourceKind,
	) bool
}

// ManagedPackageFile is one regular file relative to a managed package
// directory. Empty directories are deliberately not part of the managed
// package contract.
type ManagedPackageFile struct {
	Locator artifactstore.Locator `json:"locator"`
	Content []byte                `json:"content"`
}

// ManagedPackagePublication atomically publishes one new package directory
// inside a managed Source.
//
// ExpectedGeneration is optional. When supplied, it prevents publication
// against a Source snapshot other than the one observed by the caller.
// Repeating an already completed, byte-identical publication is idempotent
// even when ExpectedGeneration names the generation before that publication.
type ManagedPackagePublication struct {
	Directory          artifactstore.Locator `json:"directory"`
	ExpectedGeneration string                `json:"expectedGeneration,omitempty"`
	Files              []ManagedPackageFile  `json:"files"`
}

// ManagedPackageWriter is the optional writable capability for an
// application-managed Source adapter.
//
// It deliberately publishes complete package directories rather than
// arbitrary individual file mutations. This gives managed authoring a staged,
// source-side atomic publication boundary without pretending that the Source
// filesystem and metadata database share one transaction.
type ManagedPackageWriter interface {
	PublishPackage(
		ctx context.Context,
		value Source,
		publication ManagedPackagePublication,
	) (generation string, err error)

	RemovePackage(
		ctx context.Context,
		value Source,
		directory artifactstore.Locator,
		expectedGeneration string,
	) error
}

// ManagedSourceBootstrapper is an optional capability implemented by writable
// Source adapters that need to establish physical Source storage before the
// Source metadata row is published.
//
// BootstrapManagedSource returns the initial snapshot generation that must be
// persisted with the new Source. DiscardBootstrappedManagedSource compensates
// only a failed metadata creation and must refuse to remove non-empty content.
type ManagedSourceBootstrapper interface {
	BootstrapManagedSource(
		ctx context.Context,
		value Source,
	) (generation string, err error)

	DiscardBootstrappedManagedSource(
		ctx context.Context,
		value Source,
	) error
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
