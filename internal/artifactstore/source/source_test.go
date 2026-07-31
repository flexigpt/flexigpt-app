package source

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

const (
	sourceTestRootID basespec.RootID   = "019d3150-6a1a-7a6b-a34e-d9032342bc31"
	sourceTestID     basespec.SourceID = "019d3150-6a1b-7a6b-a34e-d9032342bc31"
)

func sourceTestValue() Source {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	return Source{
		ID:          sourceTestID,
		RootID:      sourceTestRootID,
		Kind:        "test.source",
		DisplayName: "Test source",
		Enabled:     true,
		Config:      json.RawMessage(`{"a":1}`),
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
}

func TestSourceCloneAndManagedPublicationNormalizationOwnMutableData(t *testing.T) {
	t.Parallel()

	value := sourceTestValue()
	retired := value.ModifiedAt.Add(time.Second)
	value.Enabled = false
	value.RetiredAt = &retired
	cloned := value.Clone()
	expectedRetiredAt := retired
	value.Config[2] = 'x'
	*value.RetiredAt = value.RetiredAt.Add(time.Second)
	if string(cloned.Config) != `{"a":1}` {
		t.Fatalf("Clone config=%q, want owned copy", cloned.Config)
	}
	if cloned.RetiredAt == nil || !cloned.RetiredAt.Equal(expectedRetiredAt) {
		t.Fatalf("Clone retiredAt=%v, want %v", cloned.RetiredAt, expectedRetiredAt)
	}

	publication := ManagedPackagePublication{
		Directory:          "packages/example",
		ExpectedGeneration: "generation-1",
		Files: []ManagedPackageFile{
			{Locator: "z.txt", Content: []byte("z")},
			{Locator: "a.txt", Content: []byte("a")},
		},
	}
	normalized, err := NormalizeManagedPackagePublication(publication)
	if err != nil {
		t.Fatalf("NormalizeManagedPackagePublication: %v", err)
	}
	if normalized.Files[0].Locator != "a.txt" || normalized.Files[1].Locator != "z.txt" {
		t.Fatalf("files were not sorted: %#v", normalized.Files)
	}
	publication.Files[0].Content[0] = 'X'
	if !bytes.Equal(normalized.Files[1].Content, []byte("z")) {
		t.Fatalf("normalized publication retained caller content: %#v", normalized.Files)
	}

	collision := publication
	collision.Files = []ManagedPackageFile{
		{Locator: "Readme.md", Content: []byte("one")},
		{Locator: "README.md", Content: []byte("two")},
	}
}

type sourceTestSnapshot struct {
	generation string
	mu         sync.Mutex
	closed     bool
}

func (s *sourceTestSnapshot) Generation() string { return s.generation }

func (s *sourceTestSnapshot) Stat(ctx context.Context, _ basespec.Locator) (Entry, error) {
	if err := ctx.Err(); err != nil {
		return Entry{}, err
	}
	return Entry{}, basespec.ErrNotFound
}

func (s *sourceTestSnapshot) ReadDir(ctx context.Context, _ basespec.Locator) ([]Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, basespec.ErrNotFound
}

func (s *sourceTestSnapshot) Open(ctx context.Context, _ basespec.Locator) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (s *sourceTestSnapshot) Confirm(ctx context.Context) error { return ctx.Err() }

func (s *sourceTestSnapshot) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

type sourceTestAdapter struct {
	kind       basespec.SourceKind
	generation string
	localPath  string

	mu           sync.Mutex
	opens        int
	publications []ManagedPackagePublication
	removals     int
}

func (a *sourceTestAdapter) Kind() basespec.SourceKind { return a.kind }

func (a *sourceTestAdapter) NormalizeConfig(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append(json.RawMessage(nil), raw...), nil
}

func (a *sourceTestAdapter) Open(ctx context.Context, value Source) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(value.Config) != 0 {
		value.Config[0] = '['
	}
	a.mu.Lock()
	a.opens++
	a.mu.Unlock()
	return &sourceTestSnapshot{generation: a.generation}, nil
}

