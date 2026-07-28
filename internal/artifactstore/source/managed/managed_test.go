package managed

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

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

func TestManagedPackagePublicationIsAtomicIdempotentAndConcurrent(t *testing.T) {
	adapter, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	value := managedTestSource()
	initialGeneration, err := adapter.BootstrapManagedSource(t.Context(), value)
	if err != nil {
		t.Fatalf("BootstrapManagedSource: %v", err)
	}
	publication := source.ManagedPackagePublication{
		Directory:          "packages/demo",
		ExpectedGeneration: initialGeneration,
		Files: []source.ManagedPackageFile{
			{Locator: "z.txt", Content: []byte("z")},
			{Locator: "nested/a.txt", Content: []byte("a")},
		},
	}
	generation, err := adapter.PublishPackage(t.Context(), value, publication)
	if err != nil {
		t.Fatalf("PublishPackage: %v", err)
	}
	if generation == initialGeneration {
		t.Fatal("package publication did not change source generation")
	}

	snapshot, err := adapter.Open(t.Context(), value)
	if err != nil {
		t.Fatalf("Open after publication: %v", err)
	}
	reader, err := snapshot.Open(t.Context(), "packages/demo/nested/a.txt")
	if err != nil {
		t.Fatalf("Open published file: %v", err)
	}
	content, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read published file errors: read=%v close=%v", readErr, closeErr)
	}
	if string(content) != "a" {
		t.Fatalf("published content=%q", content)
	}
	if err := snapshot.Close(); err != nil {
		t.Fatalf("close snapshot: %v", err)
	}

	const workers = 12
	var group sync.WaitGroup
	generations := make(chan string, workers)
	errorsSeen := make(chan error, workers)
	for range workers {
		group.Go(func() {
			result, err := adapter.PublishPackage(t.Context(), value, publication)
			if err != nil {
				errorsSeen <- err
				return
			}
			generations <- result
		})
	}
	group.Wait()
	close(generations)
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatalf("idempotent concurrent publication: %v", err)
	}
	for result := range generations {
		if result != generation {
			t.Fatalf("idempotent generation=%q, want %q", result, generation)
		}
	}

	different := publication
	different.Files = []source.ManagedPackageFile{{Locator: "z.txt", Content: []byte("different")}}
	if _, err := adapter.PublishPackage(
		t.Context(),
		value,
		different,
	); !errors.Is(
		err,
		artifactstore.ErrConflict,
	) {
		t.Fatalf("different package error=%v, want ErrConflict", err)
	}
	if err := adapter.RemovePackage(
		t.Context(),
		value,
		publication.Directory,
		"other-generation",
	); !errors.Is(
		err,
		artifactstore.ErrConflict,
	) {
		t.Fatalf("wrong generation removal error=%v, want ErrConflict", err)
	}
	if err := adapter.RemovePackage(t.Context(), value, publication.Directory, generation); err != nil {
		t.Fatalf("RemovePackage: %v", err)
	}
	removed, err := adapter.Open(t.Context(), value)
	if err != nil {
		t.Fatalf("Open after removal: %v", err)
	}
	defer removed.Close()
	if _, err := removed.Stat(t.Context(), publication.Directory); !errors.Is(err, artifactstore.ErrNotFound) {
		t.Fatalf("removed package Stat error=%v, want ErrNotFound", err)
	}
}

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
	if !errors.Is(err, artifactstore.ErrInvalid) {
		t.Fatalf("reserved directory error=%v, want ErrInvalid", err)
	}
}
