package fsdir

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func fsdirTestSource(root string, config json.RawMessage) source.Source {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	return source.Source{
		ID:          "019d3150-6a1c-7a6b-a34e-d9032342bc31",
		RootID:      "019d3150-6a1d-7a6b-a34e-d9032342bc31",
		Kind:        Kind,
		DisplayName: "Filesystem fixture",
		Enabled:     true,
		Config:      config,
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
}

func TestFilesystemAdapterUsesPortableLocatorsAndDetectsChanges(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "document.txt"), []byte("first"), 0o600); err != nil {
		t.Fatalf("write document: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatalf("make .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("ignored"), 0o600); err != nil {
		t.Fatalf("write .git config: %v", err)
	}

	adapter := New()
	raw, err := json.Marshal(Config{RootPath: root})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	normalized, err := adapter.NormalizeConfig(t.Context(), raw)
	if err != nil {
		t.Fatalf("NormalizeConfig: %v", err)
	}
	value := fsdirTestSource(root, normalized)
	snapshot, err := adapter.Open(t.Context(), value)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer snapshot.Close()

	entries, err := snapshot.ReadDir(t.Context(), ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Locator != "document.txt" || !entries[0].IsRegular {
		t.Fatalf("root entries=%#v", entries)
	}
	reader, err := snapshot.Open(t.Context(), "document.txt")
	if err != nil {
		t.Fatalf("Open document: %v", err)
	}
	content, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read document errors: read=%v close=%v", readErr, closeErr)
	}
	if string(content) != "first" {
		t.Fatalf("document content=%q", content)
	}
	location, err := adapter.ResolveLocalPath(t.Context(), value, "document.txt")
	if err != nil {
		t.Fatalf("ResolveLocalPath: %v", err)
	}
	if !filepath.IsAbs(location) || filepath.Base(location) != "document.txt" {
		t.Fatalf("resolved path=%q", location)
	}

	if err := os.WriteFile(filepath.Join(root, "document.txt"), []byte("second"), 0o600); err != nil {
		t.Fatalf("rewrite document: %v", err)
	}
	if err := snapshot.Confirm(t.Context()); !errors.Is(err, artifactstore.ErrConflict) {
		t.Fatalf("Confirm after mutation error=%v, want ErrConflict", err)
	}
	if err := snapshot.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := snapshot.Stat(t.Context(), "document.txt"); !errors.Is(err, artifactstore.ErrClosed) {
		t.Fatalf("Stat after Close error=%v, want ErrClosed", err)
	}
}

func TestFilesystemAdapterRejectsUnportableConfigurationAndPolicy(t *testing.T) {
	t.Parallel()

	adapter := New()
	if _, err := adapter.NormalizeConfig(
		t.Context(),
		[]byte(`{"rootPath":"relative"}`),
	); !errors.Is(
		err,
		artifactstore.ErrInvalid,
	) {
		t.Fatalf("relative config error=%v, want ErrInvalid", err)
	}
	if _, err := NewWithTraversalPolicy(&TraversalPolicy{ExcludedDirectoryNames: []string{"."}}); err == nil {
		t.Fatal("NewWithTraversalPolicy accepted invalid excluded directory")
	}
}
