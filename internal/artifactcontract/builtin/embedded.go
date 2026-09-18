package builtin

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

//go:embed skills
var embeddedSkillsFS embed.FS

//go:embed mcps
var embeddedMCPFS embed.FS

// EmbeddedSkillPackages exposes the embedded Skill package tree to the Skill
// built-in installer. Artifact Store itself never imports this package.
func EmbeddedSkillPackages() (fs.FS, error) {
	return embeddedSubtree(
		embeddedSkillsFS,
		EmbeddedSkillDataRoot,
	)
}

func EmbeddedMCPPackages() (fs.FS, error) {
	return embeddedSubtree(
		embeddedMCPFS,
		EmbeddedMCPDataRoot,
	)
}

// DirectPackageRoots returns validated direct package directories from an
// embedded package filesystem.
func DirectPackageRoots(
	packages fs.FS,
) ([]basespec.Locator, error) {
	if packages == nil {
		return nil, fmt.Errorf(
			"%w: embedded package filesystem is nil",
			basespec.ErrInvalid,
		)
	}

	entries, err := fs.ReadDir(packages, ".")
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf(
			"%w: embedded package filesystem has no packages",
			basespec.ErrInvalid,
		)
	}

	output := make([]basespec.Locator, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			return nil, fmt.Errorf(
				"%w: embedded package root contains non-directory %q",
				basespec.ErrInvalid,
				entry.Name(),
			)
		}
		root := basespec.Locator(entry.Name())
		if err := root.ValidatePortable(false); err != nil {
			return nil, err
		}
		output = append(output, root)
	}
	slices.Sort(output)
	return output, nil
}

// PackageFileContent returns an owned copy of one package-relative file.
func PackageFileContent(
	files []source.ManagedPackageFile,
	locator basespec.Locator,
) ([]byte, bool) {
	for _, file := range files {
		if file.Locator != locator {
			continue
		}
		return append([]byte(nil), file.Content...), true
	}
	return nil, false
}

func embeddedSubtree(
	embedded fs.FS,
	root basespec.Locator,
) (fs.FS, error) {
	if embedded == nil || !fs.ValidPath(string(root)) {
		return nil, fmt.Errorf(
			"invalid embedded built-in root %q",
			root,
		)
	}
	return fs.Sub(embedded, string(root))
}
