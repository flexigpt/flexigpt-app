package managedpackage

import (
	"fmt"
	"io/fs"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// DirectPackageRoots lists a package-set filesystem whose direct children are
// package directories. It assigns no meaning to their contents or names.
func DirectPackageRoots(packages fs.FS) ([]spec.Locator, error) {
	if packages == nil {
		return nil, fmt.Errorf("%w: package filesystem is nil", spec.ErrInvalid)
	}
	entries, err := fs.ReadDir(packages, ".")
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 || len(entries) > spec.MaxDiscoveryEntries {
		return nil, fmt.Errorf("%w: package directory count is invalid", spec.ErrInvalid)
	}

	output := make([]spec.Locator, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			return nil, fmt.Errorf(
				"%w: package-set root contains non-directory %q",
				spec.ErrInvalid,
				entry.Name(),
			)
		}
		locator := spec.Locator(entry.Name())
		if err := locator.ValidatePortable(false); err != nil {
			return nil, err
		}
		output = append(output, locator)
	}
	slices.Sort(output)
	return output, nil
}
