package artifactbuiltin

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed skills
var embeddedSkillsFS embed.FS

//go:embed mcps
var embeddedMCPFS embed.FS

func ReadEmbeddedSkillRegistry() ([]byte, error) {
	return readEmbeddedFile(
		embeddedSkillsFS,
		EmbeddedSkillDataRoot+"/"+EmbeddedSkillRegistryFileName,
	)
}

func ReadEmbeddedMCPRegistry() ([]byte, error) {
	return readEmbeddedFile(
		embeddedMCPFS,
		EmbeddedMCPDataRoot+"/"+EmbeddedMCPRegistryFileName,
	)
}

func EmbeddedSkillPackages() (fs.FS, error) {
	return embeddedSubtree(embeddedSkillsFS, EmbeddedSkillDataRoot)
}

func EmbeddedMCPArtifactPackages() (fs.FS, error) {
	return embeddedSubtree(embeddedMCPFS, EmbeddedMCPDataRoot)
}

func readEmbeddedFile(
	embedded fs.FS,
	location string,
) ([]byte, error) {
	if embedded == nil || !fs.ValidPath(location) {
		return nil, fmt.Errorf("invalid embedded built-in file %q", location)
	}
	value, err := fs.ReadFile(embedded, location)
	if err != nil {
		return nil, fmt.Errorf("read embedded built-in file %q: %w", location, err)
	}
	return append([]byte(nil), value...), nil
}

func embeddedSubtree(
	embedded fs.FS,
	root string,
) (fs.FS, error) {
	if embedded == nil || !fs.ValidPath(root) {
		return nil, fmt.Errorf("invalid embedded built-in root %q", root)
	}
	info, err := fs.Stat(embedded, root)
	if err != nil {
		return nil, fmt.Errorf("stat embedded built-in root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf(
			"embedded built-in root %q is not a directory",
			root,
		)
	}
	return fs.Sub(embedded, root)
}
