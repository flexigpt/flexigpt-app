package managed

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func TestManagedPackagePublicationUsesSemanticAddress(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	adapter, err := New(base)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	value := managedTestSource()
	_, err = adapter.PublishPackage(
		t.Context(),
		value,
		source.ManagedPackagePublication{
			Address: source.ManagedPackageAddress{
				Kind:    "agent.skill",
				Name:    "example",
				Version: "unversioned",
			},
			Files: []source.ManagedPackageFile{{
				Locator: "SKILL.md",
				Content: []byte("name: example"),
			}},
		},
	)
	if err != nil {
		t.Fatalf("PublishPackage: %v", err)
	}

	location := filepath.Join(
		base,
		"test-root",
		"managed-fixture",
		"agent.skill",
		"example",
		"unversioned",
		"SKILL.md",
	)
	if _, err := os.Stat(location); err != nil {
		t.Fatalf("semantic package file is unavailable: %v", err)
	}

	legacyLocation := filepath.Join(
		base,
		"test-root",
		"managed-fixture",
		"packages",
	)
	if _, err := os.Stat(legacyLocation); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy packages directory exists: %v", err)
	}
}

func TestManagedPackagePublicationRejectsReservedStagingPath(t *testing.T) {
	t.Parallel()

	adapter, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = adapter.PublishPackage(t.Context(), managedTestSource(), source.ManagedPackagePublication{
		Address: source.ManagedPackageAddress{
			Kind:    "test.package",
			Name:    "example",
			Version: "v1",
		},
		Files: []source.ManagedPackageFile{{
			Locator: ".artifactstore-staging/private.txt",
			Content: []byte("x"),
		}},
	})
	if !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("reserved directory error=%v, want ErrInvalid", err)
	}
}

func managedTestSource() source.Source {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	return source.Source{
		ID:             "019d3150-6a20-7a6b-a34e-d9032342bc31",
		RootID:         "019d3150-6a21-7a6b-a34e-d9032342bc31",
		RootStorageKey: "test-root",
		StorageKey:     "managed-fixture",
		Kind:           Kind,
		DisplayName:    "Managed fixture",
		Enabled:        true,
		Config:         []byte(`{}`),
		Revision:       1,
		CreatedAt:      now,
		ModifiedAt:     now,
	}
}
