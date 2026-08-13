package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureStoreLayoutCreatesOnlyGenericStoreRoots(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := ensureStoreLayout(base); err != nil {
		t.Fatalf("ensureStoreLayout: %v", err)
	}

	for _, location := range []string{
		filepath.Join(base, StoreManifestFileName),
		filepath.Join(base, StoreContentDirectoryName),
	} {
		if _, err := os.Stat(location); err != nil {
			t.Fatalf("missing %q: %v", location, err)
		}
	}

	for _, forbidden := range []string{
		"definitions",
		"managed-sources",
		"packages",
	} {
		if _, err := os.Stat(filepath.Join(base, forbidden)); !os.IsNotExist(err) {
			t.Fatalf("unexpected generic layout directory %q: %v", forbidden, err)
		}
	}
}
