package domain

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
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
	declarationLocator model.Locator,
) (model.Locator, error) {
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
		target = model.Locator(path.Join(
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
	locator model.Locator,
	subresource model.SubresourceLocator,
) (model.Locator, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return "", err
	}
	if subresource != "" {
		return "", fmt.Errorf(
			"%w: Skill bindings cannot target a subresource",
			model.ErrUnsupported,
		)
	}
	if !IsSkillDefinitionFile(locator) {
		return "", fmt.Errorf(
			"%w: Skill locator %q is not a configured Skill package document",
			model.ErrInvalid,
			locator,
		)
	}

	directory := model.Locator(path.Dir(string(locator)))
	if directory == "." {
		return directory, nil
	}
	if err := directory.ValidatePortable(false); err != nil {
		return "", err
	}
	return directory, nil
}
