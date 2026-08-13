package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
)

func TestEnsureStoreLayoutCreatesOnlyGenericStoreRoots(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := ensureStoreLayout(base); err != nil {
		t.Fatalf("ensureStoreLayout: %v", err)
	}

	for _, location := range []string{
		filepath.Join(base, builtinSchema.ArtifactStoreManifestFileName),
		filepath.Join(base, builtinSchema.ArtifactStoreContentDirectoryName),
		filepath.Join(base, builtinSchema.ArtifactStoreStagingDirectoryName),
	} {
		if _, err := os.Stat(location); err != nil {
			t.Fatalf("missing %q: %v", location, err)
		}
	}

	for _, forbidden := range []string{
		"definitions",
		"managed-sources",
		"packages",
		".artifactstore-staging",
	} {
		if _, err := os.Stat(filepath.Join(base, forbidden)); !os.IsNotExist(err) {
			t.Fatalf("unexpected generic layout directory %q: %v", forbidden, err)
		}
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("read base: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			t.Fatalf("Artifact Store created hidden path %q", entry.Name())
		}
	}
}
