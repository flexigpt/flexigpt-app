package managedpackage

import (
	"fmt"
	"io/fs"
	"slices"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// DirectPackageRoots lists a package-set filesystem whose direct children are
// package directories. It assigns no meaning to their contents or names.
func DirectPackageRoots(packages fs.FS) ([]storeSpec.Locator, error) {
	if packages == nil {
		return nil, fmt.Errorf("%w: package filesystem is nil", storeSpec.ErrInvalid)
	}
	entries, err := fs.ReadDir(packages, ".")
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 || len(entries) > storeSpec.MaxDiscoveryEntries {
		return nil, fmt.Errorf("%w: package directory count is invalid", storeSpec.ErrInvalid)
	}

	output := make([]storeSpec.Locator, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			return nil, fmt.Errorf(
				"%w: package-set root contains non-directory %q",
				storeSpec.ErrInvalid,
				entry.Name(),
			)
		}
		locator := storeSpec.Locator(entry.Name())
		if err := locator.ValidatePortable(false); err != nil {
			return nil, err
		}
		output = append(output, locator)
	}
	slices.Sort(output)
	return output, nil
}
