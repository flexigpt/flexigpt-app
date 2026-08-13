package managed

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func TestManagedPackagePublicationUsesSemanticAddress(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	contentBase := filepath.Join(base, "content")
	stagingBase := filepath.Join(base, "staging")
	adapter, err := New(contentBase, stagingBase)
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
		contentBase,
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
		contentBase,
		"test-root",
		"managed-fixture",
		"packages",
	)
	if _, err := os.Stat(legacyLocation); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy packages directory exists: %v", err)
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
