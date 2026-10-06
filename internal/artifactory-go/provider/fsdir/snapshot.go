package fsdir

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type snapshot struct {
	root            string
	generation      string
	traversalPolicy normalizedTraversalPolicy
	closed          atomic.Bool
}

func (s *snapshot) Generation() string {
	return s.generation
}

func (s *snapshot) Stat(
	ctx context.Context,
	locator spec.Locator,
) (sourceModel.Entry, error) {
	if err := s.ensureOpen(); err != nil {
		return sourceModel.Entry{}, err
	}
	if s.traversalPolicy.excludesLocator(string(locator)) {
		return sourceModel.Entry{}, fmt.Errorf(
			"%w: source locator %q is excluded by traversal policy",
			spec.ErrNotFound,
			locator,
		)
	}
	p, err := s.resolve(locator)
	if err != nil {
		return sourceModel.Entry{}, err
	}
	info, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return sourceModel.Entry{}, fmt.Errorf(
			"%w: source locator %q",
			spec.ErrNotFound,
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
	locator spec.Locator,
) ([]sourceModel.Entry, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	if s.traversalPolicy.excludesLocator(string(locator)) {
		return []sourceModel.Entry{}, nil
	}
	p, err := s.resolveDirectory(locator)
	if err != nil {
		return nil, err
	}
	if locator != "." && s.traversalPolicy.isGitSubmoduleDirectory(p) {
		return []sourceModel.Entry{}, nil
	}

	values, err := readDirectoryEntries(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf(
			"%w: source directory %q",
			spec.ErrNotFound,
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
		info, err := os.Stat(filepath.Join(p, value.Name()))
		if err != nil {
			return nil, err
		}
		if info.IsDir() &&
			(s.traversalPolicy.shouldSkipDirectory(info.Name()) ||
				s.traversalPolicy.isGitSubmoduleDirectory(filepath.Join(p, value.Name()))) {
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
		if len(batch) > spec.MaxDiscoveryEntries-len(values) {
			closeErr := directory.Close()
			return nil, errors.Join(
				fmt.Errorf(
					"%w: source directory %q exceeds %d entries",
					spec.ErrInvalid,
					location,
					spec.MaxDiscoveryEntries,
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
	locator spec.Locator,
) (io.ReadCloser, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	if s.traversalPolicy.excludesLocator(string(locator)) {
		return nil, fmt.Errorf(
			"%w: source locator %q is excluded by traversal policy",
			spec.ErrNotFound,
			locator,
		)
	}
	p, err := s.resolve(locator)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf(
			"%w: source file %q",
			spec.ErrNotFound,
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
		return nil, errors.Join(
			fmt.Errorf(
				"%w: source locator %q is not a regular file",
				spec.ErrInvalid,
				locator,
			),
			file.Close(),
		)
	}
	return file, nil
}

func (s *snapshot) Confirm(ctx context.Context) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	current, err := fingerprint(ctx, s.root, s.traversalPolicy)
	if err != nil {
		return err
	}
	if current != s.generation {
		return fmt.Errorf(
			"%w: filesystem source changed during discovery",
			spec.ErrConflict,
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
		return spec.ErrClosed
	}
	return nil
}

func (s *snapshot) resolve(
	locator spec.Locator,
) (string, error) {
	return resolveWithinRoot(s.root, locator)
}

func (s *snapshot) resolveDirectory(
	locator spec.Locator,
) (string, error) {
	return resolveWithinRoot(s.root, locator)
}

// resolveNativePath resolves an existing locator beneath a configured source
// root using normal native filesystem path semantics.
func resolveNativePath(
	root string,
	locator spec.Locator,
) (string, error) {
	rootInfo, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !rootInfo.IsDir() {
		return "", fmt.Errorf(
			"%w: filesystem source root is not a directory",
			spec.ErrInvalid,
		)
	}
	p, err := resolveWithinRoot(root, locator)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf(
			"%w: source locator %q",
			spec.ErrNotFound,
			locator,
		)
	}
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf(
			"%w: source locator %q",
			spec.ErrNotFound,
			locator,
		)
	} else if err != nil {
		return "", err
	}
	return p, nil
}

func resolveWithinRoot(
	root string,
	locator spec.Locator,
) (string, error) {
	if err := locator.Validate(true); err != nil {
		return "", err
	}
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
			spec.ErrInvalid,
			locator,
		)
	}

	return current, nil
}

func entryFromInfo(
	locator spec.Locator,
	info os.FileInfo,
) sourceModel.Entry {
	name := info.Name()
	if locator != "." {
		// Persist source-relative identity rather than host filesystem naming.
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

func joinLocator(
	parent spec.Locator,
	name string,
) (spec.Locator, error) {
	if name == "" || strings.ContainsAny(name, `/\:`) {
		return "", fmt.Errorf(
			"%w: invalid source entry name %q",
			spec.ErrInvalid,
			name,
		)
	}
	if parent == "." {
		return spec.Locator(name), nil
	}
	return spec.Locator(string(parent) + "/" + name), nil
}
