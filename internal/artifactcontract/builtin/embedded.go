package builtin

import (
	"embed"
	"fmt"
	"io/fs"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

//go:embed skills
var embeddedSkillsFS embed.FS

//go:embed agents
var embeddedAgentsFS embed.FS

//go:embed mcps
var embeddedMCPFS embed.FS

//go:embed tools
var embeddedToolsFS embed.FS

//go:embed workspaces
var embeddedWorkspacesFS embed.FS

func EmbeddedSkillPackages() (fs.FS, error) {
	return EmbeddedPackages(documentTopology.BuiltinEmbeddedPackageSkills)
}

func EmbeddedAgentPackages() (fs.FS, error) {
	return EmbeddedPackages(documentTopology.BuiltinEmbeddedPackageAgents)
}

func EmbeddedMCPPackages() (fs.FS, error) {
	return EmbeddedPackages(documentTopology.BuiltinEmbeddedPackageMCPs)
}

func EmbeddedToolPackages() (fs.FS, error) {
	return EmbeddedPackages(documentTopology.BuiltinEmbeddedPackageTools)
}

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
	case documentTopology.BuiltinEmbeddedPackageSkills:
		embedded = embeddedSkillsFS
	case documentTopology.BuiltinEmbeddedPackageAgents:
		embedded = embeddedAgentsFS
	case documentTopology.BuiltinEmbeddedPackageMCPs:
		embedded = embeddedMCPFS
	case documentTopology.BuiltinEmbeddedPackageTools:
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
		documentTopology.MustBuiltinEmbeddedPackageRoot(packageSet),
	)
}

func embeddedSubtree(embedded fs.FS, root spec.Locator) (fs.FS, error) {
	if embedded == nil || !fs.ValidPath(string(root)) {
		return nil, fmt.Errorf("invalid embedded built-in root %q", root)
	}
	return fs.Sub(embedded, string(root))
}
