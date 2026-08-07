package builtin

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

//go:embed artifact-builtin-registry.json
var registryJSON []byte

//go:embed skills/skill-registry.json
var SkillRegistryJSON []byte

//go:embed skills
var embeddedSkillsPackagesFS embed.FS

const embeddedSkillsPackagesRoot = "skills"

// EmbeddedSkillsPackages returns the root containing Skill built-in package
// directories. Generic built-in code validates the filesystem boundary but
// does not own or inspect Skill package content.
func EmbeddedSkillsPackages() (fs.FS, error) {
	return openPackageFS(embeddedSkillsPackagesFS, embeddedSkillsPackagesRoot)
}

// PackageFile is an immutable-by-convention copy of one regular embedded file
// relative to a selected embedded package directory.
type PackageFile struct {
	Locator basespec.Locator
	Content []byte
}

// ReadPackageFiles reads a complete embedded package directory into a bounded,
// deterministic, relative file list. Artifact-specific code can convert the
// result into its own publication or installation representation.
func ReadPackageFiles(
	ctx context.Context,
	packages fs.FS,
	packageRoot basespec.Locator,
) ([]PackageFile, error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: embedded package context is nil", basespec.ErrInvalid)
	}
	if packages == nil {
		return nil, fmt.Errorf("%w: embedded package filesystem is nil", basespec.ErrInvalid)
	}
	if packageRoot == "" {
		return nil, fmt.Errorf("%w: embedded package root is required", basespec.ErrInvalid)
	}
	if packageRoot != "." {
		if err := basespec.ValidatePortableLocator(packageRoot, false); err != nil {
			return nil, err
		}
	}
	info, err := fs.Stat(packages, string(packageRoot))
	if err != nil {
		return nil, fmt.Errorf("stat embedded package %q: %w", packageRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf(
			"%w: embedded package %q is not a directory",
			basespec.ErrInvalid,
			packageRoot,
		)
	}

	files := make([]PackageFile, 0)
	var totalBytes int64
	err = fs.WalkDir(packages, string(packageRoot), func(
		location string,
		entry fs.DirEntry,
		walkErr error,
	) error {
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
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if fileInfo.Size() < 0 ||
			fileInfo.Size() > basespec.MaxScanBytes-totalBytes {
			return fmt.Errorf(
				"%w: embedded package exceeds the byte limit",
				basespec.ErrInvalid,
			)
		}
		relative, err := filepath.Rel(string(packageRoot), location)
		if err != nil ||
			relative == "." ||
			relative == ".." ||
			strings.HasPrefix(relative, "../") {
			return fmt.Errorf(
				"%w: invalid embedded package file %q",
				basespec.ErrInvalid,
				location,
			)
		}
		locator := basespec.Locator(relative)
		if err := basespec.ValidatePortableLocator(locator, false); err != nil {
			return err
		}
		content, err := fs.ReadFile(packages, location)
		if err != nil {
			return err
		}
		if int64(len(content)) != fileInfo.Size() {
			return fmt.Errorf(
				"%w: embedded package file %q changed while being read",
				basespec.ErrConflict,
				location,
			)
		}
		totalBytes += int64(len(content))
		files = append(files, PackageFile{
			Locator: locator,
			Content: append([]byte(nil), content...),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(left, right int) bool {
		return files[left].Locator < files[right].Locator
	})
	return files, nil
}

// openPackageFS validates and opens an artifact-family-owned embedded package
// subtree. The caller owns the embed.FS and the subtree name.
func openPackageFS(
	embedded fs.FS,
	root string,
) (fs.FS, error) {
	if embedded == nil {
		return nil, fmt.Errorf("%w: embedded filesystem is nil", basespec.ErrInvalid)
	}
	if root == "" || !fs.ValidPath(root) {
		return nil, fmt.Errorf(
			"%w: invalid embedded package root %q",
			basespec.ErrInvalid,
			root,
		)
	}
	info, err := fs.Stat(embedded, root)
	if err != nil {
		return nil, fmt.Errorf("stat embedded package root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf(
			"%w: embedded package root %q is not a directory",
			basespec.ErrInvalid,
			root,
		)
	}
	return fs.Sub(embedded, root)
}
