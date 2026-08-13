package source

import (
	"context"
	"encoding/json"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type Reader interface {
	Get(
		ctx context.Context,
		rootID basespec.RootID,
		id basespec.SourceID,
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
		rootID basespec.RootID,
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
		rootID basespec.RootID,
		id basespec.SourceID,
		expectedRevision uint64,
	) error

	Purge(
		ctx context.Context,
		rootID basespec.RootID,
		id basespec.SourceID,
		expectedRevision uint64,
	) error
}

type Snapshot interface {
	Generation() string

	Stat(
		ctx context.Context,
		locator basespec.Locator,
	) (Entry, error)

	ReadDir(
		ctx context.Context,
		locator basespec.Locator,
	) ([]Entry, error)

	Open(
		ctx context.Context,
		locator basespec.Locator,
	) (io.ReadCloser, error)

	Confirm(ctx context.Context) error
	Close() error
}

type Opener interface {
	Open(
		ctx context.Context,
		value Source,
	) (Snapshot, error)
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
		locator basespec.Locator,
	) (string, error)
}

// LocalPathCapability advertises which Source kinds can provide trusted
// native paths. It avoids consumer hard-coding of concrete adapter kinds.
type LocalPathCapability interface {
	SupportsLocalPath(
		kind basespec.SourceKind,
	) bool
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
		address ManagedPackageAddress,
		expectedGeneration string,
	) error
}

// ManagedSourceBootstrapper is an optional capability implemented by writable
// Source adapters that need to establish physical Source storage before the
// Source metadata row is published.
//
// DiscardBootstrappedManagedSource is used for failed source provisioning and
// for discarding a bundle-owned Source after all published packages were
// removed. It must refuse to remove published package content, but may remove
// adapter-private staging state.
type ManagedSourceBootstrapper interface {
	BootstrapManagedSource(
		ctx context.Context,
		value Source,
	) error

	DiscardBootstrappedManagedSource(
		ctx context.Context,
		value Source,
	) error
}

// ManagedRootRemover is a trusted adapter capability used only when an
// application-owned topology is being replaced as a whole. It removes all
// managed Source storage below one Root, including packages that are no longer
// declared by a newer binary.
//
// It must never be exposed through public Source APIs.
type ManagedRootRemover interface {
	RemoveManagedRoot(
		ctx context.Context,
		rootStorageKey basespec.StorageKey,
	) error
}

type Adapter interface {
	Kind() basespec.SourceKind

	NormalizeConfig(
		ctx context.Context,
		raw json.RawMessage,
	) (json.RawMessage, error)

	Open(
		ctx context.Context,
		value Source,
	) (Snapshot, error)
}
