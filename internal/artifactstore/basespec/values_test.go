package basespec

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
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

func TestValueValidationBoundariesAndPlatformSafety(t *testing.T) {
	t.Parallel()

	validID := "019d3150-6a12-7a6b-a34e-d9032342bc31"
	if err := ValidateRootID(RootID(validID)); err != nil {
		t.Fatalf("ValidateRootID(valid): %v", err)
	}

	tests := []struct {
		name string
		err  error
	}{
		{name: "upper-case UUID", err: ValidateRootID(RootID(strings.ToUpper(validID)))},
		{name: "trimmed text", err: ValidateRequiredText("text", " value", 16)},
		{name: "control text", err: ValidateRequiredText("text", "value\n", 16)},
		{name: "invalid UTF-8", err: ValidateRequiredText("text", string([]byte{0xff}), 16)},
		{name: "overlong text", err: ValidateRequiredText("text", strings.Repeat("a", 17), 16)},
		{name: "portable reserved basename", err: ValidatePortableLocator("CON.txt", false)},
		{name: "portable trailing dot", err: ValidatePortableLocator("package/name.", false)},
		{name: "portable trailing space", err: ValidatePortableLocator("package/name ", false)},
		{name: "portable invalid separator", err: ValidatePortableLocator(`package\\name`, false)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(test.err.Error(), "invalid") {
				t.Fatalf("error=%v, want wrapping ErrInvalid", test.err)
			}
		})
	}

	if err := ValidateRequiredText("text", strings.Repeat("a", 16), 16); err != nil {
		t.Fatalf("ValidateRequiredText at limit: %v", err)
	}
	if err := ValidatePortableLocator("packages/example/SKILL.md", false); err != nil {
		t.Fatalf("ValidatePortableLocator(valid): %v", err)
	}
	if err := ValidatePortableLocator(".", true); err != nil {
		t.Fatalf("ValidatePortableLocator(root): %v", err)
	}
	if err := ValidatePortableLocator(".", false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ValidatePortableLocator(file root) error=%v, want ErrInvalid", err)
	}
}

func TestValueValidationIsSafeForConcurrentCallers(t *testing.T) {
	t.Parallel()

	const workers = 32
	var group sync.WaitGroup
	errorsSeen := make(chan error, workers)
	for range workers {
		group.Go(func() {
			if err := ValidatePortableLocator("portable/path.json", false); err != nil {
				errorsSeen <- err
			}
			if err := cryptoutil.ValidateDigest(cryptoutil.DigestBytes([]byte("stable content"))); err != nil {
				errorsSeen <- err
			}
		})
	}
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatalf("concurrent validation failed: %v", err)
	}
}
