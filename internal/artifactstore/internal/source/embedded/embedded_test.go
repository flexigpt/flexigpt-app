package embedded

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func TestEmbeddedAdapterNormalizesAndReadsImmutableProvider(t *testing.T) {
	provider := fstest.MapFS{
		"assets/one.txt":     &fstest.MapFile{Data: []byte("one")},
		"assets/dir/two.txt": &fstest.MapFile{Data: []byte("two")},
	}
	adapter, err := New(map[string]fs.FS{"fixture": provider})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	raw := json.RawMessage(`{"root":"assets","providerKey":"fixture"}`)
	normalized, err := adapter.NormalizeConfig(t.Context(), raw)
	if err != nil {
		t.Fatalf("NormalizeConfig: %v", err)
	}
	if string(normalized) != `{"providerKey":"fixture","root":"assets"}` {
		t.Fatalf("normalized config=%s", normalized)
	}
	value := embeddedTestSource(normalized)
	snapshot, err := adapter.Open(t.Context(), value)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	entries, err := snapshot.ReadDir(t.Context(), ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 || entries[0].Locator != "dir" || entries[1].Locator != "one.txt" {
		t.Fatalf("entries=%#v", entries)
	}
	reader, err := snapshot.Open(t.Context(), "one.txt")
	if err != nil {
		t.Fatalf("Open one.txt: %v", err)
	}
	content, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read errors: read=%v close=%v", readErr, closeErr)
	}
	if string(content) != "one" {
		t.Fatalf("content=%q", content)
	}
	if err := snapshot.Confirm(t.Context()); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if _, err := snapshot.Stat(t.Context(), "missing.txt"); !errors.Is(err, basespec.ErrNotFound) {
		t.Fatalf("missing Stat error=%v", err)
	}
	if err := snapshot.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := snapshot.Open(t.Context(), "one.txt"); !errors.Is(err, basespec.ErrClosed) {
		t.Fatalf("Open after Close error=%v", err)
	}
}

func TestEmbeddedAdapterRejectsUnavailableProviders(t *testing.T) {
	t.Parallel()

	if _, err := New(map[string]fs.FS{"missing": nil}); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("nil provider error=%v, want ErrInvalid", err)
	}
	adapter, err := New(nil)
	if err != nil {
		t.Fatalf("New empty: %v", err)
	}
	if _, err := adapter.NormalizeConfig(
		t.Context(),
		[]byte(`{"providerKey":"missing"}`),
	); !errors.Is(
		err,
		basespec.ErrSourceUnavailable,
	) {
		t.Fatalf("unavailable config error=%v", err)
	}
}

func embeddedTestSource(config json.RawMessage) source.Source {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	return source.Source{
		ID:             "019d3150-6a1e-7a6b-a34e-d9032342bc31",
		RootID:         "019d3150-6a1f-7a6b-a34e-d9032342bc31",
		RootStorageKey: "test-root",
		StorageKey:     "embedded-fixture",
		Kind:           basespec.SourceKindEmbeddedDirectory,
		DisplayName:    "Embedded fixture",
		Enabled:        true,
		Config:         config,
		Revision:       1,
		CreatedAt:      now,
		ModifiedAt:     now,
	}
}
