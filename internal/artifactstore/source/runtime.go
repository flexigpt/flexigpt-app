package source

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// Runtime is a trusted internal capability for consumers that need an
// operational source, including its normalized adapter configuration.
//
// It is intentionally separate from Service, whose query methods return
// Summary values and do not expose opaque source configuration.
type Runtime interface {
	Get(
		ctx context.Context,
		rootID basespec.RootID,
		id basespec.SourceID,
	) (Source, error)

	Open(
		ctx context.Context,
		value Source,
	) (Snapshot, error)
}

// LocalPathRuntime is an optional extension implemented by the trusted source
// runtime when its opener supports LocalPathResolver.
//
// It intentionally accepts a full Source rather than a public Summary because
// source configuration remains internal to Artifact Store consumers.
type LocalPathRuntime interface {
	ResolveLocalPath(
		ctx context.Context,
		value Source,
		locator basespec.Locator,
	) (string, error)

	SupportsLocalPath(
		kind basespec.SourceKind,
	) bool
}

type runtime struct {
	reader     Reader
	opener     Opener
	localPaths LocalPathResolver
	localKinds LocalPathCapability
}

func NewRuntime(
	reader Reader,
	opener Opener,
) (Runtime, error) {
	if reader == nil || opener == nil {
		return nil, fmt.Errorf(
			"%w: source runtime dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	value := &runtime{
		reader: reader,
		opener: opener,
	}
	if resolver, supported := opener.(LocalPathResolver); supported {
		value.localPaths = resolver
	}
	if capabilities, supported := opener.(LocalPathCapability); supported {
		value.localKinds = capabilities
	}
	return value, nil
}

// ReadSnapshotEntry reads one regular Source snapshot entry with the same
// bounded-read and size-stability rules used by discovery.
//
// It intentionally operates only through Snapshot. Feature code must not
// recreate this behavior by reading a Source's native filesystem path.
func ReadSnapshotEntry(
	ctx context.Context,
	snapshot Snapshot,
	entry Entry,
	maximumBytes int64,
) ([]byte, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: source snapshot read context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, fmt.Errorf(
			"%w: source snapshot is nil",
			basespec.ErrInvalid,
		)
	}
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if !entry.IsRegular {
		return nil, fmt.Errorf(
			"%w: source entry %q is not a regular file",
			basespec.ErrInvalid,
			entry.Locator,
		)
	}
	if maximumBytes <= 0 || maximumBytes > basespec.MaxScanBytes {
		return nil, fmt.Errorf(
			"%w: source snapshot read limit is invalid",
			basespec.ErrInvalid,
		)
	}
	if entry.SizeBytes > maximumBytes {
		return nil, fmt.Errorf(
			"%w: source entry %q exceeds byte limit",
			basespec.ErrInvalid,
			entry.Locator,
		)
	}

	reader, err := snapshot.Open(ctx, entry.Locator)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, fmt.Errorf(
			"%w: source snapshot returned a nil reader for %q",
			basespec.ErrInvalid,
			entry.Locator,
		)
	}
	content, readErr := io.ReadAll(
		io.LimitReader(reader, maximumBytes+1),
	)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if int64(len(content)) > maximumBytes {
		return nil, fmt.Errorf(
			"%w: source entry %q exceeds byte limit",
			basespec.ErrInvalid,
			entry.Locator,
		)
	}
	if int64(len(content)) != entry.SizeBytes {
		return nil, fmt.Errorf(
			"%w: source entry %q changed size during snapshot read",
			basespec.ErrConflict,
			entry.Locator,
		)
	}
	return content, nil
}

// ReadVerifiedSnapshotEntry reads one source entry from an exact Source
// generation and confirms the snapshot before returning the owned bytes and
// their digest.
//
// It is the generic Artifact Store boundary for feature code that needs the
// canonical source document itself. Features must not reopen native paths or
// duplicate snapshot generation, bounded-read, confirmation, and close rules.
func ReadVerifiedSnapshotEntry(
	ctx context.Context,
	runtime Runtime,
	value Source,
	locator basespec.Locator,
	expectedGeneration string,
	maximumBytes int64,
) (
	content []byte,
	digest cryptoutil.Digest,
	returnErr error,
) {
	if ctx == nil {
		return nil, "", fmt.Errorf(
			"%w: verified source read context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if runtime == nil {
		return nil, "", fmt.Errorf(
			"%w: verified source read runtime is nil",
			basespec.ErrInvalid,
		)
	}
	if err := value.Validate(); err != nil {
		return nil, "", err
	}
	if err := basespec.ValidateLocator(locator, false); err != nil {
		return nil, "", err
	}
	if err := basespec.ValidateSourceGeneration(expectedGeneration); err != nil {
		return nil, "", err
	}
	if maximumBytes <= 0 || maximumBytes > basespec.MaxScanBytes {
		return nil, "", fmt.Errorf(
			"%w: verified source read limit is invalid",
			basespec.ErrInvalid,
		)
	}

	snapshot, err := runtime.Open(ctx, value)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		returnErr = errors.Join(returnErr, snapshot.Close())
	}()

	if snapshot.Generation() != expectedGeneration {
		return nil, "", fmt.Errorf(
			"%w: source generation changed since it was observed",
			basespec.ErrConflict,
		)
	}
	content, err = readSnapshotLocator(
		ctx,
		snapshot,
		locator,
		maximumBytes,
	)
	if err != nil {
		return nil, "", err
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return nil, "", err
	}
	return content, cryptoutil.DigestBytes(content), nil
}

