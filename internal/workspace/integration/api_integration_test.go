package integration

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
	"github.com/flexigpt/flexigpt-app/internal/workspace/selection"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

func TestAPIFilesystemWorkspaceRefreshRuntimeAndAttachmentLifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non win test")
	}
	fixture := newWorkspaceTestFixture(t)
	directory := writeWorkspaceFixtureFiles(t)

	created, err := fixture.api.CreateFilesystemWorkspace(t.Context(), &workspace.CreateFilesystemWorkspaceRequest{
		RootID: fixture.root.ID,
		Body: &workspace.CreateFilesystemWorkspaceRequestBody{
			DisplayName: "Repository",
			Description: "Workspace integration test",
			RootPath:    directory,
			Discovery:   workspace.WorkspaceDiscovery{IncludeReadme: true},
		},
	})
	if err != nil || created == nil || created.Body == nil {
		t.Fatalf("CreateFilesystemWorkspace response=%#v err=%v", created, err)
	}
	workspaceView := *created.Body
	if workspaceView.Mode != spec.ModeFilesystem || workspaceView.PrimarySourceID == "" ||
		workspaceView.PrimaryPath != filepath.Clean(directory) || len(workspaceView.Attachments) != 1 ||
		workspaceView.Attachments[0].SourceKind != string(fsdir.Kind) ||
		workspaceView.Attachments[0].Path != filepath.Clean(directory) {
		t.Fatalf("created workspace=%#v", workspaceView)
	}

	listed, err := fixture.api.ListWorkspaces(t.Context(), &workspace.ListWorkspacesRequest{RootID: fixture.root.ID})
	if err != nil || listed == nil || listed.Body == nil || len(listed.Body.Workspaces) != 1 ||
		listed.Body.Workspaces[0].Workspace != workspaceView.Workspace {
		t.Fatalf("ListWorkspaces response=%#v err=%v", listed, err)
	}
	refs, err := fixture.api.WorkspaceRefs(t.Context())
	if err != nil || len(refs) != 1 || refs[0] != workspaceView.Workspace {
		t.Fatalf("WorkspaceRefs=%#v err=%v", refs, err)
	}

	refreshed, err := fixture.api.RefreshWorkspace(
		t.Context(),
		&workspace.RefreshWorkspaceRequest{Workspace: workspaceView.Workspace},
	)
	if err != nil || refreshed == nil || refreshed.Body == nil || refreshed.Body.CatalogRevision == 0 ||
		refreshed.Body.Candidates < 3 || len(refreshed.Body.CreatedArtifacts) < 3 {
		t.Fatalf("RefreshWorkspace response=%#v err=%v", refreshed, err)
	}
	catalogView, err := fixture.api.GetWorkspaceCatalog(
		t.Context(),
		&workspace.GetWorkspaceCatalogRequest{Workspace: workspaceView.Workspace},
	)
	if err != nil || catalogView == nil || catalogView.Body == nil || !catalogView.Body.CatalogCurrent ||
		len(catalogView.Body.Resources) < 3 || len(catalogView.Body.ValidOccurrences) < 3 {
		t.Fatalf("GetWorkspaceCatalog response=%#v err=%v", catalogView, err)
	}

	artifacts, err := fixture.api.ListWorkspaceArtifacts(
		t.Context(),
		&workspace.ListWorkspaceArtifactsRequest{Workspace: workspaceView.Workspace},
	)
	if err != nil || artifacts == nil || artifacts.Body == nil || len(artifacts.Body.Artifacts) < 3 {
		t.Fatalf("ListWorkspaceArtifacts response=%#v err=%v", artifacts, err)
	}
	var contextArtifact, skillArtifact workspace.WorkspaceArtifactView
	for _, value := range artifacts.Body.Artifacts {
		switch value.Kind {
		case "workspace.context":
			if contextArtifact.Artifact.ArtifactID == "" {
				contextArtifact = value
			}
		case "agent.skill":
			skillArtifact = value
		}
	}
	if contextArtifact.Artifact.ArtifactID == "" || skillArtifact.Artifact.ArtifactID == "" {
		t.Fatalf("expected Context and Skill Artifacts, got %#v", artifacts.Body.Artifacts)
	}

	gotArtifact, err := fixture.api.GetWorkspaceArtifact(t.Context(), &workspace.GetWorkspaceArtifactRequest{
		Workspace: workspaceView.Workspace,
		Artifact:  contextArtifact.Artifact,
	})
	if err != nil || gotArtifact == nil || gotArtifact.Body == nil ||
		gotArtifact.Body.Artifact != contextArtifact.Artifact {
		t.Fatalf("GetWorkspaceArtifact response=%#v err=%v", gotArtifact, err)
	}
	loadPlan, err := fixture.api.ComposeWorkspaceLoadPlan(t.Context(), &workspace.ComposeWorkspaceLoadPlanRequest{
		Workspace: workspaceView.Workspace,
		Body: &workspace.ComposeWorkspaceLoadPlanRequestBody{
			Artifacts: []artifact.ArtifactRef{contextArtifact.Artifact, skillArtifact.Artifact},
		},
	})
	if err != nil || loadPlan == nil || loadPlan.Body == nil || len(loadPlan.Body.Items) != 2 {
		t.Fatalf("ComposeWorkspaceLoadPlan response=%#v err=%v", loadPlan, err)
	}

	removed, err := fixture.api.UnadoptWorkspaceArtifact(t.Context(), &workspace.UnadoptWorkspaceArtifactRequest{
		Workspace:        workspaceView.Workspace,
		Artifact:         contextArtifact.Artifact,
		ExpectedRevision: contextArtifact.Revision,
	})
	if err != nil || removed == nil || removed.Body == nil || removed.Body.Artifact != contextArtifact.Artifact {
		t.Fatalf("UnadoptWorkspaceArtifact response=%#v err=%v", removed, err)
	}
	adopted, err := fixture.api.AdoptWorkspaceOccurrence(t.Context(), &workspace.AdoptWorkspaceOccurrenceRequest{
		Workspace: workspaceView.Workspace,
		Body: &workspace.AdoptWorkspaceOccurrenceRequestBody{
			ExpectedCatalogRevision: catalogView.Body.CatalogRevision,
			Occurrence: workspace.WorkspaceOccurrenceRef{
				SourceID:           contextArtifact.SourceID,
				Locator:            contextArtifact.Locator,
				SubresourceLocator: contextArtifact.SubresourceLocator,
			},
			Name:    "Re-adopted Context",
			Enabled: true,
		},
	})
	if err != nil || adopted == nil || adopted.Body == nil || adopted.Body.Artifact == contextArtifact.Artifact {
		t.Fatalf("AdoptWorkspaceOccurrence response=%#v err=%v", adopted, err)
	}
	contextArtifact = *adopted.Body

	disabledRecord, err := fixture.api.SetWorkspaceArtifactEnabled(
		t.Context(),
		&workspace.SetWorkspaceArtifactEnabledRequest{
			Workspace: workspaceView.Workspace,
			Artifact:  contextArtifact.Artifact,
			Body: &workspace.SetWorkspaceArtifactEnabledRequestBody{
				ExpectedRevision: contextArtifact.Revision,
				Enabled:          false,
			},
		},
	)
	if err != nil || disabledRecord == nil || disabledRecord.Body == nil || disabledRecord.Body.Enabled {
		t.Fatalf("SetWorkspaceArtifactEnabled(false) response=%#v err=%v", disabledRecord, err)
	}
	contextArtifact = *disabledRecord.Body
	enabledRecord, err := fixture.api.SetWorkspaceArtifactEnabled(
		t.Context(),
		&workspace.SetWorkspaceArtifactEnabledRequest{
			Workspace: workspaceView.Workspace,
			Artifact:  contextArtifact.Artifact,
			Body: &workspace.SetWorkspaceArtifactEnabledRequestBody{
				ExpectedRevision: contextArtifact.Revision,
				Enabled:          true,
			},
		},
	)
	if err != nil || enabledRecord == nil || enabledRecord.Body == nil || !enabledRecord.Body.Enabled {
		t.Fatalf("SetWorkspaceArtifactEnabled(true) response=%#v err=%v", enabledRecord, err)
	}
	contextArtifact = *enabledRecord.Body

	pinned, err := fixture.api.PinWorkspaceArtifact(t.Context(), &workspace.PinWorkspaceArtifactRequest{
		Workspace: workspaceView.Workspace,
		Body: &workspace.PinWorkspaceArtifactRequestBody{
			ExpectedCollectionRevision: workspaceView.Revision,
			Binding: artifact.SourceBinding{
				SourceID:     workspaceView.PrimarySourceID,
				Locator:      "manual.md",
				ExpectedKind: "workspace.context",
			},
			Name:    "Pinned Context",
			Enabled: true,
		},
	})
	if err != nil || pinned == nil || pinned.Body == nil || pinned.Body.State != artifact.StateMissing {
		t.Fatalf("PinWorkspaceArtifact response=%#v err=%v", pinned, err)
	}

	suppressed, err := fixture.api.SuppressWorkspaceBinding(t.Context(), &workspace.SuppressWorkspaceBindingRequest{
		Workspace: workspaceView.Workspace,
		Body: &workspace.SuppressWorkspaceBindingRequestBody{
			ExpectedCollectionRevision: workspaceView.Revision,
			Binding: artifact.SourceBinding{
				SourceID:     workspaceView.PrimarySourceID,
				Locator:      "suppressed.md",
				ExpectedKind: "workspace.context",
			},
		},
	})
	if err != nil || suppressed == nil || suppressed.Body == nil || suppressed.Body.Revision == 0 {
		t.Fatalf("SuppressWorkspaceBinding response=%#v err=%v", suppressed, err)
	}
	suppressions, err := fixture.api.ListWorkspaceSuppressions(
		t.Context(),
		&workspace.ListWorkspaceSuppressionsRequest{Workspace: workspaceView.Workspace},
	)
	if err != nil || suppressions == nil || suppressions.Body == nil || len(suppressions.Body.Suppressions) != 1 {
		t.Fatalf("ListWorkspaceSuppressions response=%#v err=%v", suppressions, err)
	}
	unsuppressed, err := fixture.api.UnsuppressWorkspaceBinding(
		t.Context(),
		&workspace.UnsuppressWorkspaceBindingRequest{
			Workspace:        workspaceView.Workspace,
			Binding:          suppressed.Body.Binding,
			ExpectedRevision: suppressed.Body.Revision,
		},
	)
	if err != nil || unsuppressed == nil || unsuppressed.Body == nil ||
		unsuppressed.Body.Binding != suppressed.Body.Binding {
		t.Fatalf("UnsuppressWorkspaceBinding response=%#v err=%v", unsuppressed, err)
	}
	if _, err := fixture.api.PurgeWorkspaceArtifact(t.Context(), &workspace.PurgeWorkspaceArtifactRequest{
		Workspace:        workspaceView.Workspace,
		Artifact:         pinned.Body.Artifact,
		ExpectedRevision: pinned.Body.Revision,
	}); err != nil {
		t.Fatalf("PurgeWorkspaceArtifact: %v", err)
	}

	contexts, err := fixture.api.ListWorkspaceContexts(
		t.Context(),
		&workspace.ListWorkspaceContextsRequest{Workspace: workspaceView.Workspace},
	)
	if err != nil || contexts == nil || contexts.Body == nil || len(contexts.Body.Contexts) != 2 {
		t.Fatalf("ListWorkspaceContexts response=%#v err=%v", contexts, err)
	}
	inspection, err := fixture.api.LoadWorkspaceContexts(t.Context(), &workspace.LoadWorkspaceContextsRequest{
		Workspace: workspaceView.Workspace,
		Body: &workspace.LoadWorkspaceContextsRequestBody{
			Artifacts: []artifact.ArtifactRef{contextArtifact.Artifact},
		},
	})
	if err != nil || inspection == nil || inspection.Body == nil || len(inspection.Body.Contributions) != 1 ||
		inspection.Body.Contributions[0].Artifact != contextArtifact.Artifact {
		t.Fatalf("LoadWorkspaceContexts response=%#v err=%v", inspection, err)
	}
	composed, err := fixture.api.ComposeWorkspaceContext(t.Context(), &workspace.ComposeWorkspaceContextRequest{
		Workspace: workspaceView.Workspace,
		Body:      &workspace.ComposeWorkspaceContextRequestBody{},
	})
	if err != nil || composed == nil || composed.Body == nil || len(composed.Body.Contributions) != 2 ||
		!strings.Contains(
			composed.Body.Prompt,
			"Follow repository instructions.",
		) || composed.Body.PromptBytes != len(composed.Body.Prompt) {
		t.Fatalf("ComposeWorkspaceContext response=%#v err=%v", composed, err)
	}

	skills, err := fixture.api.ListWorkspaceSkills(
		t.Context(),
		&workspace.ListWorkspaceSkillsRequest{Workspace: workspaceView.Workspace},
	)
	if err != nil || skills == nil || skills.Body == nil || len(skills.Body.Skills) != 1 ||
		skills.Body.Skills[0].Skill.Name != "weather" || skills.Body.Skills[0].MarkdownBody != "" {
		t.Fatalf("ListWorkspaceSkills response=%#v err=%v", skills, err)
	}
	loadedSkills, err := fixture.api.LoadWorkspaceSkills(t.Context(), &workspace.LoadWorkspaceSkillsRequest{
		Workspace: workspaceView.Workspace,
		Body:      &workspace.LoadWorkspaceSkillsRequestBody{Artifacts: []artifact.ArtifactRef{skillArtifact.Artifact}},
	})
	if err != nil || loadedSkills == nil || loadedSkills.Body == nil || len(loadedSkills.Body.Skills) != 1 ||
		!strings.Contains(loadedSkills.Body.Skills[0].MarkdownBody, "Use the weather Skill.") {
		t.Fatalf("LoadWorkspaceSkills response=%#v err=%v", loadedSkills, err)
	}

	resolved, err := fixture.api.ResolveWorkspaceResource(t.Context(), &workspace.ResolveWorkspaceResourceRequest{
		Workspace: workspaceView.Workspace,
		Body:      &workspace.ResolveWorkspaceResourceRequestBody{Artifact: new(contextArtifact.Artifact)},
	})
	if err != nil || resolved == nil || resolved.Body == nil ||
		resolved.Body.Resource.Artifact.Artifact != contextArtifact.Artifact {
		t.Fatalf("ResolveWorkspaceResource response=%#v err=%v", resolved, err)
	}

	cs, err := selection.NewConversationResolver(fixture.api)
	if err != nil {
		t.Fatalf("convo selection returned err %v", err)
	}
	conversation, err := cs.ResolveConversationSelection(t.Context(), selection.ConversationSelection{
		Workspace: workspaceView.Workspace,
		ContextRefs: []selection.ConversationResourceSelectionRef{{
			Artifact: contextArtifact.Artifact,
		}},
		SkillRefs: []selection.ConversationSkillSelectionRef{
			{
				ConversationResourceSelectionRef: selection.ConversationResourceSelectionRef{
					Artifact: skillArtifact.Artifact,
				},
			},
		},
	})
	if err != nil || conversation.Usage.Status != selection.ConversationSelectionReady ||
		len(conversation.Usage.Contexts) != 1 || len(conversation.Usage.Skills) != 1 ||
		!strings.Contains(conversation.Prompt, "Follow repository instructions.") {
		t.Fatalf("ResolveConversationSelection resolution=%#v err=%v", conversation, err)
	}

	disabled, err := fixture.api.SetWorkspaceArtifactRuntimeDisabled(
		t.Context(),
		&workspace.SetWorkspaceArtifactRuntimeDisabledRequest{
			Workspace: workspaceView.Workspace,
			Artifact:  contextArtifact.Artifact,
			Body: &workspace.SetWorkspaceArtifactRuntimeDisabledRequestBody{
				ExpectedRevision: contextArtifact.Revision,
				RuntimeDisabled:  true,
			},
		},
	)
	if err != nil || disabled == nil || disabled.Body == nil || !disabled.Body.RuntimeDisabled {
		t.Fatalf("SetWorkspaceArtifactRuntimeDisabled response=%#v err=%v", disabled, err)
	}
	composed, err = fixture.api.ComposeWorkspaceContext(t.Context(), &workspace.ComposeWorkspaceContextRequest{
		Workspace: workspaceView.Workspace,
		Body: &workspace.ComposeWorkspaceContextRequestBody{
			Artifacts: []artifact.ArtifactRef{contextArtifact.Artifact},
		},
	})
	if err != nil || len(composed.Body.Contributions) != 0 || len(composed.Body.Decisions) != 1 ||
		composed.Body.Decisions[0].Status != "denied" {
		t.Fatalf("runtime-disabled composition=%#v err=%v", composed, err)
	}

	current, err := fixture.api.GetWorkspace(
		t.Context(),
		&workspace.GetWorkspaceRequest{Workspace: workspaceView.Workspace},
	)
	if err != nil || current == nil || current.Body == nil {
		t.Fatalf("GetWorkspace response=%#v err=%v", current, err)
	}
	updated, err := fixture.api.UpdateWorkspace(t.Context(), &workspace.UpdateWorkspaceRequest{
		Workspace: workspaceView.Workspace,
		Body: &workspace.UpdateWorkspaceRequestBody{
			ExpectedRevision: current.Body.Revision,
			DisplayName:      "Repository updated",
			Description:      "Updated description",
			Enabled:          true,
			Discovery:        current.Body.Discovery,
		},
	})
	if err != nil || updated == nil || updated.Body == nil || updated.Body.Revision != current.Body.Revision+1 ||
		updated.Body.DisplayName != "Repository updated" {
		t.Fatalf("UpdateWorkspace response=%#v err=%v", updated, err)
	}
	workspaceView = *updated.Body

	attachedDirectory := t.TempDir()
	writeWorkspaceFixtureFile(t, filepath.Join(attachedDirectory, "README.md"), "Attached library.\n")
	config, err := json.Marshal(fsdir.Config{RootPath: attachedDirectory})
	if err != nil {
		t.Fatalf("marshal attached source config: %v", err)
	}
	attachedSource, err := fixture.components.Sources.Create(
		t.Context(),
		fixture.root.ID,
		sourceDraftForWorkspaceTest(config),
	)
	if err != nil {
		t.Fatalf("create attached source: %v", err)
	}
	attached, err := fixture.api.AttachWorkspaceSource(t.Context(), &workspace.AttachWorkspaceSourceRequest{
		Workspace: workspaceView.Workspace,
		Body: &workspace.AttachWorkspaceSourceRequestBody{
			ExpectedCollectionRevision: workspaceView.Revision,
			SourceID:                   attachedSource.ID,
			Role:                       spec.RoleLibrary,
			Enabled:                    true,
		},
	})
	if err != nil || attached == nil || attached.Body == nil || len(attached.Body.Attachments) != 2 {
		t.Fatalf("AttachWorkspaceSource response=%#v err=%v", attached, err)
	}
	workspaceView = *attached.Body
	attachment := findWorkspaceAttachment(t, workspaceView, attachedSource.ID)
	falseValue := false
	changedAttachment, err := fixture.api.UpdateWorkspaceAttachment(
		t.Context(),
		&workspace.UpdateWorkspaceAttachmentRequest{
			Workspace: workspaceView.Workspace,
			SourceID:  attachedSource.ID,
			Body: &workspace.UpdateWorkspaceAttachmentRequestBody{
				ExpectedCollectionRevision: workspaceView.Revision,
				ExpectedAttachmentRevision: attachment.Revision,
				Role:                       spec.RoleLibrary,
				Enabled:                    true,
				Settings:                   workspace.WorkspaceAttachmentSettings{Recursive: &falseValue},
			},
		},
	)
	if err != nil || changedAttachment == nil || changedAttachment.Body == nil {
		t.Fatalf("UpdateWorkspaceAttachment response=%#v err=%v", changedAttachment, err)
	}
	workspaceView = *changedAttachment.Body
	attachment = findWorkspaceAttachment(t, workspaceView, attachedSource.ID)
	detached, err := fixture.api.DetachWorkspaceSource(t.Context(), &workspace.DetachWorkspaceSourceRequest{
		Workspace:                  workspaceView.Workspace,
		SourceID:                   attachedSource.ID,
		ExpectedCollectionRevision: workspaceView.Revision,
		ExpectedAttachmentRevision: attachment.Revision,
	})
	if err != nil || detached == nil || detached.Body == nil || len(detached.Body.Attachments) != 1 {
		t.Fatalf("DetachWorkspaceSource response=%#v err=%v", detached, err)
	}
	workspaceView = *detached.Body

	concurrentWorkspaceUpdate(t, fixture.api, workspaceView)
}

