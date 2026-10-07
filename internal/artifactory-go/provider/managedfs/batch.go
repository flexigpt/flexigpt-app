package managedfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (a *Adapter) ApplyPackageBatch(
	ctx context.Context,
	value sourceModel.Source,
	publications []managedpackageModel.ManagedPackagePublication,
	removals []managedpackageModel.ManagedPackageAddress,
) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	root, err := a.sourceRootPath(value, true)
	if err != nil {
		return err
	}
	stagingRoot, err := a.sourceStagingPath(value, true)
	if err != nil {
		return err
	}

	type stagedPackage struct {
		address   managedpackageModel.ManagedPackageAddress
		target    string
		temporary string
	}

	staged := make([]stagedPackage, 0, len(publications))
	defer func() {
		for _, value := range staged {
			_ = os.RemoveAll(value.temporary)
		}
	}()

	published := make(map[managedpackageModel.ManagedPackageAddress]struct{})
	for _, publication := range publications {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, duplicate := published[publication.Address]; duplicate {
			return fmt.Errorf(
				"%w: repeated package publication %q/%q/%q",
				spec.ErrConflict,
				publication.Address.Kind,
				publication.Address.Name,
				publication.Address.Version,
			)
		}
		published[publication.Address] = struct{}{}

		directory, err := publication.Address.Directory()
		if err != nil {
			return err
		}
		target, err := managedPackagePath(root, directory, true)
		if err != nil {
			return err
		}

		exists, equivalent, err := equivalentPackage(
			target,
			publication.Files,
		)
		if err != nil {
			return err
		}
		if exists && equivalent {
			continue
		}

		temporary, err := os.MkdirTemp(
			stagingRoot,
			managedPackageTemporaryPrefix,
		)
		if err != nil {
			return err
		}
		if err := writeManagedPackageFiles(
			ctx,
			temporary,
			publication.Files,
		); err != nil {
			_ = os.RemoveAll(temporary)
			return err
		}

		staged = append(staged, stagedPackage{
			address:   publication.Address,
			target:    target,
			temporary: temporary,
		})
	}

	for _, value := range staged {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := replaceStagedPackage(
			stagingRoot,
			value.temporary,
			value.target,
		); err != nil {
			return err
		}
	}

	removed := make(map[managedpackageModel.ManagedPackageAddress]struct{})
	for _, address := range removals {
		if _, replaced := published[address]; replaced {
			continue
		}
		if _, duplicate := removed[address]; duplicate {
			continue
		}
		removed[address] = struct{}{}

		if err := ctx.Err(); err != nil {
			return err
		}
		directory, err := address.Directory()
		if err != nil {
			return err
		}
		target, err := managedPackagePath(root, directory, false)
		if err != nil {
			return err
		}

		info, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf(
				"%w: managed package target is not a directory",
				spec.ErrInvalid,
			)
		}
		if err := os.RemoveAll(target); err != nil {
			return err
		}
		if err := pruneEmptyManagedParents(
			root,
			filepath.Dir(target),
		); err != nil {
			return err
		}
	}
	return nil
}

func replaceStagedPackage(
	stagingRoot string,
	temporary string,
	target string,
) error {
	previous := ""

	info, err := os.Lstat(target)
	switch {
	case err == nil:
		if !info.IsDir() {
			return fmt.Errorf(
				"%w: managed package target is not a directory",
				spec.ErrInvalid,
			)
		}
		previous, err = os.MkdirTemp(
			stagingRoot,
			managedPackagePreviousPrefix,
		)
		if err != nil {
			return err
		}
		if err := os.Remove(previous); err != nil {
			return err
		}
		if err := os.Rename(target, previous); err != nil {
			return err
		}

	case errors.Is(err, os.ErrNotExist):
	default:
		return err
	}

	if err := os.Rename(temporary, target); err != nil {
		if previous == "" {
			return err
		}
		return errors.Join(err, os.Rename(previous, target))
	}
	if previous == "" {
		return nil
	}
	return os.RemoveAll(previous)
}
