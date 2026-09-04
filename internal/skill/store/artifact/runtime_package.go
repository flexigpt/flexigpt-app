package artifact

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// RuntimePackageLocator applies only Agent Skill binding rules. Artifact Store
// owns source generation verification, digest verification, snapshot lifecycle,
// and native path resolution.
func RuntimePackageLocator(
	locator basespec.Locator,
	subresource basespec.SubresourceLocator,
) (basespec.Locator, error) {
	if subresource != "" {
		return "", fmt.Errorf(
			"%w: Agent Skill bindings cannot target a subresource",
			basespec.ErrUnsupported,
		)
	}
	if basespec.Locator(path.Base(string(locator))) != artifactbuiltin.AgentSkillDefinitionFileName {
		return "", fmt.Errorf(
			"%w: Agent Skill locator %q is not %q",
			basespec.ErrInvalid,
			locator,
			artifactbuiltin.AgentSkillDefinitionFileName,
		)
	}

	packageLocator := basespec.Locator(path.Dir(string(locator)))
	if packageLocator == "." {
		return "", fmt.Errorf(
			"%w: Agent Skill package cannot be the Source root",
			basespec.ErrInvalid,
		)
	}

	return packageLocator, nil
}
