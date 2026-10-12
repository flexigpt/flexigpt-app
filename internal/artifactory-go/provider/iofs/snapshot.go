package iofs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync/atomic"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type snapshot struct {
	provider   fs.FS
	generation string
	immutable  bool
	closed     atomic.Bool
}

func (s *snapshot) Generation() string {
	return s.generation
}

func (s *snapshot) Stat(
	ctx context.Context,
	locator storeSpec.Locator,
) (sourceModel.Entry, error) {
	if err := s.ensureOpen(); err != nil {
		return sourceModel.Entry{}, err
	}
	name, err := fsName(locator)
	if err != nil {
		return sourceModel.Entry{}, err
	}
	info, err := fs.Stat(s.provider, name)
	if errors.Is(err, fs.ErrNotExist) {
		return sourceModel.Entry{}, fmt.Errorf(
			"%w: embedded locator %q",
			storeSpec.ErrNotFound,
			locator,
		)
	}
	if err != nil {
		return sourceModel.Entry{}, err
	}
	return entryFromInfo(locator, info), nil
}

func (s *snapshot) ReadDir(
	ctx context.Context,
	locator storeSpec.Locator,
) ([]sourceModel.Entry, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	name, err := fsName(locator)
	if err != nil {
		return nil, err
	}
	values, err := fs.ReadDir(s.provider, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf(
			"%w: embedded directory %q",
			storeSpec.ErrNotFound,
			locator,
		)
	}
	if err != nil {
		return nil, err
	}

	output := make([]sourceModel.Entry, 0, len(values))
	for _, value := range values {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		child, err := joinLocator(locator, value.Name())
		if err != nil {
			return nil, err
		}
		info, err := fs.Stat(s.provider, string(child))
		if err != nil {
			return nil, err
		}
		output = append(output, entryFromInfo(child, info))
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].Locator < output[right].Locator
	})
	return output, nil
}

func (s *snapshot) Open(
	ctx context.Context,
	locator storeSpec.Locator,
) (io.ReadCloser, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	name, err := fsName(locator)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(s.provider, name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf(
			"%w: embedded locator %q is not a regular file",
			storeSpec.ErrInvalid,
			locator,
		)
	}
	file, err := s.provider.Open(name)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (s *snapshot) Confirm(ctx context.Context) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if s.immutable {
		return nil
	}
	current, err := fingerprint(ctx, s.provider)
	if err != nil {
		return err
	}
	if current != s.generation {
		return fmt.Errorf(
			"%w: embedded source changed during discovery",
			storeSpec.ErrConflict,
		)
	}
	return nil
}

func (s *snapshot) Close() error {
	s.closed.Store(true)
	return nil
}

func (s *snapshot) ensureOpen() error {
	if s.closed.Load() {
		return storeSpec.ErrClosed
	}
	return nil
}

func fsName(locator storeSpec.Locator) (string, error) {
	if err := locator.Validate(true); err != nil {
		return "", err
	}
	if locator == "." {
		return ".", nil
	}
	return string(locator), nil
}

func joinLocator(
	parent storeSpec.Locator,
	name string,
) (storeSpec.Locator, error) {
	if name == "" || strings.Contains(name, "/") || !fs.ValidPath(name) {
		return "", fmt.Errorf(
			"%w: invalid embedded entry name %q",
			storeSpec.ErrInvalid,
			name,
		)
	}
	if parent == "." {
		return storeSpec.Locator(name), nil
	}
	return storeSpec.Locator(path.Join(string(parent), name)), nil
}

func entryFromInfo(
	locator storeSpec.Locator,
	info fs.FileInfo,
) sourceModel.Entry {
	name := info.Name()
	if locator != "." {
		// Entry identity is source-relative and must not depend on a provider
		// returning a target or implementation-specific FileInfo name.
		name = path.Base(string(locator))
	}
	return sourceModel.Entry{
		Locator:     locator,
		Name:        name,
		SizeBytes:   info.Size(),
		Mode:        uint32(info.Mode()),
		ModifiedAt:  info.ModTime().UTC(),
		IsDirectory: info.IsDir(),
		IsRegular:   info.Mode().IsRegular(),
	}
}
