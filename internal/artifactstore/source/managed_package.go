package source

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

// ValidateManagedPackageDirectory validates one managed package directory
// relative to its managed Source root.
func ValidateManagedPackageDirectory(
	directory artifactstore.Locator,
) error {
	return artifactstore.ValidatePortableLocator(directory, false)
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
		if err := artifactstore.ValidateSourceGeneration(
			input.ExpectedGeneration,
		); err != nil {
			return ManagedPackagePublication{}, err
		}
	}
	if len(input.Files) == 0 {
		return ManagedPackagePublication{}, fmt.Errorf(
			"%w: managed package must contain at least one file",
			artifactstore.ErrInvalid,
		)
	}
	if len(input.Files) > artifactstore.MaxDiscoveryEntries {
		return ManagedPackagePublication{}, fmt.Errorf(
			"%w: managed package exceeds the file count limit",
			artifactstore.ErrInvalid,
		)
	}

	output := input
	output.Files = make([]ManagedPackageFile, len(input.Files))
	seen := make(map[artifactstore.Locator]struct{}, len(input.Files))
	seenCaseFolded := make(
		map[string]artifactstore.Locator,
		len(input.Files),
	)
	var total int64

	for index, file := range input.Files {
		if err := artifactstore.ValidatePortableLocator(
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
				artifactstore.ErrInvalid,
				file.Locator,
			)
		}
		seen[file.Locator] = struct{}{}

		caseFolded := strings.ToLower(string(file.Locator))
		if previous, duplicate := seenCaseFolded[caseFolded]; duplicate {
			return ManagedPackagePublication{}, fmt.Errorf(
				"%w: managed package files %q and %q collide on case-insensitive filesystems",
				artifactstore.ErrInvalid,
				previous,
				file.Locator,
			)
		}
		seenCaseFolded[caseFolded] = file.Locator

		if int64(len(file.Content)) > artifactstore.MaxScanBytes-total {
			return ManagedPackagePublication{}, fmt.Errorf(
				"%w: managed package exceeds the byte limit",
				artifactstore.ErrInvalid,
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
