package uuidutil

import (
	"context"
	"errors"
	"testing"
)

func TestUUIDv7GeneratorProducesValidatedIDsAndHonorsCancellation(t *testing.T) {
	t.Parallel()

	generator := UUIDv7Generator{}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := generator.NewID(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("NewID canceled context error = %v, want context.Canceled", err)
	}

	value, err := generator.NewID(t.Context())
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	if err := ValidateUUIDv7("generated ID", value); err != nil {
		t.Fatalf("generated ID %q is invalid: %v", value, err)
	}
}
