package spec

import (
	"fmt"

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
		if err := root.Validate(); err != nil {
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
	}
	return nil
}
