package managedpackage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"

	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

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
	packageRoot spec.Locator,
) ([]managedpackageModel.ManagedPackageFile, error) {
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded package filesystem is nil",
			spec.ErrInvalid,
		)
	}
	if packageRoot == "" {
		return nil, fmt.Errorf(
			"%w: embedded package root is required",
			spec.ErrInvalid,
		)
	}
	if packageRoot != "." {
		if err := packageRoot.ValidatePortable(false); err != nil {
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
			spec.ErrInvalid,
			packageRoot,
		)
	}

	files := make([]managedpackageModel.ManagedPackageFile, 0)
	seen := make(map[spec.Locator]struct{})
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
					spec.ErrInvalid,
					location,
				)
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf(
					"%w: embedded package file %q is not regular",
					spec.ErrInvalid,
					location,
				)
			}
			if len(files) >= spec.MaxDiscoveryEntries {
				return fmt.Errorf(
					"%w: embedded package exceeds the file count limit",
					spec.ErrInvalid,
				)
			}

			relative, err := packageRelativeLocator(packageRoot, location)
			if err != nil {
				return err
			}
			if _, duplicate := seen[relative]; duplicate {
				return fmt.Errorf(
					"%w: embedded package contains duplicate file %q",
					spec.ErrInvalid,
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
				fileInfo.Size() > spec.MaxScanBytes-totalBytes {
				return fmt.Errorf(
					"%w: embedded package exceeds the byte limit",
					spec.ErrInvalid,
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
			files = append(files, managedpackageModel.ManagedPackageFile{
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
	packageRoot spec.Locator,
	location string,
) (spec.Locator, error) {
	relative := location
	if packageRoot != "." {
		prefix := string(packageRoot) + "/"
		var found bool
		relative, found = strings.CutPrefix(location, prefix)
		if !found || relative == "" {
			return "", fmt.Errorf(
				"%w: embedded package file %q is outside package root %q",
				spec.ErrInvalid,
				location,
				packageRoot,
			)
		}
	}

	value := spec.Locator(relative)
	if err := value.ValidatePortable(false); err != nil {
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
	if expectedSize < 0 || expectedSize > spec.MaxScanBytes {
		return nil, fmt.Errorf(
			"%w: embedded package file %q has invalid size",
			spec.ErrInvalid,
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

	if int64(len(content)) != expectedSize {
		return nil, fmt.Errorf(
			"%w: embedded package file %q changed while being read",
			spec.ErrConflict,
			location,
		)
	}
	return content, nil
}