func TestAPIEmptyWorkspacePrimaryTransitionsReferencesAndPurge(t *testing.T) {
	fixture := newWorkspaceTestFixture(t)

	empty, err := fixture.api.CreateEmptyWorkspace(t.Context(), &workspace.CreateEmptyWorkspaceRequest{
		RootID: fixture.root.ID,
		Body:   &workspace.CreateEmptyWorkspaceRequestBody{DisplayName: "Empty workspace"},
	})
	if err != nil || empty == nil || empty.Body == nil || empty.Body.Mode != spec.ModeEmpty {
		t.Fatalf("CreateEmptyWorkspace response=%#v err=%v", empty, err)
	}
	workspaceView := *empty.Body

	sourceDirectory := t.TempDir()
	config, err := json.Marshal(fsdir.Config{RootPath: sourceDirectory})
	if err != nil {
		t.Fatalf("marshal primary source config: %v", err)
	}
	primarySource, err := fixture.components.Sources.Create(
		t.Context(),
		fixture.root.ID,
		sourceDraftForWorkspaceTest(config),
	)
	if err != nil {
		t.Fatalf("create primary source: %v", err)
	}
	set, err := fixture.api.SetWorkspacePrimarySource(t.Context(), &workspace.SetWorkspacePrimarySourceRequest{
		Workspace: workspaceView.Workspace,
		Body: &workspace.SetWorkspacePrimarySourceRequestBody{
			ExpectedCollectionRevision: workspaceView.Revision,
			SourceID:                   primarySource.ID,
		},
	})
	if err != nil || set == nil || set.Body == nil || set.Body.Mode != spec.ModeFilesystem ||
		set.Body.PrimarySourceID != primarySource.ID {
		t.Fatalf("SetWorkspacePrimarySource response=%#v err=%v", set, err)
	}
	workspaceView = *set.Body
	replacementSource, err := fixture.components.Sources.Create(
		t.Context(),
		fixture.root.ID,
		sourceDraftForWorkspaceTest(config),
	)
	if err != nil {
		t.Fatalf("create replacement primary source: %v", err)
	}
	replaced, err := fixture.api.ReplaceWorkspacePrimarySource(
		t.Context(),
		&workspace.ReplaceWorkspacePrimarySourceRequest{
			Workspace: workspaceView.Workspace,
			Body: &workspace.ReplaceWorkspacePrimarySourceRequestBody{
				ExpectedCollectionRevision: workspaceView.Revision,
				PreviousSourceID:           primarySource.ID,
				ExpectedPreviousAttachmentRevision: findWorkspaceAttachment(
					t,
					workspaceView,
					primarySource.ID,
				).Revision,
				SourceID: replacementSource.ID,
			},
		},
	)
	if err != nil || replaced == nil || replaced.Body == nil || replaced.Body.PrimarySourceID != replacementSource.ID {
		t.Fatalf("ReplaceWorkspacePrimarySource response=%#v err=%v", replaced, err)
	}
	workspaceView = *replaced.Body
	primaryAttachment := findWorkspaceAttachment(t, workspaceView, replacementSource.ID)
	cleared, err := fixture.api.SetWorkspacePrimarySource(t.Context(), &workspace.SetWorkspacePrimarySourceRequest{
		Workspace: workspaceView.Workspace,
		Body: &workspace.SetWorkspacePrimarySourceRequestBody{
			ExpectedCollectionRevision:         workspaceView.Revision,
			PreviousSourceID:                   replacementSource.ID,
			ExpectedPreviousAttachmentRevision: primaryAttachment.Revision,
			Clear:                              true,
		},
	})
	if err != nil || cleared == nil || cleared.Body == nil || cleared.Body.Mode != spec.ModeEmpty ||
		cleared.Body.PrimarySourceID != "" {
		t.Fatalf("clear primary response=%#v err=%v", cleared, err)
	}
	workspaceView = *cleared.Body

	second, err := fixture.api.CreateEmptyWorkspace(t.Context(), &workspace.CreateEmptyWorkspaceRequest{
		RootID: fixture.root.ID,
		Body:   &workspace.CreateEmptyWorkspaceRequestBody{DisplayName: "Second workspace"},
	})
	if err != nil || second == nil || second.Body == nil {
		t.Fatalf("create second Workspace response=%#v err=%v", second, err)
	}
	refs, err := fixture.api.WorkspaceRefs(t.Context())
	if err != nil || len(refs) != 2 || refs[0].RootID != fixture.root.ID || refs[1].RootID != fixture.root.ID ||
		refs[0].CollectionID > refs[1].CollectionID {
		t.Fatalf("WorkspaceRefs=%#v err=%v", refs, err)
	}

	retired, err := fixture.api.RetireWorkspace(t.Context(), &workspace.RetireWorkspaceRequest{
		Workspace:        workspaceView.Workspace,
		ExpectedRevision: workspaceView.Revision,
	})
	if err != nil || retired == nil || retired.Body == nil || retired.Body.Revision != workspaceView.Revision+1 {
		t.Fatalf("RetireWorkspace response=%#v err=%v", retired, err)
	}
	purged, err := fixture.api.PurgeWorkspace(t.Context(), &workspace.PurgeWorkspaceRequest{
		Workspace:        workspaceView.Workspace,
		ExpectedRevision: retired.Body.Revision,
	})
	if err != nil || purged == nil || purged.Body == nil || purged.Body.Workspace != workspaceView.Workspace {
		t.Fatalf("PurgeWorkspace response=%#v err=%v", purged, err)
	}
	if _, err := fixture.api.GetWorkspace(
		t.Context(),
		&workspace.GetWorkspaceRequest{Workspace: workspaceView.Workspace},
	); !errors.Is(
		err,
		basespec.ErrCollectionNotFound,
	) {
		t.Fatalf("GetWorkspace after purge error=%v, want ErrCollectionNotFound", err)
	}
}

