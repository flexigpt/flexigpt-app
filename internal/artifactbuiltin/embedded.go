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
	return EmbeddedPackages(topology.BuiltinEmbeddedPackageSkills)
}

//go:embed agents
var embeddedAgentsFS embed.FS

func EmbeddedAgentPackages() (fs.FS, error) {
	return EmbeddedPackages(topology.BuiltinEmbeddedPackageAgents)
}

//go:embed mcps
var embeddedMCPFS embed.FS

func EmbeddedMCPPackages() (fs.FS, error) {
	return EmbeddedPackages(topology.BuiltinEmbeddedPackageMCPs)
}

//go:embed tools
var embeddedToolsFS embed.FS

func EmbeddedToolPackages() (fs.FS, error) {
	return EmbeddedPackages(topology.BuiltinEmbeddedPackageTools)
}

//go:embed workspaces
var embeddedWorkspacesFS embed.FS

// EmbeddedWorkspacePackages supplies the application base Workspace policy.
// It is not installed as protected built-in content.
func EmbeddedWorkspacePackages() (fs.FS, error) {
	return embeddedSubtree(embeddedWorkspacesFS, "workspaces")
}

// EmbeddedPackages selects application-owned compiled-in content.
// Generic package enumeration and reading belong to ManagedPackage.
func EmbeddedPackages(packageSet string) (fs.FS, error) {
	var embedded fs.FS
	switch packageSet {
	case topology.BuiltinEmbeddedPackageSkills:
		embedded = embeddedSkillsFS
	case topology.BuiltinEmbeddedPackageAgents:
		embedded = embeddedAgentsFS
	case topology.BuiltinEmbeddedPackageMCPs:
		embedded = embeddedMCPFS
	case topology.BuiltinEmbeddedPackageTools:
		embedded = embeddedToolsFS
	default:
		return nil, fmt.Errorf(
			"%w: embedded package set %q is not compiled into the application",
			spec.ErrNotFound,
			packageSet,
		)
	}
	return embeddedSubtree(
		embedded,
		topology.MustBuiltinEmbeddedPackageRoot(packageSet),
	)
}

func embeddedSubtree(embedded fs.FS, root spec.Locator) (fs.FS, error) {
	if embedded == nil || !fs.ValidPath(string(root)) {
		return nil, fmt.Errorf("invalid embedded built-in root %q", root)
	}
	return fs.Sub(embedded, string(root))
}
