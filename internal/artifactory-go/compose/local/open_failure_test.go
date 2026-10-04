package local

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
)

func TestOpenFailureRetainsCallerSecretValueOwnership(t *testing.T) {
	t.Parallel()

	values := &closeTrackingValueStore{}
	_, err := Open(t.Context(), Config{
		BaseDirectory:           t.TempDir(),
		SecretValues:            values,
		AdditionalSourceDrivers: []driver.Driver{nil},
	})
	if err == nil {
		t.Fatal("Open unexpectedly succeeded with a nil additional Source driver")
	}
	if values.closeCalls != 0 {
		t.Fatalf("failed Open closed caller-owned secret values %d times", values.closeCalls)
	}
	if err := values.Close(); err != nil {
		t.Fatalf("caller cleanup: %v", err)
	}
	if values.closeCalls != 1 {
		t.Fatalf("caller cleanup calls=%d, want 1", values.closeCalls)
	}
}