func TestAPIRejectsUninitializedClosedAndNilRequests(t *testing.T) {
	var nilAPI *workspace.API
	if _, err := nilAPI.GetWorkspace(
		t.Context(),
		&workspace.GetWorkspaceRequest{},
	); !errors.Is(
		err,
		spec.ErrInvalidWorkspace,
	) {
		t.Fatalf("nil API GetWorkspace error=%v", err)
	}

	fixture := newWorkspaceTestFixture(t)
	for _, test := range []struct {
		name string
		call func() error
	}{
		{name: "create filesystem", call: func() error { _, err := fixture.api.CreateFilesystemWorkspace(t.Context(), nil); return err }},
		{name: "create empty", call: func() error { _, err := fixture.api.CreateEmptyWorkspace(t.Context(), nil); return err }},
		{name: "get", call: func() error { _, err := fixture.api.GetWorkspace(t.Context(), nil); return err }},
		{name: "list", call: func() error { _, err := fixture.api.ListWorkspaces(t.Context(), nil); return err }},
		{name: "update", call: func() error { _, err := fixture.api.UpdateWorkspace(t.Context(), nil); return err }},
		{name: "compose context", call: func() error { _, err := fixture.api.ComposeWorkspaceContext(t.Context(), nil); return err }},
		{name: "load skills", call: func() error { _, err := fixture.api.LoadWorkspaceSkills(t.Context(), nil); return err }},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, spec.ErrInvalidWorkspace) {
				t.Fatalf("error=%v, want ErrInvalidWorkspace", err)
			}
		})
	}
	if fixture.api.SkillAdapter() == nil {
		t.Fatal("SkillAdapter is nil before Close")
	}
	if err := fixture.api.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if fixture.api.SkillAdapter() != nil {
		t.Fatal("SkillAdapter remained available after Close")
	}
	if _, err := fixture.api.ListWorkspaces(
		t.Context(),
		&workspace.ListWorkspacesRequest{RootID: fixture.root.ID},
	); !errors.Is(
		err,
		spec.ErrInvalidWorkspace,
	) {
		t.Fatalf("closed API error=%v", err)
	}
}

