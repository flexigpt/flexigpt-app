package source

import (
	"fmt"
	"path"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
)

// SourceDocumentLocator resolves the SKILL.md document used by a canonical
// Skill declaration. A nil portable locator means that the declaration-origin
// entry itself is SKILL.md. A local locator may identify SKILL.md directly or
// its containing Skill directory.
//
// URL, Git, package, archive, and command locators remain the responsibility
// of a locator resolver outside the local Skill runtime path.
func SourceDocumentLocator(
	documents support.Documents,
	locator *declaration.Locator,
	declarationLocator storeSpec.Locator,
) (storeSpec.Locator, error) {
	if err := documents.Validate(); err != nil {
		return "", err
	}
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
	if !IsSkillDefinitionFile(documents, target) {
		target = storeSpec.Locator(path.Join(
			string(target),
			string(SkillDefinitionFileName(documents)),
		))
	}
	if err := target.Validate(false); err != nil {
		return "", err
	}
	return target, nil
}

// RuntimePackageLocator derives the Skill package directory from a verified
// configured Skill document Artifact binding. Artifact Store verifies source
// generation and source content evidence before exposing the local path.
func RuntimePackageLocator(
	locator storeSpec.Locator,
	subresource storeSpec.SubresourceLocator,
) (storeSpec.Locator, error) {
	if err := locator.ValidatePortable(false); err != nil {
		return "", err
	}
	if subresource != "" {
		return "", fmt.Errorf(
			"%w: Skill bindings cannot target a subresource",
			storeSpec.ErrUnsupported,
		)
	}

	directory := storeSpec.Locator(path.Dir(string(locator)))
	if directory == "." {
		return directory, nil
	}
	if err := directory.ValidatePortable(false); err != nil {
		return "", err
	}
	return directory, nil
}
