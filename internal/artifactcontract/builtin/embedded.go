package builtin

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
