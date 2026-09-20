package builtin

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

//go:embed skills
var embeddedSkillsFS embed.FS

//go:embed agents
var embeddedAgentsFS embed.FS

//go:embed mcps
var embeddedMCPFS embed.FS

// EmbeddedSkillPackages exposes the embedded Skill package tree to the Skill
// built-in installer. Artifact Store itself never imports this package.
func EmbeddedSkillPackages() (fs.FS, error) {
	return EmbeddedPackages(
		documentTopology.BuiltinEmbeddedPackageSkills,
	)
}

// EmbeddedAgentPackages exposes the embedded Agent Collection package tree to
// the Agent built-in installer.
func EmbeddedAgentPackages() (fs.FS, error) {
	return EmbeddedPackages(
		documentTopology.BuiltinEmbeddedPackageAgents,
	)
}

func EmbeddedMCPPackages() (fs.FS, error) {
	return EmbeddedPackages(
		documentTopology.BuiltinEmbeddedPackageMCPs,
	)
}

// EmbeddedPackages exposes one configured embedded package set. The switch is
// intentionally limited to compile-time go:embed roots; adding a new embedded
// package family requires an explicit Go embed declaration as well as YAML.
func EmbeddedPackages(packageSet string) (fs.FS, error) {
	var embedded fs.FS
	switch packageSet {
	case documentTopology.BuiltinEmbeddedPackageSkills:
		embedded = embeddedSkillsFS
	case documentTopology.BuiltinEmbeddedPackageAgents:
		embedded = embeddedAgentsFS
	case documentTopology.BuiltinEmbeddedPackageMCPs:
		embedded = embeddedMCPFS
	default:
		return nil, fmt.Errorf(
			"%w: embedded package set %q is not compiled into the application",
			basespec.ErrNotFound,
			packageSet,
		)
	}

	return embeddedSubtree(
		embedded,
		documentTopology.MustBuiltinEmbeddedPackageRoot(packageSet),
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
