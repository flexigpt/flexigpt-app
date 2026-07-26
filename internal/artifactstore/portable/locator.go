package portable

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

// DocumentBaseLocator returns the source-relative directory containing a
// portable document. A document in the source root has "." as its base.
func DocumentBaseLocator(
	document artifactstore.Locator,
) (artifactstore.Locator, error) {
	if err := artifactstore.ValidateLocator(document, false); err != nil {
		return "", fmt.Errorf("portable document locator: %w", err)
	}

	base := artifactstore.Locator(path.Dir(string(document)))
	if err := artifactstore.ValidateLocator(base, true); err != nil {
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
	base artifactstore.Locator,
	relative artifactstore.Locator,
) (artifactstore.Locator, error) {
	return resolveRelativeLocator(base, relative, false)
}

// ResolveRelativeDirectoryLocator resolves a portable directory locator
// relative to a containing portable document or package directory. Unlike a
// file locator, "." is valid and denotes the containing directory itself.
func ResolveRelativeDirectoryLocator(
	base artifactstore.Locator,
	relative artifactstore.Locator,
) (artifactstore.Locator, error) {
	return resolveRelativeLocator(base, relative, true)
}

func resolveRelativeLocator(
	base artifactstore.Locator,
	relative artifactstore.Locator,
	allowRelativeRoot bool,
) (artifactstore.Locator, error) {
	if err := artifactstore.ValidatePortableLocator(base, true); err != nil {
		return "", fmt.Errorf("portable base locator: %w", err)
	}
	if err := artifactstore.ValidatePortableLocator(relative, allowRelativeRoot); err != nil {
		return "", fmt.Errorf("portable relative locator: %w", err)
	}

	var resolved artifactstore.Locator
	switch {
	case base == ".":
		resolved = relative
	case relative == ".":
		resolved = base
	default:
		resolved = artifactstore.Locator(
			path.Join(string(base), string(relative)),
		)
	}

	if err := artifactstore.ValidatePortableLocator(resolved, allowRelativeRoot); err != nil {
		return "", fmt.Errorf("resolved portable locator: %w", err)
	}
	return resolved, nil
}
