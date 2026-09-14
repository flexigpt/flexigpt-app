package artifactbuiltin

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

//go:embed skills
var embeddedSkillsFS embed.FS

//go:embed mcps
var embeddedMCPFS embed.FS

// ReadEmbeddedSkillRegistry reads the application-owned non-portable built-in
// Skill registration manifest. It is not a portable Artifact declaration.
func ReadEmbeddedSkillRegistry() ([]byte, error) {
	return readEmbeddedFile(
		embeddedSkillsFS,
		EmbeddedSkillRegistryLocator,
	)
}

// EmbeddedSkillPackages exposes the embedded Skill package tree to the Skill
// built-in installer. Artifact Store itself never imports this package.
func EmbeddedSkillPackages() (fs.FS, error) {
	return embeddedSubtree(
		embeddedSkillsFS,
		EmbeddedSkillDataRoot,
	)
}

// ReadEmbeddedMCPRegistry reads application-owned MCP package registration
// metadata. It is a physical package index, not a Store Collection.
func ReadEmbeddedMCPRegistry() ([]byte, error) {
	return readEmbeddedFile(
		embeddedMCPFS,
		EmbeddedMCPRegistryLocator,
	)
}

// EmbeddedMCPPackages exposes the embedded MCP package tree to the MCP
// built-in installer. Artifact Store never imports this package.
func EmbeddedMCPPackages() (fs.FS, error) {
	return embeddedSubtree(
		embeddedMCPFS,
		EmbeddedMCPDataRoot,
	)
}

func readEmbeddedFile(
	embedded fs.FS,
	location basespec.Locator,
) ([]byte, error) {
	if embedded == nil || !fs.ValidPath(string(location)) {
		return nil, fmt.Errorf(
			"invalid embedded built-in file %q",
			location,
		)
	}
	value, err := fs.ReadFile(embedded, string(location))
	if err != nil {
		return nil, fmt.Errorf(
			"read embedded built-in file %q: %w",
			location,
			err,
		)
	}
	return append([]byte(nil), value...), nil
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
