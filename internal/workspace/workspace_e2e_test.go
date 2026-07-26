package workspace_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
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

	currentWorkspace, err := api.GetWorkspace(
		ctx,
		&workspace.GetWorkspaceRequest{
			Workspace: created.Body.Workspace,
		},
	)
	if err != nil {
		t.Fatalf("get Workspace before retirement: %v", err)
	}
	retired, err := api.RetireWorkspace(
		ctx,
		&workspace.RetireWorkspaceRequest{
			Workspace:        created.Body.Workspace,
			ExpectedRevision: currentWorkspace.Body.Revision,
		},
	)
	if err != nil {
		t.Fatalf("retire Workspace: %v", err)
	}
	if retired.Body == nil || retired.Body.Revision == 0 {
		t.Fatal("retire Workspace returned an invalid response")
	}
	if _, err := api.PurgeWorkspace(
		ctx,
		&workspace.PurgeWorkspaceRequest{
			Workspace:        created.Body.Workspace,
			ExpectedRevision: retired.Body.Revision,
		},
	); err != nil {
		t.Fatalf("purge retired Workspace: %v", err)
	}
	_, err = api.GetWorkspace(ctx, &workspace.GetWorkspaceRequest{
		Workspace: created.Body.Workspace,
	})
	if !errors.Is(err, artifactstore.ErrCollectionNotFound) {
		t.Fatalf("get purged Workspace error=%v, want collection not found", err)
	}
}

func TestEmptyWorkspaceAttachedMarkdownSuppressionRoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows filesystem integration test")
	}
	t.Parallel()

	ctx := t.Context()
	libraryDirectory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(libraryDirectory, "notes.md"),
		[]byte("# Attached library\n\nUse the attached project context.\n"),
		0o600,
	); err != nil {
		t.Fatalf("write attached Markdown: %v", err)
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
		DisplayName: "Attached source test",
	})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	sourceConfig, err := json.Marshal(fsdir.Config{
		RootPath: libraryDirectory,
	})
	if err != nil {
		t.Fatalf("encode filesystem source config: %v", err)
	}
	librarySource, err := components.Sources.Create(
		ctx,
		rootValue.ID,
		source.Draft{
			Kind:        fsdir.Kind,
			DisplayName: "Attached library",
			Enabled:     true,
			Config:      sourceConfig,
		},
	)
	if err != nil {
		t.Fatalf("create attached filesystem source: %v", err)
	}

	created, err := api.CreateEmptyWorkspace(
		ctx,
		&workspace.CreateEmptyWorkspaceRequest{
			RootID: rootValue.ID,
			Body: &workspace.CreateEmptyWorkspaceRequestBody{
				DisplayName: "Empty workspace",
			},
		},
	)
	if err != nil {
		t.Fatalf("create empty Workspace: %v", err)
	}
	if created.Body == nil {
		t.Fatal("create empty Workspace returned nil body")
	}

	attached, err := api.AttachWorkspaceSource(
		ctx,
		&workspace.AttachWorkspaceSourceRequest{
			Workspace: created.Body.Workspace,
			Body: &workspace.AttachWorkspaceSourceRequestBody{
				ExpectedCollectionRevision: created.Body.Revision,
				SourceID:                   librarySource.ID,
				Role:                       artifactstore.AttachmentRole("library"),
				Enabled:                    true,
			},
		},
	)
	if err != nil {
		t.Fatalf("attach library source: %v", err)
	}
	if attached.Body == nil {
		t.Fatal("attach library source returned nil body")
	}

	if _, err := api.RefreshWorkspace(
		ctx,
		&workspace.RefreshWorkspaceRequest{
			Workspace: created.Body.Workspace,
		},
	); err != nil {
		t.Fatalf("refresh attached Workspace: %v", err)
	}

	artifacts, err := api.ListWorkspaceArtifacts(
		ctx,
		&workspace.ListWorkspaceArtifactsRequest{
			Workspace: created.Body.Workspace,
		},
	)
	if err != nil {
		t.Fatalf("list attached Workspace Artifacts: %v", err)
	}
	if artifacts.Body == nil {
		t.Fatal("list attached Workspace Artifacts returned nil body")
	}

	var observed *workspace.WorkspaceArtifactView
	for index := range artifacts.Body.Artifacts {
		value := &artifacts.Body.Artifacts[index]
		if value.SourceID == librarySource.ID &&
			value.Locator == "notes.md" {
			observed = value
			break
		}
	}
	if observed == nil {
		t.Fatalf("attached Markdown was not automatically adopted: %#v", artifacts.Body.Artifacts)
	}

	if _, err := api.UnadoptWorkspaceArtifact(
		ctx,
		&workspace.UnadoptWorkspaceArtifactRequest{
			Workspace:        created.Body.Workspace,
			Artifact:         observed.Artifact,
			ExpectedRevision: observed.Revision,
			Suppress:         true,
		},
	); err != nil {
		t.Fatalf("unadopt and suppress attached Artifact: %v", err)
	}
	if _, err := api.RefreshWorkspace(
		ctx,
		&workspace.RefreshWorkspaceRequest{
			Workspace: created.Body.Workspace,
		},
	); err != nil {
		t.Fatalf("refresh suppressed Workspace: %v", err)
	}

	artifacts, err = api.ListWorkspaceArtifacts(
		ctx,
		&workspace.ListWorkspaceArtifactsRequest{
			Workspace: created.Body.Workspace,
		},
	)
	if err != nil {
		t.Fatalf("list suppressed Workspace Artifacts: %v", err)
	}
	for _, value := range artifacts.Body.Artifacts {
		if value.SourceID == librarySource.ID &&
			value.Locator == "notes.md" {
			t.Fatal("suppressed occurrence was automatically recreated")
		}
	}

	suppressions, err := api.ListWorkspaceSuppressions(
		ctx,
		&workspace.ListWorkspaceSuppressionsRequest{
			Workspace: created.Body.Workspace,
		},
	)
	if err != nil {
		t.Fatalf("list Workspace suppressions: %v", err)
	}
	if suppressions.Body == nil || len(suppressions.Body.Suppressions) != 1 {
		t.Fatalf("suppressions=%#v, want one suppression", suppressions.Body)
	}
	suppression := suppressions.Body.Suppressions[0]

	if _, err := api.UnsuppressWorkspaceBinding(
		ctx,
		&workspace.UnsuppressWorkspaceBindingRequest{
			Workspace:        created.Body.Workspace,
			Binding:          suppression.Binding,
			ExpectedRevision: suppression.Revision,
		},
	); err != nil {
		t.Fatalf("unsuppress attached Artifact binding: %v", err)
	}
	if _, err := api.RefreshWorkspace(
		ctx,
		&workspace.RefreshWorkspaceRequest{
			Workspace: created.Body.Workspace,
		},
	); err != nil {
		t.Fatalf("refresh unsuppressed Workspace: %v", err)
	}

	artifacts, err = api.ListWorkspaceArtifacts(
		ctx,
		&workspace.ListWorkspaceArtifactsRequest{
			Workspace: created.Body.Workspace,
		},
	)
	if err != nil {
		t.Fatalf("list restored Workspace Artifacts: %v", err)
	}
	for _, value := range artifacts.Body.Artifacts {
		if value.SourceID == librarySource.ID &&
			value.Locator == "notes.md" {
			return
		}
	}
	t.Fatal("unsuppressed attached Markdown was not automatically adopted")
}
