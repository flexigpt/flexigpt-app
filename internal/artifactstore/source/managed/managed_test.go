package managed

import (
	"errors"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func TestManagedPackagePublicationRejectsReservedDirectory(t *testing.T) {
	t.Parallel()

	adapter, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = adapter.PublishPackage(t.Context(), managedTestSource(), source.ManagedPackagePublication{
		Directory: ".artifactstore-staging/private",
		Files:     []source.ManagedPackageFile{{Locator: "file.txt", Content: []byte("x")}},
	})
	if !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("reserved directory error=%v, want ErrInvalid", err)
	}
}

func managedTestSource() source.Source {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	return source.Source{
		ID:          "019d3150-6a20-7a6b-a34e-d9032342bc31",
		RootID:      "019d3150-6a21-7a6b-a34e-d9032342bc31",
		Kind:        Kind,
		DisplayName: "Managed fixture",
		Enabled:     true,
		Config:      []byte(`{}`),
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
}
