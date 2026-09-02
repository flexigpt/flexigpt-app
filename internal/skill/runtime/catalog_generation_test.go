package runtime

import (
	"context"
	"testing"
	"time"
)

type blockingCatalogSource struct {
	started chan struct{}
	release chan struct{}
}

func (s *blockingCatalogSource) Skills(
	ctx context.Context,
	_ CatalogID,
) ([]SkillRegistration, error) {
	select {
	case s.started <- struct{}{}:
	default:
	}

	select {
	case <-s.release:
		return nil, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestSyncCatalogDoesNotRestoreAfterRemove(t *testing.T) {
	source := &blockingCatalogSource{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}

	service, err := New(WithCatalogSource(source))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() {
		if err := service.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	catalogID := CatalogID("test-catalog")
	syncResult := make(chan error, 1)
	go func() {
		syncResult <- service.SyncCatalog(t.Context(), catalogID)
	}()

	select {
	case <-source.started:
	case <-time.After(time.Second):
		t.Fatal("catalog source was not called")
	}

	if err := service.RemoveCatalog(t.Context(), catalogID); err != nil {
		t.Fatalf("RemoveCatalog: %v", err)
	}

	close(source.release)
	if err := <-syncResult; err != nil {
		t.Fatalf("SyncCatalog: %v", err)
	}

	service.catalogMu.Lock()
	_, restored := service.catalogs[catalogID]
	service.catalogMu.Unlock()

	if restored {
		t.Fatalf("stale sync restored catalog %q after removal", catalogID)
	}
}
