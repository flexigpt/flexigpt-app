package builtin

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

//go:embed artifact-builtin-registry.json
var registryJSON []byte

//go:embed skills/skill-registry.json
var SkillRegistryJSON []byte

//go:embed mcps/mcp_artifact_registry.json
var MCPRegistryJSON []byte

//go:embed skills
var embeddedSkillsPackagesFS embed.FS

//go:embed all:mcps
var embeddedMCPArtifactPackagesFS embed.FS

const embeddedSkillsPackagesRoot = "skills"

const embeddedMCPArtifactPackagesRoot = "mcps"

// EmbeddedSkillsPackages returns the root containing Skill built-in package
// directories. Generic built-in code validates the filesystem boundary but
// does not own or inspect Skill package content.
func EmbeddedSkillsPackages() (fs.FS, error) {
	return openPackageFS(embeddedSkillsPackagesFS, embeddedSkillsPackagesRoot)
}

func EmbeddedMCPArtifactPackages() (fs.FS, error) {
	return openPackageFS(
		embeddedMCPArtifactPackagesFS,
		embeddedMCPArtifactPackagesRoot,
	)
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
