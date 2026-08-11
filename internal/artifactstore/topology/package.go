package topology

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// PackageFile is one owned copy of a regular package file relative to the
// selected package root.
//
// It intentionally contains no host-native path. Built-in registries and
// feature installers may decide package semantics, but Artifact Store owns the
// generic bounded embedded-package read boundary.
type PackageFile struct {
	Locator basespec.Locator
	Content []byte
}

// ReadPackageFiles reads a complete portable package directory from an fs.FS.
//
// The caller owns package identity and feature semantics. This helper owns
// generic package safety: relative-locator validation, regular-file-only
// enforcement, bounded file count, bounded aggregate bytes, deterministic
// ordering, and stable reads.
//
// It deliberately uses slash-native fs paths rather than filepath helpers.
// "fs.FS" paths are always slash-separated, including on Windows.
func ReadPackageFiles(
	ctx context.Context,
	packages fs.FS,
	packageRoot basespec.Locator,
) ([]PackageFile, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: embedded package context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded package filesystem is nil",
			basespec.ErrInvalid,
		)
	}
	if packageRoot == "" {
		return nil, fmt.Errorf(
			"%w: embedded package root is required",
			basespec.ErrInvalid,
		)
	}
	if packageRoot != "." {
		if err := basespec.ValidatePortableLocator(packageRoot, false); err != nil {
			return nil, err
		}
	}

	info, err := fs.Stat(packages, string(packageRoot))
	if err != nil {
		return nil, fmt.Errorf(
			"stat embedded package %q: %w",
			packageRoot,
			err,
		)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf(
			"%w: embedded package %q is not a directory",
			basespec.ErrInvalid,
			packageRoot,
		)
	}

	files := make([]PackageFile, 0)
	seen := make(map[basespec.Locator]struct{})
	var totalBytes int64

	err = fs.WalkDir(
		packages,
		string(packageRoot),
		func(location string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry == nil {
				return fmt.Errorf(
					"%w: embedded package walk returned no entry for %q",
					basespec.ErrInvalid,
					location,
				)
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf(
					"%w: embedded package file %q is not regular",
					basespec.ErrInvalid,
					location,
				)
			}
			if len(files) >= basespec.MaxDiscoveryEntries {
				return fmt.Errorf(
					"%w: embedded package exceeds the file count limit",
					basespec.ErrInvalid,
				)
			}

			relative, err := packageRelativeLocator(packageRoot, location)
			if err != nil {
				return err
			}
			if _, duplicate := seen[relative]; duplicate {
				return fmt.Errorf(
					"%w: embedded package contains duplicate file %q",
					basespec.ErrInvalid,
					relative,
				)
			}
			seen[relative] = struct{}{}

			fileInfo, err := entry.Info()
			if err != nil {
				return err
			}
			if !fileInfo.Mode().IsRegular() ||
				fileInfo.Size() < 0 ||
				fileInfo.Size() > basespec.MaxScanBytes-totalBytes {
				return fmt.Errorf(
					"%w: embedded package exceeds the byte limit",
					basespec.ErrInvalid,
				)
			}

			content, err := readPackageFile(
				ctx,
				packages,
				location,
				fileInfo.Size(),
			)
			if err != nil {
				return err
			}
			totalBytes += int64(len(content))
			files = append(files, PackageFile{
				Locator: relative,
				Content: append([]byte(nil), content...),
			})
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(left, right int) bool {
		return files[left].Locator < files[right].Locator
	})
	return files, nil
}

func packageRelativeLocator(
	packageRoot basespec.Locator,
	location string,
) (basespec.Locator, error) {
	relative := location
	if packageRoot != "." {
		prefix := string(packageRoot) + "/"
		var found bool
		relative, found = strings.CutPrefix(location, prefix)
		if !found || relative == "" {
			return "", fmt.Errorf(
				"%w: embedded package file %q is outside package root %q",
				basespec.ErrInvalid,
				location,
				packageRoot,
			)
		}
	}
	if relative == "." ||
		relative == ".." ||
		strings.HasPrefix(relative, "../") ||
		!fs.ValidPath(relative) {
		return "", fmt.Errorf(
			"%w: invalid embedded package file %q",
			basespec.ErrInvalid,
			location,
		)
	}

	value := basespec.Locator(relative)
	if err := basespec.ValidatePortableLocator(value, false); err != nil {
		return "", err
	}
	return value, nil
}

func readPackageFile(
	ctx context.Context,
	packages fs.FS,
	location string,
	expectedSize int64,
) ([]byte, error) {
	if expectedSize < 0 || expectedSize > basespec.MaxScanBytes {
		return nil, fmt.Errorf(
			"%w: embedded package file %q has invalid size",
			basespec.ErrInvalid,
			location,
		)
	}

	file, err := packages.Open(location)
	if err != nil {
		return nil, err
	}
	content, readErr := io.ReadAll(io.LimitReader(file, expectedSize+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if int64(len(content)) != expectedSize {
		return nil, fmt.Errorf(
			"%w: embedded package file %q changed while being read",
			basespec.ErrConflict,
			location,
		)
	}
	return content, nil
}
