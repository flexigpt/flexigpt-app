package workspace_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
)

func TestFilesystemWorkspaceRefreshContextSkillAndCurrentCatalogPin(
	t *testing.T,
) {
	if runtime.GOOS == "windows" {
		t.Skip("non-win test")
	}
	t.Parallel()

	ctx := t.Context()
	project := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(project, "AGENTS.md"),
		[]byte("# Project instructions\n\nUse the project conventions.\n"),
		0o600,
	); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	skillDirectory := filepath.Join(
		project,
		".flexigpt",
		"skills",
		"example",
	)
	if err := os.MkdirAll(skillDirectory, 0o700); err != nil {
		t.Fatalf("create Skill directory: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDirectory, "SKILL.md"),
		[]byte("---\nname: example\ndescription: Example Skill\n---\n\nExample instructions.\n"),
		0o600,
	); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	components, err := system.Open(ctx, system.Config{
		BaseDirectory: t.TempDir(),
		Decoders:      workspace.BuiltinDecoders(),
	})
	if err != nil {
		t.Fatalf("open artifact system: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := components.Close(); closeErr != nil {
			t.Errorf("close artifact system: %v", closeErr)
		}
	})

	api, err := workspace.New(workspace.Dependencies{
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
	}, workspace.DefaultConfig())
	if err != nil {
		t.Fatalf("create Workspace API: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := api.Close(); closeErr != nil {
			t.Errorf("close Workspace API: %v", closeErr)
		}
	})

	rootValue, err := components.Roots.Create(ctx, root.RootDraft{
		DisplayName: "Workspace test",
	})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}

	created, err := api.CreateFilesystemWorkspace(
		ctx,
		&workspace.CreateFilesystemWorkspaceRequest{
			RootID: rootValue.ID,
			Body: &workspace.CreateFilesystemWorkspaceRequestBody{
				DisplayName: "Project",
				RootPath:    project,
			},
		},
	)
	if err != nil {
		t.Fatalf("create filesystem Workspace: %v", err)
	}
	if created.Body == nil {
		t.Fatal("create filesystem Workspace returned nil body")
	}

	refreshed, err := api.RefreshWorkspace(
		ctx,
		&workspace.RefreshWorkspaceRequest{
			Workspace: created.Body.Workspace,
		},
	)
	if err != nil {
		t.Fatalf("refresh Workspace: %v", err)
	}
	if refreshed.Body == nil || refreshed.Body.CatalogRevision == 0 {
		t.Fatal("refresh did not publish a catalog")
	}

	contextPlan, err := api.ComposeWorkspaceContext(
		ctx,
		&workspace.ComposeWorkspaceContextRequest{
			Workspace: created.Body.Workspace,
			Body:      &workspace.ComposeWorkspaceContextRequestBody{},
		},
	)
	if err != nil {
		t.Fatalf("compose Workspace Context: %v", err)
	}
	if contextPlan.Body == nil || contextPlan.Body.Prompt == "" {
		t.Fatal("expected AGENTS.md Context contribution")
	}

	skills, err := api.ListWorkspaceSkills(
		ctx,
		&workspace.ListWorkspaceSkillsRequest{
			Workspace: created.Body.Workspace,
		},
	)
	if err != nil {
		t.Fatalf("list Workspace Skills: %v", err)
	}
	if skills.Body == nil || len(skills.Body.Skills) != 1 {
		t.Fatalf("Workspace Skills=%#v, want one Skill", skills.Body)
	}

	loaded, err := api.LoadWorkspaceSkills(
		ctx,
		&workspace.LoadWorkspaceSkillsRequest{
			Workspace: created.Body.Workspace,
			Body: &workspace.LoadWorkspaceSkillsRequestBody{
				Artifacts: []artifactstore.ArtifactRef{
					skills.Body.Skills[0].Artifact,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("load Workspace Skill: %v", err)
	}
	if loaded.Body == nil || len(loaded.Body.Skills) != 1 {
		t.Fatalf("loaded Workspace Skills=%#v, want one Skill", loaded.Body)
	}

	records, err := components.Artifacts.ListByCollection(
		ctx,
		created.Body.Workspace,
	)
	if err != nil {
		t.Fatalf("list Artifact records: %v", err)
	}
	var observed artifact.Artifact
	for _, record := range records {
		if record.Binding.Locator == "AGENTS.md" {
			observed = record
			break
		}
	}
	if observed.ID == "" {
		t.Fatal("expected auto-adopted AGENTS.md Artifact")
	}

	if err := components.Artifacts.Unadopt(
		ctx,
		observed.Ref(),
		observed.Revision,
		false,
	); err != nil {
		t.Fatalf("unadopt current Context Artifact: %v", err)
	}

	current, err := api.GetWorkspace(
		ctx,
		&workspace.GetWorkspaceRequest{
			Workspace: created.Body.Workspace,
		},
	)
	if err != nil {
		t.Fatalf("get Workspace: %v", err)
	}
	pinned, err := components.Artifacts.Pin(ctx, artifact.PinRequest{
		Collection:                 created.Body.Workspace,
		ExpectedCollectionRevision: current.Body.Revision,
		Binding:                    observed.Binding,
		Name:                       "Pinned AGENTS",
		Enabled:                    true,
	})
	if err != nil {
		t.Fatalf("pin current Context Artifact: %v", err)
	}
	if pinned.State != artifact.StateAvailable ||
		pinned.ResolvedDefinition == nil {
		t.Fatalf("pinned Artifact=%#v, want available current Artifact", pinned)
	}
}
