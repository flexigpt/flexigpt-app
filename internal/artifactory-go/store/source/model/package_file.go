package source

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// ManagedPackageFile is one regular file relative to a managed package root.
// Empty directories are deliberately not represented.
type ManagedPackageFile struct {
	Locator spec.Locator `json:"locator"`
	Content []byte       `json:"content"`
}

// NormalizeManagedPackageFiles validates and returns an independently owned,
// deterministic package file set. It is separate from package addressing so a
// caller can validate package contents before it has derived the final semantic
// package address.
func NormalizeManagedPackageFiles(
	input []ManagedPackageFile,
) ([]ManagedPackageFile, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf(
			"%w: managed package must contain at least one file",
			spec.ErrInvalid,
		)
	}
	if len(input) > spec.MaxDiscoveryEntries {
		return nil, fmt.Errorf(
			"%w: managed package exceeds the file count limit",
			spec.ErrInvalid,
		)
	}

	output := make([]ManagedPackageFile, len(input))
	seen := make(map[spec.Locator]struct{}, len(input))
	filesByIdentity := make(map[string]spec.Locator, len(input))
	directoriesByIdentity := make(map[string]spec.Locator)

	var total int64
	for index, file := range input {
		if err := file.Locator.ValidatePortable(false); err != nil {
			return nil, fmt.Errorf(
				"managed package files[%d]: %w",
				index,
				err,
			)
		}
		if _, duplicate := seen[file.Locator]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate managed package file %q",
				spec.ErrInvalid,
				file.Locator,
			)
		}
		seen[file.Locator] = struct{}{}

		identity := portableLocatorIdentity(file.Locator)
		if previous, collision := filesByIdentity[identity]; collision {
			return nil, fmt.Errorf(
				"%w: managed package file %q collides with %q",
				spec.ErrInvalid,
				file.Locator,
				previous,
			)
		}
		if child, conflict := directoriesByIdentity[identity]; conflict {
			return nil, fmt.Errorf(
				"%w: managed package file %q conflicts with child path %q",
				spec.ErrInvalid,
				file.Locator,
				child,
			)
		}

		for parent := path.Dir(string(file.Locator)); parent != "."; parent = path.Dir(parent) {
			parentIdentity := portableLocatorIdentity(
				spec.Locator(parent),
			)
			if parentFile, conflict := filesByIdentity[parentIdentity]; conflict {
				return nil, fmt.Errorf(
					"%w: managed package path %q is below file %q",
					spec.ErrInvalid,
					file.Locator,
					parentFile,
				)
			}
			directoriesByIdentity[parentIdentity] = file.Locator
		}
		filesByIdentity[identity] = file.Locator

		if int64(len(file.Content)) > spec.MaxScanBytes-total {
			return nil, fmt.Errorf(
				"%w: managed package exceeds the byte limit",
				spec.ErrInvalid,
			)
		}
		total += int64(len(file.Content))

		output[index] = ManagedPackageFile{
			Locator: file.Locator,
			Content: append([]byte(nil), file.Content...),
		}
	}

	sort.Slice(output, func(left, right int) bool {
		return output[left].Locator < output[right].Locator
	})
	return output, nil
}

// PackageFileContentOneOf returns exactly one package file whose locator is
// listed in candidates. A package cannot contain two alternative root
// documents because that would make its declaration root ambiguous.
func PackageFileContentOneOf(
	files []ManagedPackageFile,
	candidates []spec.Locator,
) (documentFile spec.Locator, document []byte, found bool, err error) {
	if len(candidates) == 0 {
		return "", nil, false, fmt.Errorf(
			"%w: package document candidates are required",
			spec.ErrInvalid,
		)
	}

	seenCandidates := make(map[spec.Locator]struct{}, len(candidates))
	for _, candidate := range candidates {
		if err := candidate.ValidatePortable(false); err != nil {
			return "", nil, false, err
		}
		if _, duplicate := seenCandidates[candidate]; duplicate {
			return "", nil, false, fmt.Errorf(
				"%w: duplicate package document candidate %q",
				spec.ErrInvalid,
				candidate,
			)
		}
		seenCandidates[candidate] = struct{}{}
	}

	var (
		selected        spec.Locator
		selectedContent []byte
	)
	for _, candidate := range candidates {
		content, found := packageFileContent(files, candidate)
		if !found {
			continue
		}
		if selected != "" {
			return "", nil, false, fmt.Errorf(
				"%w: package contains both %q and %q",
				spec.ErrIdentityConflict,
				selected,
				candidate,
			)
		}
		selected = candidate
		selectedContent = content
	}
	if selected == "" {
		return "", nil, false, nil
	}
	return selected, selectedContent, true, nil
}

// packageFileContent returns an owned copy of one package-relative file.
func packageFileContent(
	files []ManagedPackageFile,
	locator spec.Locator,
) ([]byte, bool) {
	for _, file := range files {
		if file.Locator != locator {
			continue
		}
		return append([]byte(nil), file.Content...), true
	}
	return nil, false
}

func portableLocatorIdentity(
	value spec.Locator,
) string {
	if value == "." {
		return "."
	}
	return strings.ToLower(string(value))
}
