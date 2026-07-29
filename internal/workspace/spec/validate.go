package spec

import (
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

func ValidateDiscoveryPreferences(
	value DiscoveryPreferences,
) error {
	seenLocators := make(map[basespec.Locator]struct{})
	for _, locator := range value.AdditionalLocators {
		if err := basespec.ValidateLocator(locator, false); err != nil {
			return err
		}
		if _, duplicate := seenLocators[locator]; duplicate {
			return fmt.Errorf(
				"%w: duplicate discovery locator %q",
				basespec.ErrInvalid,
				locator,
			)
		}
		seenLocators[locator] = struct{}{}
	}

	seenRoots := make(map[basespec.Locator]struct{})
	for _, root := range value.AdditionalRoots {
		if err := basespec.ValidateLocator(root.Root, true); err != nil {
			return err
		}
		if _, duplicate := seenRoots[root.Root]; duplicate {
			return fmt.Errorf(
				"%w: duplicate discovery root %q",
				basespec.ErrInvalid,
				root.Root,
			)
		}
		seenRoots[root.Root] = struct{}{}
		seenPatterns := make(map[string]struct{}, len(root.IncludePatterns))
		for _, pattern := range root.IncludePatterns {
			if err := ValidateIncludePattern(pattern); err != nil {
				return err
			}
			if _, duplicate := seenPatterns[pattern]; duplicate {
				return fmt.Errorf("%w: duplicate include pattern %q", basespec.ErrInvalid, pattern)
			}
			seenPatterns[pattern] = struct{}{}
		}
	}
	return nil
}

// ValidateIncludePattern validates a source-relative glob. It deliberately
// rejects path traversal and host-path syntax before passing the pattern to
// path.Match.
func ValidateIncludePattern(pattern string) error {
	if err := basespec.ValidateRequiredText(
		"discovery pattern",
		pattern,
		basespec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if strings.HasPrefix(pattern, "/") ||
		strings.ContainsAny(pattern, `\:`) {
		return fmt.Errorf(
			"%w: discovery pattern contains a disallowed path character",
			basespec.ErrInvalid,
		)
	}
	for segment := range strings.SplitSeq(pattern, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf(
				"%w: discovery pattern contains an invalid path segment",
				basespec.ErrInvalid,
			)
		}
	}
	if _, err := path.Match(pattern, "candidate"); err != nil {
		return fmt.Errorf(
			"%w: invalid discovery pattern %q: %w",
			basespec.ErrInvalid,
			pattern,
			err,
		)
	}
	return nil
}