func sourceDraftForWorkspaceTest(config json.RawMessage) source.Draft {
	return source.Draft{
		Kind:        fsdir.Kind,
		DisplayName: "Filesystem source",
		Enabled:     true,
		Config:      config,
	}
}

func findWorkspaceAttachment(
	t *testing.T,
	workspaceView workspace.WorkspaceView,
	sourceID basespec.SourceID,
) workspace.WorkspaceAttachmentView {
	t.Helper()
	for _, value := range workspaceView.Attachments {
		if value.SourceID == sourceID {
			return value
		}
	}
	t.Fatalf("attachment for source %q not found in %#v", sourceID, workspaceView.Attachments)
	return workspace.WorkspaceAttachmentView{}
}

func concurrentWorkspaceUpdate(t *testing.T, api *workspace.API, workspaceView workspace.WorkspaceView) {
	t.Helper()
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Go(func() {
			<-start
			_, err := api.UpdateWorkspace(t.Context(), &workspace.UpdateWorkspaceRequest{
				Workspace: workspaceView.Workspace,
				Body: &workspace.UpdateWorkspaceRequestBody{
					ExpectedRevision: workspaceView.Revision,
					DisplayName:      workspaceView.DisplayName + " concurrent",
					Description:      workspaceView.Description,
					Enabled:          true,
					Discovery:        workspaceView.Discovery,
				},
			})
			results <- err
		})
	}
	close(start)
	group.Wait()
	close(results)

	successes := 0
	conflicts := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, basespec.ErrConflict):
			conflicts++
		default:
			t.Fatalf("concurrent UpdateWorkspace error=%v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent UpdateWorkspace successes=%d conflicts=%d", successes, conflicts)
	}
}

type workspaceTestFixture struct {
	components *system.Components
	api        *workspace.API
	root       root.Root
}

func newWorkspaceTestFixture(t *testing.T) *workspaceTestFixture {
	t.Helper()
	components, err := system.Open(t.Context(), system.Config{
		BaseDirectory: t.TempDir(),
		Decoders:      workspace.BuiltinDecoders(),
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
