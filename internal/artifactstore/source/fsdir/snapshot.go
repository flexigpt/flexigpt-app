package fsdir

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type snapshot struct {
	root            string
	generation      string
	traversalPolicy normalizedTraversalPolicy
	closed          bool
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
	if s.traversalPolicy.excludesLocator(string(locator)) {
		return source.Entry{}, fmt.Errorf(
			"%w: source locator %q is excluded by traversal policy",
			artifactstore.ErrNotFound,
			locator,
		)
	}
	path, err := s.resolve(locator)
	if err != nil {
		return source.Entry{}, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return source.Entry{}, fmt.Errorf(
			"%w: source locator %q",
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
	if s.traversalPolicy.excludesLocator(string(locator)) {
		return []source.Entry{}, nil
	}
	path, err := s.resolveDirectory(locator)
	if err != nil {
		return nil, err
	}
	if locator != "." && s.traversalPolicy.isGitSubmoduleDirectory(path) {
		return []source.Entry{}, nil
	}

	values, err := readDirectoryEntries(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf(
			"%w: source directory %q",
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
		info, err := os.Stat(filepath.Join(path, value.Name()))
		if err != nil {
			return nil, err
		}
		if info.IsDir() &&
			(s.traversalPolicy.shouldSkipDirectory(info.Name()) ||
				s.traversalPolicy.isGitSubmoduleDirectory(filepath.Join(path, value.Name()))) {
			continue
		}
		output = append(output, entryFromInfo(child, info))
	}
	sort.Slice(output, func(left, right int) bool {
		return output[left].Locator < output[right].Locator
	})
	return output, nil
}

func readDirectoryEntries(location string) ([]os.DirEntry, error) {
	directory, err := os.Open(location)
	if err != nil {
		return nil, err
	}

	values := make([]os.DirEntry, 0)
	for {
		batch, readErr := directory.ReadDir(directoryReadBatchSize)
		if len(batch) > artifactstore.MaxDiscoveryEntries-len(values) {
			closeErr := directory.Close()
			return nil, errors.Join(
				fmt.Errorf(
					"%w: source directory %q exceeds %d entries",
					artifactstore.ErrInvalid,
					location,
					artifactstore.MaxDiscoveryEntries,
				),
				closeErr,
			)
		}
		values = append(values, batch...)
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			closeErr := directory.Close()
			return nil, errors.Join(readErr, closeErr)
		}
	}
	if err := directory.Close(); err != nil {
		return nil, err
	}
	return values, nil
}

func (s *snapshot) Open(
	ctx context.Context,
	locator artifactstore.Locator,
) (io.ReadCloser, error) {
	if err := s.ensureOpen(ctx); err != nil {
		return nil, err
	}
	if s.traversalPolicy.excludesLocator(string(locator)) {
		return nil, fmt.Errorf(
			"%w: source locator %q is excluded by traversal policy",
			artifactstore.ErrNotFound,
			locator,
		)
	}
	path, err := s.resolve(locator)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf(
			"%w: source file %q",
			artifactstore.ErrNotFound,
			locator,
		)
	}
	if err != nil {
		return nil, err
	}
	info, statErr := file.Stat()
	if statErr != nil {
		return nil, errors.Join(statErr, file.Close())
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf(
			"%w: source locator %q is not a regular file",
			artifactstore.ErrInvalid,
			locator,
		)
	}
	return file, nil
}

func (s *snapshot) Confirm(ctx context.Context) error {
	if err := s.ensureOpen(ctx); err != nil {
		return err
	}
	current, err := fingerprint(ctx, s.root, s.traversalPolicy)
	if err != nil {
		return err
	}
	if current != s.generation {
		return fmt.Errorf(
			"%w: filesystem source changed during discovery",
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

func (s *snapshot) resolve(
	locator artifactstore.Locator,
) (string, error) {
	return resolveWithinRoot(s.root, locator)
}

func (s *snapshot) resolveDirectory(
	locator artifactstore.Locator,
) (string, error) {
	return resolveWithinRoot(s.root, locator)
}

// resolveNativePath resolves an existing locator beneath a configured source
// root using normal native filesystem path semantics.
func resolveNativePath(
	root string,
	locator artifactstore.Locator,
) (string, error) {
	if err := artifactstore.ValidateLocator(locator, true); err != nil {
		return "", err
	}
	root = filepath.Clean(root)
	rootInfo, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !rootInfo.IsDir() {
		return "", fmt.Errorf(
			"%w: filesystem source root is not a directory",
			artifactstore.ErrInvalid,
		)
	}
	path, err := resolveWithinRoot(root, locator)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf(
			"%w: source locator %q",
			artifactstore.ErrNotFound,
			locator,
		)
	}
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf(
			"%w: source locator %q",
			artifactstore.ErrNotFound,
			locator,
		)
	} else if err != nil {
		return "", err
	}
	return path, nil
}

func resolveWithinRoot(
	root string,
	locator artifactstore.Locator,
) (string, error) {
	if err := artifactstore.ValidateLocator(locator, true); err != nil {
		return "", err
	}
	root = filepath.Clean(root)
	current := root
	if locator != "." {
		current = filepath.Join(root, filepath.FromSlash(string(locator)))
	}

	relative, err := filepath.Rel(root, current)
	if err != nil {
		return "", err
	}
	if relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(relative) {
		return "", fmt.Errorf(
			"%w: locator %q escapes source root",
			artifactstore.ErrInvalid,
			locator,
		)
	}

	return current, nil
}

func entryFromInfo(
	locator artifactstore.Locator,
	info os.FileInfo,
) source.Entry {
	return source.Entry{
		Locator:     locator,
		Name:        info.Name(),
		SizeBytes:   info.Size(),
		Mode:        uint32(info.Mode()),
		ModifiedAt:  info.ModTime().UTC(),
		IsDirectory: info.IsDir(),
		IsRegular:   info.Mode().IsRegular(),
	}
}

func joinLocator(
	parent artifactstore.Locator,
	name string,
) (artifactstore.Locator, error) {
	if name == "" || strings.ContainsAny(name, `/\:`) {
		return "", fmt.Errorf(
			"%w: invalid source entry name %q",
			artifactstore.ErrInvalid,
			name,
		)
	}
	if parent == "." {
		return artifactstore.Locator(name), nil
	}
	return artifactstore.Locator(string(parent) + "/" + name), nil
}
