package source

import (
	"context"
	"errors"
	"fmt"
	"io"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type runtime struct {
	reader     Reader
	opener     driver.Opener
	localPaths driver.LocalPathResolver
	localKinds driver.LocalPathCapability
}

func NewRuntime(reader Reader, opener driver.Opener) (Runtime, error) {
	if reader == nil || opener == nil {
		return nil, fmt.Errorf("%w: source runtime dependencies are incomplete", spec.ErrInvalid)
	}
	value := &runtime{reader: reader, opener: opener}
	if resolver, supported := opener.(driver.LocalPathResolver); supported {
		value.localPaths = resolver
	}
	if capabilities, supported := opener.(driver.LocalPathCapability); supported {
		value.localKinds = capabilities
	}
	return value, nil
}

// ReadSnapshotEntry reads one regular Source snapshot entry with the same
// bounded-read and size-stability rules used by discovery.
func ReadSnapshotEntry(
	ctx context.Context,
	snapshot driver.Snapshot,
	entry sourceModel.Entry,
	maximumBytes int64,
) ([]byte, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("%w: source snapshot is nil", spec.ErrInvalid)
	}
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if !entry.IsRegular {
		return nil, fmt.Errorf("%w: source entry %q is not a regular file", spec.ErrInvalid, entry.Locator)
	}
	if maximumBytes <= 0 || maximumBytes > spec.MaxScanBytes {
		return nil, fmt.Errorf("%w: source snapshot read limit is invalid", spec.ErrInvalid)
	}
	if entry.SizeBytes > maximumBytes {
		return nil, fmt.Errorf("%w: source entry %q exceeds byte limit", spec.ErrInvalid, entry.Locator)
	}
	reader, err := snapshot.Open(ctx, entry.Locator)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, fmt.Errorf("%w: source snapshot returned a nil reader for %q", spec.ErrInvalid, entry.Locator)
	}
	content, readErr := io.ReadAll(io.LimitReader(reader, maximumBytes+1))
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if int64(len(content)) > maximumBytes {
		return nil, fmt.Errorf("%w: source entry %q exceeds byte limit", spec.ErrInvalid, entry.Locator)
	}
	if int64(len(content)) != entry.SizeBytes {
		return nil, fmt.Errorf("%w: source entry %q changed size during snapshot read", spec.ErrConflict, entry.Locator)
	}
	return content, nil
}

// ReadVerifiedSnapshotEntry reads one source entry from an exact Source
// generation and confirms the snapshot before returning owned bytes and digest.
func ReadVerifiedSnapshotEntry(
	ctx context.Context,
	runtime Runtime,
	value sourceModel.Source,
	locator spec.Locator,
	expectedGeneration string,
	maximumBytes int64,
) (content []byte, digest cryptoutil.Digest, returnErr error) {
	if err := value.ValidateRead(); err != nil {
		return nil, "", err
	}
	if err := locator.Validate(false); err != nil {
		return nil, "", err
	}
	if err := spec.ValidateSourceGeneration(expectedGeneration); err != nil {
		return nil, "", err
	}
	if maximumBytes <= 0 || maximumBytes > spec.MaxScanBytes {
		return nil, "", fmt.Errorf("%w: verified source read limit is invalid", spec.ErrInvalid)
	}
	snapshot, err := runtime.Open(ctx, value)
	if err != nil {
		return nil, "", err
	}
	defer func() { returnErr = errors.Join(returnErr, snapshot.Close()) }()
	if snapshot.Generation() != expectedGeneration {
		return nil, "", fmt.Errorf("%w: source generation changed since it was observed", spec.ErrConflict)
	}
	content, err = readSnapshotLocator(ctx, snapshot, locator, maximumBytes)
	if err != nil {
		return nil, "", err
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return nil, "", err
	}
	return content, cryptoutil.DigestBytes(content), nil
}

// VerifySnapshotContentDigest confirms both Source generation and exact bytes
// of one refreshed Source entry through the Source runtime.
func VerifySnapshotContentDigest(
	ctx context.Context,
	runtime Runtime,
	value sourceModel.Source,
	locator spec.Locator,
	expectedGeneration string,
	expectedDigest cryptoutil.Digest,
	maximumBytes int64,
) error {
	if err := cryptoutil.ValidateDigest(expectedDigest); err != nil {
		return err
	}
	if maximumBytes <= 0 || maximumBytes > spec.MaxCandidateBytes {
		return fmt.Errorf("%w: source digest verification limit is invalid", spec.ErrInvalid)
	}
	_, actualDigest, err := ReadVerifiedSnapshotEntry(ctx, runtime, value, locator, expectedGeneration, maximumBytes)
	if err != nil {
		return err
	}
	if actualDigest != expectedDigest {
		return fmt.Errorf("%w: Source content for %q changed since refresh", spec.ErrConflict, locator)
	}
	return nil
}

func (r *runtime) Get(
	ctx context.Context,
	rootID rootModel.RootID,
	id sourceModel.SourceID,
) (sourceModel.Source, error) {
	value, err := r.reader.Get(ctx, rootID, id)
	if err != nil {
		return sourceModel.Source{}, err
	}
	if value.ID != id || value.RootID != rootID {
		return sourceModel.Source{}, fmt.Errorf("%w: source reader returned another Source identity", spec.ErrInvalid)
	}
	if err := value.ValidateRead(); err != nil {
		return sourceModel.Source{}, fmt.Errorf("invalid source returned by runtime reader: %w", err)
	}
	return value.Clone(), nil
}

func (r *runtime) List(ctx context.Context, rootID rootModel.RootID) ([]sourceModel.Source, error) {
	values, err := r.reader.List(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]sourceModel.Source, len(values))
	for index, value := range values {
		if value.RootID != rootID {
			return nil, fmt.Errorf("%w: source reader returned Source %q for another Root", spec.ErrInvalid, value.ID)
		}
		if err := value.ValidateRead(); err != nil {
			return nil, err
		}
		output[index] = value.Clone()
	}
	return output, nil
}

func (r *runtime) Open(ctx context.Context, value sourceModel.Source) (driver.Snapshot, error) {
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

func (r *runtime) ResolveLocalPath(
	ctx context.Context,
	value sourceModel.Source,
	locator spec.Locator,
) (string, error) {
	if r.localPaths == nil || !r.SupportsLocalPath(value.Kind) {
		return "", fmt.Errorf("%w: source runtime has no native path resolver", spec.ErrUnsupported)
	}
	return r.localPaths.ResolveLocalPath(ctx, value.Clone(), locator)
}

func (r *runtime) SupportsLocalPath(kind sourceModel.SourceKind) bool {
	return r.localKinds != nil && r.localKinds.SupportsLocalPath(kind)
}

func validateSnapshot(snapshot driver.Snapshot) error {
	if snapshot == nil {
		return fmt.Errorf("%w: source opener returned a nil snapshot", spec.ErrInvalid)
	}
	if err := spec.ValidateSourceGeneration(snapshot.Generation()); err != nil {
		return fmt.Errorf("%w: source snapshot returned an invalid generation: %w", spec.ErrInvalid, err)
	}
	return nil
}
