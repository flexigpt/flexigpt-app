package system_test

import (
	"encoding/json"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
)

func TestManagedPackagePublicationAdvancesSourceRevisionOnlyOnChange(
	t *testing.T,
) {
	t.Parallel()

	ctx := t.Context()
	components, err := system.Open(ctx, system.Config{
		BaseDirectory: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("open artifact system: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := components.Close(); closeErr != nil {
			t.Errorf("close artifact system: %v", closeErr)
		}
	})

	rootValue, err := components.Roots.Create(ctx, root.RootDraft{
		DisplayName: "Managed package test",
	})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	rawConfig, err := json.Marshal(map[string]any{})
	if err != nil {
		t.Fatalf("marshal managed source config: %v", err)
	}
	sourceValue, err := components.Sources.Create(ctx, rootValue.ID, source.Draft{
		Kind:        managed.Kind,
		DisplayName: "Managed packages",
		Enabled:     true,
		Config:      rawConfig,
	})
	if err != nil {
		t.Fatalf("create managed source: %v", err)
	}

	first, err := components.PublishManagedPackage(
		ctx,
		rootValue.ID,
		sourceValue.ID,
		sourceValue.Revision,
		source.ManagedPackagePublication{
			Directory: artifactstore.Locator("items/first"),
			Files: []source.ManagedPackageFile{{
				Locator: "SKILL.md",
				Content: []byte("first package"),
			}},
		},
	)
	if err != nil {
		t.Fatalf("publish first package: %v", err)
	}
	if first.Source.Revision != sourceValue.Revision+1 {
		t.Fatalf(
			"first revision=%d, want %d",
			first.Source.Revision,
			sourceValue.Revision+1,
		)
	}

	idempotent, err := components.PublishManagedPackage(
		ctx,
		rootValue.ID,
		sourceValue.ID,
		first.Source.Revision,
		source.ManagedPackagePublication{
			Directory:          artifactstore.Locator("items/first"),
			ExpectedGeneration: first.Generation,
			Files: []source.ManagedPackageFile{{
				Locator: "SKILL.md",
				Content: []byte("first package"),
			}},
		},
	)
	if err != nil {
		t.Fatalf("repeat first package: %v", err)
	}
	if idempotent.Source.Revision != first.Source.Revision {
		t.Fatalf(
			"idempotent revision=%d, want %d",
			idempotent.Source.Revision,
			first.Source.Revision,
		)
	}

	second, err := components.PublishManagedPackage(
		ctx,
		rootValue.ID,
		sourceValue.ID,
		idempotent.Source.Revision,
		source.ManagedPackagePublication{
			Directory: artifactstore.Locator("items/second"),
			Files: []source.ManagedPackageFile{{
				Locator: "payload.txt",
				Content: []byte("second package"),
			}},
		},
	)
	if err != nil {
		t.Fatalf("publish second package: %v", err)
	}
	if second.Source.Revision != idempotent.Source.Revision+1 {
		t.Fatalf(
			"second revision=%d, want %d",
			second.Source.Revision,
			idempotent.Source.Revision+1,
		)
	}
}