func (a *sourceTestAdapter) ResolveLocalPath(
	ctx context.Context,
	_ Source,
	_ basespec.Locator,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return a.localPath, nil
}

func (a *sourceTestAdapter) PublishPackage(
	ctx context.Context,
	_ Source,
	publication ManagedPackagePublication,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.publications = append(a.publications, publication)
	return a.generation, nil
}

func (a *sourceTestAdapter) RemovePackage(
	ctx context.Context,
	_ Source,
	_ basespec.Locator,
	_ string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	a.removals++
	a.mu.Unlock()
	return nil
}

type sourceTestReader struct{ value Source }

func (r sourceTestReader) Get(
	context.Context,
	basespec.RootID,
	basespec.SourceID,
) (Source, error) {
	return r.value, nil
}

func TestRegistryAndRuntimeValidateCapabilitiesAndConcurrentOpens(t *testing.T) {
	t.Parallel()

	value := sourceTestValue()
	adapter := &sourceTestAdapter{
		kind:       value.Kind,
		generation: "generation-2",
		localPath:  filepath.Join(t.TempDir(), "document.txt"),
	}
	registry, err := NewRegistry(adapter)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	if kinds := registry.Kinds(); len(kinds) != 1 || kinds[0] != value.Kind {
		t.Fatalf("Kinds=%#v", kinds)
	}

	snapshot, err := registry.Open(t.Context(), value)
	if err != nil {
		t.Fatalf("Registry.Open: %v", err)
	}
	if snapshot.Generation() != "generation-2" {
		t.Fatalf("generation=%q", snapshot.Generation())
	}
	if err := snapshot.Close(); err != nil {
		t.Fatalf("snapshot close: %v", err)
	}
	if string(value.Config) != `{"a":1}` {
		t.Fatalf("adapter mutated caller source config: %q", value.Config)
	}

	publication := ManagedPackagePublication{
		Directory: "packages/example",
		Files: []ManagedPackageFile{
			{Locator: "z.txt", Content: []byte("z")},
			{Locator: "a.txt", Content: []byte("a")},
		},
	}
	generation, err := registry.PublishPackage(t.Context(), value, publication)
	if err != nil {
		t.Fatalf("PublishPackage: %v", err)
	}
	if generation != "generation-2" {
		t.Fatalf("published generation=%q", generation)
	}
	if err := registry.RemovePackage(t.Context(), value, "packages/example", generation); err != nil {
		t.Fatalf("RemovePackage: %v", err)
	}
	adapter.mu.Lock()
	if len(adapter.publications) != 1 || adapter.publications[0].Files[0].Locator != "a.txt" || adapter.removals != 1 {
		t.Fatalf("adapter calls: publications=%#v removals=%d", adapter.publications, adapter.removals)
	}
	adapter.mu.Unlock()

	runtime, err := NewRuntime(sourceTestReader{value: value}, registry)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	read, err := runtime.Get(t.Context(), value.RootID, value.ID)
	if err != nil {
		t.Fatalf("runtime Get: %v", err)
	}
	read.Config[2] = 'x'
	if string(value.Config) != `{"a":1}` {
		t.Fatalf("runtime Get returned caller-owned config: %q", value.Config)
	}
	localRuntime, ok := runtime.(LocalPathRuntime)
	if !ok {
		t.Fatal("runtime did not expose LocalPathRuntime")
	}
	location, err := localRuntime.ResolveLocalPath(t.Context(), read, "document.txt")
	if err != nil {
		t.Fatalf("ResolveLocalPath: %v", err)
	}
	if location != adapter.localPath || !filepath.IsAbs(location) {
		t.Fatalf("local path=%q", location)
	}

	const workers = 24
	var group sync.WaitGroup
	errorsSeen := make(chan error, workers)
	for range workers {
		group.Go(func() {
			snapshot, err := registry.Open(t.Context(), value)
			if err == nil {
				err = snapshot.Close()
			}
			if err != nil {
				errorsSeen <- err
			}
		})
	}
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatalf("concurrent Registry.Open: %v", err)
	}
}