// VerifySnapshotContentDigest confirms both the Source generation and the
// exact bytes of one catalogued Source entry through the Source runtime.
//
// This is intentionally generic Source behavior. It does not parse Skill
// content, resolve native paths, or apply runtime sandbox policy.
func VerifySnapshotContentDigest(
	ctx context.Context,
	runtime Runtime,
	value Source,
	locator basespec.Locator,
	expectedGeneration string,
	expectedDigest cryptoutil.Digest,
	maximumBytes int64,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: source digest verification context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if runtime == nil {
		return fmt.Errorf(
			"%w: source digest verification runtime is nil",
			basespec.ErrInvalid,
		)
	}
	if err := value.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateLocator(locator, false); err != nil {
		return err
	}
	if err := basespec.ValidateSourceGeneration(expectedGeneration); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(expectedDigest); err != nil {
		return err
	}
	if maximumBytes <= 0 || maximumBytes > basespec.MaxCandidateBytes {
		return fmt.Errorf(
			"%w: source digest verification limit is invalid",
			basespec.ErrInvalid,
		)
	}

	_, actualDigest, err := ReadVerifiedSnapshotEntry(
		ctx,
		runtime,
		value,
		locator,
		expectedGeneration,
		maximumBytes,
	)
	if err != nil {
		return err
	}
	if actualDigest != expectedDigest {
		return fmt.Errorf(
			"%w: source content for %q changed since catalog publication",
			basespec.ErrConflict,
			locator,
		)
	}
	return nil
}

func (r *runtime) Get(
	ctx context.Context,
	rootID basespec.RootID,
	id basespec.SourceID,
) (Source, error) {
	if r == nil || r.reader == nil {
		return Source{}, basespec.ErrClosed
	}
	if ctx == nil {
		return Source{}, fmt.Errorf(
			"%w: source runtime context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return Source{}, err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return Source{}, err
	}
	if err := basespec.ValidateSourceID(id); err != nil {
		return Source{}, err
	}
	value, err := r.reader.Get(ctx, rootID, id)
	if err != nil {
		return Source{}, err
	}
	if value.ID != id {
		return Source{}, fmt.Errorf(
			"%w: source reader returned %q for requested source %q",
			basespec.ErrInvalid,
			value.ID,
			id,
		)
	}
	if value.RootID != rootID {
		return Source{}, fmt.Errorf(
			"%w: source reader returned root %q for requested root %q",
			basespec.ErrInvalid,
			value.RootID,
			rootID,
		)
	}
	if err := value.Validate(); err != nil {
		return Source{}, fmt.Errorf("invalid source returned by runtime reader: %w", err)
	}
	return value.Clone(), nil
}

func (r *runtime) Open(
	ctx context.Context,
	value Source,
) (Snapshot, error) {
	if r == nil || r.opener == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: source runtime context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := value.Validate(); err != nil {
		return nil, err
	}
	snapshot, err := r.opener.Open(ctx, value.Clone())
	if err != nil {
		return nil, err
	}
	if err := validateSnapshot(snapshot); err != nil {
		if snapshot != nil {
			_ = snapshot.Close()
		}
		return nil, err
	}
	return snapshot, nil
}

// ResolveLocalPath delegates only to adapters that explicitly support native
// filesystem paths. Non-filesystem sources remain source-backed but do not
// become path-backed implicitly.
func (r *runtime) ResolveLocalPath(
	ctx context.Context,
	value Source,
	locator basespec.Locator,
) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("%w: source local-path context is nil", basespec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := value.Validate(); err != nil {
		return "", err
	}
	if err := basespec.ValidateLocator(locator, true); err != nil {
		return "", err
	}
	if r.localPaths == nil ||
		!r.SupportsLocalPath(value.Kind) {
		return "", fmt.Errorf(
			"%w: source runtime has no native path resolver",
			basespec.ErrUnsupported,
		)
	}
	location, err := r.localPaths.ResolveLocalPath(
		ctx,
		value.Clone(),
		locator,
	)
	if err != nil {
		return "", err
	}
	return location, nil
}

func (r *runtime) SupportsLocalPath(
	kind basespec.SourceKind,
) bool {
	if r == nil || r.localKinds == nil {
		return false
	}
	return r.localKinds.SupportsLocalPath(kind)
}

func validateSnapshot(snapshot Snapshot) error {
	if snapshot == nil {
		return fmt.Errorf("%w: source opener returned a nil snapshot", basespec.ErrInvalid)
	}
	if err := basespec.ValidateSourceGeneration(snapshot.Generation()); err != nil {
		return fmt.Errorf(
			"%w: source snapshot returned an invalid generation: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return nil
}
