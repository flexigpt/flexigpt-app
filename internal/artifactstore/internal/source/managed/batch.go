package managed

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

func (a *Adapter) ApplyPackageBatch(
	ctx context.Context,
	value source.Source,
	publications []source.ManagedPackagePublication,
	removals []source.ManagedPackageAddress,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.validateSource(ctx, value); err != nil {
		return err
	}

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
		address   source.ManagedPackageAddress
		target    string
		temporary string
	}

	staged := make([]stagedPackage, 0, len(publications))
	defer func() {
		for _, value := range staged {
			_ = os.RemoveAll(value.temporary)
		}
	}()

	published := make(map[source.ManagedPackageAddress]struct{})
	for _, publication := range publications {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, duplicate := published[publication.Address]; duplicate {
			return fmt.Errorf(
				"%w: repeated package publication %q/%q/%q",
				basespec.ErrConflict,
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
			basespec.ManagedPackageTemporaryPrefix,
		)
		if err != nil {
			return err
		}
		if err := writeStagedPackageFiles(
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

	removed := make(map[source.ManagedPackageAddress]struct{})
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
				basespec.ErrInvalid,
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

func writeStagedPackageFiles(
	ctx context.Context,
	root string,
	files []source.ManagedPackageFile,
) error {
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := file.Locator.ValidatePortable(false); err != nil {
			return err
		}

		location := filepath.Join(
			root,
			filepath.FromSlash(string(file.Locator)),
		)
		if err := os.MkdirAll(
			filepath.Dir(location),
			basespec.ArtifactStoreDirectoryMode,
		); err != nil {
			return err
		}
		if err := os.WriteFile(location, file.Content, 0o600); err != nil {
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
				basespec.ErrInvalid,
			)
		}
		previous, err = os.MkdirTemp(
			stagingRoot,
			basespec.ManagedPackagePreviousPrefix,
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
