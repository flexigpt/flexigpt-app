package artifactstore

import (
	"strings"
	"testing"
)

func TestValidatePortableLocatorRejectsPlatformSpecificNames(t *testing.T) {
	t.Parallel()

	invalid := []Locator{
		"package/name<invalid",
		"package/name>invalid",
		`package/name"invalid`,
		"package/name|invalid",
		"package/name?invalid",
		"package/name*invalid",
		Locator(strings.Repeat("a", maxPortablePathSegmentBytes+1)),
	}

	for _, locator := range invalid {
		t.Run(string(locator), func(t *testing.T) {
			t.Parallel()
			if err := ValidatePortableLocator(locator, false); err == nil {
				t.Fatalf("ValidatePortableLocator(%q) succeeded", locator)
			}
		})
	}

	if err := ValidatePortableLocator(
		"packages/example/SKILL.md",
		false,
	); err != nil {
		t.Fatalf("ValidatePortableLocator(valid): %v", err)
	}
}
