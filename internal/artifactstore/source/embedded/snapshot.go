package embedded

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type snapshot struct {
	provider   fs.FS
	generation string
	closed     bool
}

func (s *snapshot) Generation() string {
	return s.generation
}

func (s *snapshot) Stat(
	ctx context.Context,
	locator artifactstore.Locator,
) (source.Entry, error) {
	if err := s.ensureOpen(ctx); err != nil {
		return source.Entry{}, err
	}
	name, err := fsName(locator)
	if err != nil {
		return source.Entry{}, err
	}
	info, err := fs.Stat(s.provider, name)
	if errors.Is(err, fs.ErrNotExist) {
		return source.Entry{}, fmt.Errorf(
			"%w: embedded locator %q",
			artifactstore.ErrNotFound,
			locator,
		)
	}
	if err != nil {
		return source.Entry{}, err
	}
	return entryFromInfo(locator, info), nil
}

func (s *snapshot) ReadDir(
	ctx context.Context,
	locator artifactstore.Locator,
) ([]source.Entry, error) {
	if err := s.ensureOpen(ctx); err != nil {
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
			artifactstore.ErrNotFound,
			locator,
		)
	}
	if err != nil {
		return nil, err
	}

	output := make([]source.Entry, 0, len(values))
	for _, value := range values {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		child, err := joinLocator(locator, value.Name())
		if err != nil {
			return nil, err
		}
		info, err := value.Info()
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
	locator artifactstore.Locator,
) (io.ReadCloser, error) {
	if err := s.ensureOpen(ctx); err != nil {
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
			artifactstore.ErrInvalid,
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
	if err := s.ensureOpen(ctx); err != nil {
		return err
	}
	current, err := fingerprint(ctx, s.provider)
	if err != nil {
		return err
	}
	if current != s.generation {
		return fmt.Errorf(
			"%w: embedded source changed during discovery",
			artifactstore.ErrConflict,
		)
	}
	return nil
}

func (s *snapshot) Close() error {
	s.closed = true
	return nil
}

func (s *snapshot) ensureOpen(ctx context.Context) error {
	if s == nil || s.closed {
		return artifactstore.ErrClosed
	}
	return ctx.Err()
}

func fsName(locator artifactstore.Locator) (string, error) {
	if err := artifactstore.ValidateLocator(locator, true); err != nil {
		return "", err
	}
	if locator == "." {
		return ".", nil
	}
	if !fs.ValidPath(string(locator)) {
		return "", fmt.Errorf(
			"%w: invalid embedded locator %q",
			artifactstore.ErrInvalid,
			locator,
		)
	}
	return string(locator), nil
}

func joinLocator(
	parent artifactstore.Locator,
	name string,
) (artifactstore.Locator, error) {
	if name == "" || strings.Contains(name, "/") || !fs.ValidPath(name) {
		return "", fmt.Errorf(
			"%w: invalid embedded entry name %q",
			artifactstore.ErrInvalid,
			name,
		)
	}
	if parent == "." {
		return artifactstore.Locator(name), nil
	}
	return artifactstore.Locator(path.Join(string(parent), name)), nil
}

func entryFromInfo(
	locator artifactstore.Locator,
	info fs.FileInfo,
) source.Entry {
	return source.Entry{
		Locator:     locator,
		Name:        info.Name(),
		SizeBytes:   info.Size(),
		Mode:        uint32(info.Mode()),
		ModifiedAt:  info.ModTime().UTC(),
		IsDirectory: info.IsDir(),
		IsRegular:   info.Mode().IsRegular(),
		IsSymlink:   info.Mode()&fs.ModeSymlink != 0,
	}
}
