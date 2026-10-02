package basespec

import (
	"strings"
	"sync"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestValueValidationBoundariesAndPlatformSafety(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "trimmed text", err: ValidateRequiredText("text", " value", 16)},
		{name: "control text", err: ValidateRequiredText("text", "value\n", 16)},
		{name: "invalid UTF-8", err: ValidateRequiredText("text", string([]byte{0xff}), 16)},
		{name: "overlong text", err: ValidateRequiredText("text", strings.Repeat("a", 17), 16)},
		{name: "invalid storage key", err: StorageKey("Not portable").Validate()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.err != nil && !strings.Contains(test.err.Error(), "invalid") {
				t.Fatalf("error=%v, want wrapping ErrInvalid", test.err)
			}
		})
	}

	if err := ValidateRequiredText("text", strings.Repeat("a", 16), 16); err != nil {
		t.Fatalf("ValidateRequiredText at limit: %v", err)
	}
	if err := StorageKey("personal").Validate(); err != nil {
		t.Fatalf("Validate StorageKey(valid): %v", err)
	}
}

func TestValueValidationIsSafeForConcurrentCallers(t *testing.T) {
	t.Parallel()

	const workers = 32
	var group sync.WaitGroup
	errorsSeen := make(chan error, workers)
	for range workers {
		group.Go(func() {
			if err := Locator("portable/path.json").ValidatePortable(false); err != nil {
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
