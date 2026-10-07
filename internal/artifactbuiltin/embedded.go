package artifactbuiltin

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
)

//go:embed skills
var embeddedSkillsFS embed.FS

func EmbeddedSkillPackages() (fs.FS, error) {
	return embeddedSubtree(
		embeddedSkillsFS,
		topology.MustBuiltinEmbeddedPackageRoot(
			topology.BuiltinEmbeddedPackageSkills,
		),
	)
}

//go:embed agents
var embeddedAgentsFS embed.FS

func EmbeddedAgentPackages() (fs.FS, error) {
	return embeddedSubtree(
		embeddedAgentsFS,
		topology.MustBuiltinEmbeddedPackageRoot(
			topology.BuiltinEmbeddedPackageAgents,
		),
	)
}

//go:embed mcps
var embeddedMCPFS embed.FS

func EmbeddedMCPPackages() (fs.FS, error) {
	return embeddedSubtree(
		embeddedMCPFS,
		topology.MustBuiltinEmbeddedPackageRoot(
			topology.BuiltinEmbeddedPackageMCPs,
		),
	)
}

//go:embed tools
var embeddedToolsFS embed.FS

func EmbeddedToolPackages() (fs.FS, error) {
	return embeddedSubtree(
		embeddedToolsFS,
		topology.MustBuiltinEmbeddedPackageRoot(
			topology.BuiltinEmbeddedPackageTools,
		),
	)
}

//go:embed workspaces
var embeddedWorkspacesFS embed.FS

// EmbeddedWorkspacePackages supplies the application base Workspace policy.
// It is not installed as protected built-in content.
func EmbeddedWorkspacePackages() (fs.FS, error) {
	return embeddedSubtree(embeddedWorkspacesFS, "workspaces")
}

func embeddedSubtree(embedded fs.FS, root spec.Locator) (fs.FS, error) {
	if embedded == nil || !fs.ValidPath(string(root)) {
		return nil, fmt.Errorf("invalid embedded built-in root %q", root)
	}
	return fs.Sub(embedded, string(root))
}
