package materializer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

// CopyingSourceMaterializer copies any directory-like SourceDriver into an
// atomic DirectoryPublisher. It performs no direct filesystem operation.
type CopyingSourceMaterializer struct {
	publisher spec.DirectoryPublisher
}

func NewCopyingSourceMaterializer(
	publisher spec.DirectoryPublisher,
) (*CopyingSourceMaterializer, error) {
	if publisher == nil {
		return nil, fmt.Errorf("%w: directory publisher is nil", spec.ErrInvalidRequest)
	}
	return &CopyingSourceMaterializer{publisher: publisher}, nil
}

func (m *CopyingSourceMaterializer) Materialize(
	ctx context.Context,
	input spec.SourceMaterializationInput,
) (result spec.MaterializedSource, err error) {
	if m == nil || m.publisher == nil || input.Driver == nil {
		return spec.MaterializedSource{}, fmt.Errorf(
			"%w: source materializer is not configured",
			spec.ErrMaterializerUnavailable,
		)
	}
	if strings.TrimSpace(input.PublicationKey) == "" {
		return spec.MaterializedSource{}, fmt.Errorf(
			"%w: materialization publication key is empty",
			spec.ErrInvalidRequest,
		)
	}
	if input.Root == "" {
		input.Root = "."
	}
	applyMaterializationDefaults(&input)

	rootEntry, err := input.Driver.Stat(ctx, input.Source, input.Root)
	if err != nil {
		return spec.MaterializedSource{}, err
	}
	if !rootEntry.IsDirectory || rootEntry.IsSymlink {
		return spec.MaterializedSource{}, fmt.Errorf(
			"%w: materialization root %q is not a safe directory",
			spec.ErrInvalidRequest,
			input.Root,
		)
	}

	generation, err := input.Driver.Snapshot(ctx, input.Source)
	if err != nil {
		return spec.MaterializedSource{}, err
	}
	publication, err := m.publisher.BeginDirectoryPublication(
		ctx,
		input.PublicationKey,
		generation,
	)
	if err != nil {
		return spec.MaterializedSource{}, err
	}
	if publication == nil {
		return spec.MaterializedSource{}, fmt.Errorf(
			"%w: directory publisher returned a nil publication",
			spec.ErrMaterializerUnavailable,
		)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		abortErr := publication.Abort(context.WithoutCancel(ctx))
		if abortErr != nil {
			err = errors.Join(err, fmt.Errorf("abort directory publication: %w", abortErr))
		}
	}()

	err = input.Driver.Walk(ctx, input.Source, input.Root, func(
		ctx context.Context,
		entry spec.SourceEntry,
	) error {
		result.Entries++
		if result.Entries > input.MaxEntries {
			return fmt.Errorf(
				"%w: materialization exceeds %d entries",
				spec.ErrInvalidRequest,
				input.MaxEntries,
			)
		}
		if entry.IsSymlink {
			return fmt.Errorf(
				"%w: materialization refuses symlink %q",
				spec.ErrInvalidRequest,
				entry.Locator,
			)
		}
		relative, err := relativePortablePath(input.Root, entry.Locator)
		if err != nil {
			return err
		}
		switch {
		case entry.IsDirectory:
			return publication.MakeDirectory(ctx, relative, entry.Mode)
		case entry.IsRegular:
		default:
			return fmt.Errorf(
				"%w: materialization refuses special entry %q",
				spec.ErrInvalidRequest,
				entry.Locator,
			)
		}

		result.Files++
		if result.Files > input.MaxFiles {
			return fmt.Errorf(
				"%w: materialization exceeds %d files",
				spec.ErrInvalidRequest,
				input.MaxFiles,
			)
		}
		remaining := input.MaxBytes - result.Bytes
		if remaining < 0 || entry.SizeBytes < 0 || entry.SizeBytes > remaining {
			return fmt.Errorf(
				"%w: materialization exceeds %d bytes",
				spec.ErrInvalidRequest,
				input.MaxBytes,
			)
		}

		reader, err := input.Driver.Open(ctx, input.Source, entry.Locator)
		if err != nil {
			return err
		}
		readLimit := remaining
		const maximumInt64 = int64(^uint64(0) >> 1)
		if readLimit < maximumInt64 {
			readLimit++
		}
		limited := &io.LimitedReader{R: reader, N: readLimit}
		writeErr := publication.WriteFile(ctx, relative, entry.Mode, limited)
		closeErr := reader.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		consumed := readLimit - limited.N
		if consumed > remaining {
			return fmt.Errorf(
				"%w: materialization exceeds %d bytes",
				spec.ErrInvalidRequest,
				input.MaxBytes,
			)
		}
		if consumed != entry.SizeBytes {
			return fmt.Errorf(
				"%w: source entry %q changed size during materialization",
				spec.ErrConflict,
				entry.Locator,
			)
		}
		result.Bytes += consumed
		return nil
	})
	if err != nil {
		return spec.MaterializedSource{}, err
	}

	confirmed, err := input.Driver.Snapshot(ctx, input.Source)
	if err != nil {
		return spec.MaterializedSource{}, err
	}
	if confirmed != generation {
		return spec.MaterializedSource{}, fmt.Errorf(
			"%w: source %q changed during materialization",
			spec.ErrConflict,
			input.Source.SourceID,
		)
	}

	rootPath, err := publication.Commit(ctx)
	if err != nil {
		return spec.MaterializedSource{}, err
	}
	if strings.TrimSpace(rootPath) == "" {
		return spec.MaterializedSource{}, fmt.Errorf(
			"%w: directory publication returned an empty root path",
			spec.ErrMaterializerUnavailable,
		)
	}
	committed = true
	result.PublicationKey = input.PublicationKey
	result.RootPath = rootPath
	result.Generation = generation
	return result, nil
}

func applyMaterializationDefaults(input *spec.SourceMaterializationInput) {
	if input.MaxEntries <= 0 {
		input.MaxEntries = spec.DefaultMaxMaterializedEntries
	}
	if input.MaxFiles <= 0 {
		input.MaxFiles = spec.DefaultMaxMaterializedFiles
	}
	if input.MaxBytes <= 0 {
		input.MaxBytes = spec.DefaultMaxMaterializedBytes
	}
}

func relativePortablePath(
	root spec.SourceLocator,
	locator spec.SourceLocator,
) (spec.PortablePath, error) {
	value := string(locator)
	if root != "." {
		prefix := string(root) + "/"
		if !strings.HasPrefix(value, prefix) {
			return "", fmt.Errorf(
				"%w: locator %q is outside materialization root %q",
				spec.ErrInvalidRequest,
				locator,
				root,
			)
		}
		value = strings.TrimPrefix(value, prefix)
	}
	if value == "" || value == "." || path.Clean(value) != value ||
		strings.Contains(value, "\\") || strings.Contains(value, ":") ||
		strings.HasPrefix(value, "../") {
		return "", fmt.Errorf(
			"%w: invalid materialized relative path %q",
			spec.ErrInvalidRequest,
			value,
		)
	}
	return spec.PortablePath(value), nil
}

var _ spec.SourceMaterializer = (*CopyingSourceMaterializer)(nil)
