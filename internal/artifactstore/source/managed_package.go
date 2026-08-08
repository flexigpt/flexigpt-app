package source

import (
	"fmt"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// ValidateManagedPackageDirectory validates one managed package directory
// relative to its managed Source root.
func ValidateManagedPackageDirectory(
	directory basespec.Locator,
) error {
	return basespec.ValidatePortableLocator(directory, false)
}

// NormalizeManagedPackagePublication validates and returns an independently
// owned deterministic managed package publication.
//
// Files are relative to Directory. Empty directories are deliberately not
// represented because package publication is defined by its regular files.
func NormalizeManagedPackagePublication(
	input ManagedPackagePublication,
) (ManagedPackagePublication, error) {
	if err := ValidateManagedPackageDirectory(input.Directory); err != nil {
		return ManagedPackagePublication{}, err
	}
	if input.ExpectedGeneration != "" {
		if err := basespec.ValidateSourceGeneration(
			input.ExpectedGeneration,
		); err != nil {
			return ManagedPackagePublication{}, err
		}
	}
	if len(input.Files) == 0 {
		return ManagedPackagePublication{}, fmt.Errorf(
			"%w: managed package must contain at least one file",
			basespec.ErrInvalid,
		)
	}
	if len(input.Files) > basespec.MaxDiscoveryEntries {
		return ManagedPackagePublication{}, fmt.Errorf(
			"%w: managed package exceeds the file count limit",
			basespec.ErrInvalid,
		)
	}

	output := input
	output.Files = make([]ManagedPackageFile, len(input.Files))
	seen := make(map[basespec.Locator]struct{}, len(input.Files))
	filesByIdentity := make(map[string]basespec.Locator, len(input.Files))
	directoriesByIdentity := make(map[string]basespec.Locator)

	var total int64
	for index, file := range input.Files {
		if err := basespec.ValidatePortableLocator(
			file.Locator,
			false,
		); err != nil {
			return ManagedPackagePublication{}, fmt.Errorf(
				"managed package files[%d]: %w",
				index,
				err,
			)
		}
		if _, duplicate := seen[file.Locator]; duplicate {
			return ManagedPackagePublication{}, fmt.Errorf(
				"%w: duplicate managed package file %q",
				basespec.ErrInvalid,
				file.Locator,
			)
		}
		seen[file.Locator] = struct{}{}

		identity, err := basespec.PortableLocatorIdentity(
			file.Locator,
			false,
		)
		if err != nil {
			return ManagedPackagePublication{}, err
		}
		if previous, collision := filesByIdentity[identity]; collision {
			return ManagedPackagePublication{}, fmt.Errorf(
				"%w: managed package file %q collides with %q",
				basespec.ErrInvalid,
				file.Locator,
				previous,
			)
		}
		if child, conflict := directoriesByIdentity[identity]; conflict {
			return ManagedPackagePublication{}, fmt.Errorf(
				"%w: managed package file %q conflicts with child path %q",
				basespec.ErrInvalid,
				file.Locator,
				child,
			)
		}

		for parent := path.Dir(string(file.Locator)); parent != "."; parent = path.Dir(parent) {
			parentIdentity, err := basespec.PortableLocatorIdentity(
				basespec.Locator(parent),
				false,
			)
			if err != nil {
				return ManagedPackagePublication{}, err
			}
			if parentFile, conflict := filesByIdentity[parentIdentity]; conflict {
				return ManagedPackagePublication{}, fmt.Errorf(
					"%w: managed package path %q is below file %q",
					basespec.ErrInvalid,
					file.Locator,
					parentFile,
				)
			}
			directoriesByIdentity[parentIdentity] = file.Locator
		}
		filesByIdentity[identity] = file.Locator

		if int64(len(file.Content)) > basespec.MaxScanBytes-total {
			return ManagedPackagePublication{}, fmt.Errorf(
				"%w: managed package exceeds the byte limit",
				basespec.ErrInvalid,
			)
		}
		total += int64(len(file.Content))
		output.Files[index] = ManagedPackageFile{
			Locator: file.Locator,
			Content: append([]byte(nil), file.Content...),
		}
	}

	sort.Slice(output.Files, func(left, right int) bool {
		return output.Files[left].Locator < output.Files[right].Locator
	})
	return output, nil
}
