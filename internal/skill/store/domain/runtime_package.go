package domain

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// RuntimePackageLocator derives the Skill package directory from a verified
// SKILL.md Artifact binding. Artifact Store verifies source generation and the
// source content digest before exposing the returned local path.
func RuntimePackageLocator(
	locator basespec.Locator,
	subresource basespec.SubresourceLocator,
) (basespec.Locator, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return "", err
	}
	if subresource != "" {
		return "", fmt.Errorf(
			"%w: Skill bindings cannot target a subresource",
			basespec.ErrUnsupported,
		)
	}
	if path.Base(string(locator)) != string(SkillDefinitionFileName) {
		return "", fmt.Errorf(
			"%w: Skill locator %q is not %q",
			basespec.ErrInvalid,
			locator,
			SkillDefinitionFileName,
		)
	}

	directory := basespec.Locator(path.Dir(string(locator)))
	if directory == "." {
		return directory, nil
	}
	if err := directory.ValidatePortable(false); err != nil {
		return "", err
	}
	return directory, nil
}
