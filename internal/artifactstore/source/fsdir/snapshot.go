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
	info, err := os.Lstat(path)
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
	path, err := s.resolve(locator)
	if err != nil {
		return nil, err
	}
	if locator != "." && s.traversalPolicy.isGitSubmoduleDirectory(path) {
		return []source.Entry{}, nil
	}

	values, err := os.ReadDir(path)
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
		info, err := os.Lstat(filepath.Join(path, value.Name()))
		if err != nil {
			return nil, err
		}
		if info.IsDir() &&
			(s.traversalPolicy.shouldSkipDirectory(info.Name()) ||
				s.traversalPolicy.isGitSubmoduleDirectory(filepath.Join(path, value.Name()))) {
			continue
		}

		// Symlinks remain visible as entries so generic discovery can safely
		// skip them without treating the source as invalid.
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
	info, err := os.Lstat(path)
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
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf(
			"%w: source locator %q is not a regular file",
			artifactstore.ErrInvalid,
			locator,
		)
	}
	return os.Open(path)
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
	if err := artifactstore.ValidateLocator(locator, true); err != nil {
		return "", err
	}
	if locator == "." {
		return s.root, nil
	}
	candidate := filepath.Join(s.root, filepath.FromSlash(string(locator)))
	relative, err := filepath.Rel(s.root, candidate)
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
	return candidate, nil
}

// resolveNativePath resolves an existing locator beneath a configured source
// root while refusing symlinks below that root. The configured root itself may
// be a symlink, but it is canonicalized first so containment is evaluated
// against its actual directory.
func resolveNativePath(
	root string,
	locator artifactstore.Locator,
) (string, error) {
	if err := artifactstore.ValidateLocator(locator, true); err != nil {
		return "", err
	}

	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf(
			"%w: resolve filesystem source root: %w",
			artifactstore.ErrSourceUnavailable,
			err,
		)
	}
	rootInfo, err := os.Stat(resolvedRoot)
	if err != nil {
		return "", err
	}
	if !rootInfo.IsDir() {
		return "", fmt.Errorf(
			"%w: filesystem source root is not a directory",
			artifactstore.ErrInvalid,
		)
	}

	current := resolvedRoot
	if locator == "." {
		return current, nil
	}

	for part := range strings.SplitSeq(string(locator), "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
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
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf(
				"%w: symbolic link %q is not allowed",
				artifactstore.ErrInvalid,
				locator,
			)
		}
	}

	relative, err := filepath.Rel(resolvedRoot, current)
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
		IsSymlink:   info.Mode()&os.ModeSymlink != 0,
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
