package compose

import "testing"

func TestStoreCloseCallsDeploymentShutdownOnce(t *testing.T) {
	t.Parallel()

	calls := 0
	store := &Store{shutdown: func() error {
		calls++
		return nil
	}}
	if err := store.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if calls != 1 {
		t.Fatalf("shutdown calls=%d, want 1", calls)
	}
}
