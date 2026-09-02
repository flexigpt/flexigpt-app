package runtime

import (
	"context"
	"testing"
	"time"
)

func TestAgentOperationsDoNotWaitForCatalogLock(t *testing.T) {
	service, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := service.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	tests := []struct {
		name string
		call func(context.Context) error
	}{
		{
			name: "supports run script",
			call: func(context.Context) error {
				_ = service.SupportsRunScript()
				return nil
			},
		},
		{
			name: "lists agent skills",
			call: func(ctx context.Context) error {
				_, err := service.ListAgentSkills(ctx, nil)
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service.catalogMu.Lock()
			defer service.catalogMu.Unlock()

			result := make(chan error, 1)
			go func() {
				result <- test.call(t.Context())
			}()

			select {
			case err := <-result:
				if err != nil {
					t.Fatalf("operation: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("agent operation waited for catalog reconciliation")
			}
		})
	}
}
