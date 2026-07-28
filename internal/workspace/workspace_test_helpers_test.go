package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
)

type workspaceTestFixture struct {
	components *system.Components
	api        *API
	root       root.Root
}

func newWorkspaceTestFixture(t *testing.T) *workspaceTestFixture {
	t.Helper()
	components, err := system.Open(t.Context(), system.Config{
		BaseDirectory: t.TempDir(),
		Decoders:      BuiltinDecoders(),
	})
	if err != nil {
		t.Fatalf("open Artifact Store components: %v", err)
	}
	t.Cleanup(func() {
		if err := components.Close(); err != nil {
			t.Errorf("close Artifact Store components: %v", err)
		}
	})

	rootValue, err := components.Roots.Create(t.Context(), root.RootDraft{
		DisplayName: "Workspace test root",
	})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	api, err := New(Dependencies{
		Roots:              components.Roots,
		Sources:            components.Sources,
		Collections:        components.Collections,
		Artifacts:          components.Artifacts,
		Refresh:            components.Refresh,
		Catalogs:           components.Catalogs,
		Definitions:        components.Definitions,
		SourceRuntime:      components.SourceRuntime,
		HasDecoder:         components.HasDecoder,
		DecoderFingerprint: components.DecoderFingerprint,
	}, DefaultConfig())
	if err != nil {
		t.Fatalf("create workspace API: %v", err)
	}
	t.Cleanup(func() {
		if err := api.Close(); err != nil {
			t.Errorf("close workspace API: %v", err)
		}
	})
	return &workspaceTestFixture{
		components: components,
		api:        api,
		root:       rootValue,
	}
}

func writeWorkspaceFixtureFiles(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	writeWorkspaceFixtureFile(t, filepath.Join(directory, "AGENTS.md"), "Follow repository instructions.\n")
	writeWorkspaceFixtureFile(t, filepath.Join(directory, "README.md"), "Repository readme.\n")
	skillDirectory := filepath.Join(directory, ".flexigpt", "skills", "weather")
	if err := os.MkdirAll(skillDirectory, 0o755); err != nil {
		t.Fatalf("create Skill directory: %v", err)
	}
	writeWorkspaceFixtureFile(
		t,
		filepath.Join(skillDirectory, "SKILL.md"),
		"---\nname: weather\ndescription: Weather guidance\ninsert: instructions\n---\nUse the weather Skill.\n",
	)
	return directory
}

func writeWorkspaceFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent for %q: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}
