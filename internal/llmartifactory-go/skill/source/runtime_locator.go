package source

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

// SourceDocumentLocator resolves the SKILL.md document used by a canonical
// Skill declaration. A nil portable locator means that the declaration-origin
// entry itself is SKILL.md. A local locator may identify SKILL.md directly or
// its containing Skill directory.
//
// URL, Git, package, archive, and command locators remain the responsibility
// of a locator resolver outside the local Skill runtime path.
func SourceDocumentLocator(
	locator *declaration.Locator,
	declarationLocator spec.Locator,
) (spec.Locator, error) {
	if locator == nil {
		if err := declarationLocator.Validate(false); err != nil {
			return "", err
		}
		return declarationLocator, nil
	}

	target, err := declaration.ResolveSourceRelativePathLocator(
		*locator,
		declarationLocator,
	)
	if err != nil {
		return "", err
	}
	if !IsSkillDefinitionFile(target) {
		target = spec.Locator(path.Join(
			string(target),
			string(SkillDefinitionFileName()),
		))
	}
	if err := target.Validate(false); err != nil {
		return "", err
	}
	return target, nil
}

// RuntimePackageLocator derives the Skill package directory from a verified
// SKILL.md Artifact binding. Artifact Store verifies source generation and the
// source content digest before exposing the returned local path.
func RuntimePackageLocator(
	locator spec.Locator,
	subresource spec.SubresourceLocator,
) (spec.Locator, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return "", err
	}
	if subresource != "" {
		return "", fmt.Errorf(
			"%w: Skill bindings cannot target a subresource",
			spec.ErrUnsupported,
		)
	}
	if !IsSkillDefinitionFile(locator) {
		return "", fmt.Errorf(
			"%w: Skill locator %q is not a configured Skill package document",
			spec.ErrInvalid,
			locator,
		)
	}

	directory := spec.Locator(path.Dir(string(locator)))
	if directory == "." {
		return directory, nil
	}
	if err := directory.ValidatePortable(false); err != nil {
		return "", err
	}
	return directory, nil
}
