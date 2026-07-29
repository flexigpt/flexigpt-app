package discovery

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// DocumentBaseLocator returns the source-relative directory containing a
// portable document. A document in the source root has "." as its base.
func DocumentBaseLocator(
	document basespec.Locator,
) (basespec.Locator, error) {
	if err := basespec.ValidateLocator(document, false); err != nil {
		return "", fmt.Errorf("portable document locator: %w", err)
	}

	base := basespec.Locator(path.Dir(string(document)))
	if err := basespec.ValidateLocator(base, true); err != nil {
		return "", fmt.Errorf("portable document base locator: %w", err)
	}
	return base, nil
}

// ResolveRelativeLocator resolves a portable file locator relative to a
// containing portable document or package directory.
//
// Portable locators must remain relative and cannot use "." or ".." path
// segments. This function does not resolve external URIs or embedded content.
func ResolveRelativeLocator(
	base basespec.Locator,
	relative basespec.Locator,
) (basespec.Locator, error) {
	return resolveRelativeLocator(base, relative, false)
}

// ResolveRelativeDirectoryLocator resolves a portable directory locator
// relative to a containing portable document or package directory. Unlike a
// file locator, "." is valid and denotes the containing directory itself.
func ResolveRelativeDirectoryLocator(
	base basespec.Locator,
	relative basespec.Locator,
) (basespec.Locator, error) {
	return resolveRelativeLocator(base, relative, true)
}

func resolveRelativeLocator(
	base basespec.Locator,
	relative basespec.Locator,
	allowRelativeRoot bool,
) (basespec.Locator, error) {
	if err := basespec.ValidatePortableLocator(base, true); err != nil {
		return "", fmt.Errorf("portable base locator: %w", err)
	}
	if err := basespec.ValidatePortableLocator(relative, allowRelativeRoot); err != nil {
		return "", fmt.Errorf("portable relative locator: %w", err)
	}

	var resolved basespec.Locator
	switch {
	case base == ".":
		resolved = relative
	case relative == ".":
		resolved = base
	default:
		resolved = basespec.Locator(
			path.Join(string(base), string(relative)),
		)
	}

	if err := basespec.ValidatePortableLocator(resolved, allowRelativeRoot); err != nil {
		return "", fmt.Errorf("resolved portable locator: %w", err)
	}
	return resolved, nil
}
