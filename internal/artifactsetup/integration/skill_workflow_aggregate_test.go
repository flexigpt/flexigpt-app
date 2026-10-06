package integration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	inferencewrapperSpec "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inferencewrapper/spec"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/skillruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill"
)

func TestSkillStoreWorkflowRuntimeAdapterFollowsSkillLifecycle(t *testing.T) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	ctx := t.Context()
	pluginValue, err := fixture.api.CreateSkillPlugin(
		ctx,
		pluginAPI.CreateRequest{
			RootID:      topology.UserRootID(),
			Name:        "catalog-workflow",
			DisplayName: "Catalog workflow",
			Description: "Skill Plugin used to verify runtime catalog sync.",
		},
	)
	requireNoError(t, err)

	const skillName = "catalog-notes"
	created, _ := createManagedSkillInPlugin(
		t,
		fixture.api,
		pluginValue,
		skillName,
		"Prepare catalog-aware release notes.",
		"Use the current verified Skill package.",
		"Check the release catalog.\n",
	)

	catalogSkill, err := fixture.api.GetSkill(ctx, created.Artifact.Ref())
	requireNoError(t, err)
	if !catalogSkill.Enabled {
		t.Fatal("managed Skill is disabled before initial synchronization")
	}

	inspection, err := fixture.store.Refresh.InspectSource(
		ctx,
		catalogSkill.RootID,
		catalogSkill.Binding.SourceID,
	)
	requireNoError(t, err)
	if !inspection.IsCurrent() {
		t.Fatalf("managed Skill Source is stale: %+v", inspection)
	}

	adapter, runtimeService := newSkillRuntimeAdapter(t, fixture)
	requireNoError(t, adapter.SyncRootCatalog(ctx, topology.UserRootID()))

	initial, err := resolveWorkflowSkill(ctx, adapter, created.Artifact.Ref())
	requireNoError(t, err)
	if !initial.Enabled || initial.Artifact != created.Artifact.Ref() {
		t.Fatalf("unexpected initial resolution: %+v", initial)
	}
	if initial.Definition.Name != skillName {
		t.Fatalf("Skill name=%q, want %q", initial.Definition.Name, skillName)
	}

	sessionID := newWorkflowSkillSession(t, runtimeService)
	request := inferencewrapperSpec.SkillSessionRequest{
		SessionID: sessionID,
		Artifacts: []artifactModel.ArtifactRef{created.Artifact.Ref()},
	}
	session, err := adapter.ResolveSkillSession(ctx, request)
	requireNoError(t, err)
	if !artifactRefSetEquals(session.AvailableArtifacts, request.Artifacts) {
		t.Fatalf("available refs=%+v, want %+v", session.AvailableArtifacts, request.Artifacts)
	}
	if !strings.Contains(session.Prompt, skillName) {
		t.Fatalf("prompt does not mention Skill %q: %q", skillName, session.Prompt)
	}
	if session.RulesPrompt == "" {
		t.Fatal("Skill rules prompt is empty")
	}
	if len(session.ToolChoices) != 1 || session.ToolChoices[0].Name != "skills-load" {
		t.Fatalf("inactive session tools=%+v, want only skills-load", session.ToolChoices)
	}

	_, err = adapter.ResolveSkillSession(ctx, inferencewrapperSpec.SkillSessionRequest{
		SessionID: sessionID,
	})
	if !errors.Is(err, skillruntime.ErrArtifactSkillSelectionRequired) {
		t.Fatalf("empty selection error=%v, want selection required", err)
	}

	disabled, err := fixture.api.SetSkillEnabled(
		ctx,
		created.Artifact.Ref(),
		created.Artifact.Revision,
		false,
	)
	requireNoError(t, err)
	if disabled.Enabled {
		t.Fatal("Skill remains enabled after disable")
	}

	_, err = resolveWorkflowSkill(ctx, adapter, disabled.Ref())
	if !errors.Is(err, spec.ErrReferenceUnresolved) {
		t.Fatalf("resolution after disable error=%v, want ErrReferenceUnresolved", err)
	}
	if runtimeService.IsRegistered(skillRuntime.SkillRegistration{
		Definition: initial.Definition,
		Revision:   initial.Version,
	}) {
		t.Fatal("disabled Skill remains registered")
	}
	runtimeRecords, err := runtimeService.ListAgentSkills(ctx, nil)
	requireNoError(t, err)
	for _, record := range runtimeRecords {
		if record.Def == initial.Definition {
			t.Fatalf("disabled Skill remains indexed: %+v", record.Def)
		}
	}

	reenabled, err := fixture.api.SetSkillEnabled(
		ctx,
		disabled.Ref(),
		disabled.Revision,
		true,
	)
	requireNoError(t, err)
	if !reenabled.Enabled {
		t.Fatal("Skill remains disabled after enable")
	}
	active, err := resolveWorkflowSkill(ctx, adapter, reenabled.Ref())
	requireNoError(t, err)
	if !active.Enabled {
		t.Fatal("re-enabled Skill resolved as disabled")
	}

	currentPlugin, err := fixture.api.GetSkillPlugin(
		ctx,
		created.Plugin.Artifact.Ref(),
	)
	requireNoError(t, err)
	currentSkill, err := fixture.api.GetSkill(ctx, reenabled.Ref())
	requireNoError(t, err)

	replacementDocument := workflowSkillMarkdown(
		skillName,
		"Prepare catalog-aware release notes with migration details.",
		"Use the replacement Skill package after catalog reconciliation.",
	)
	replaced, err := fixture.api.ReplaceManagedSkill(
		ctx,
		skillAPI.ManagedSkillReplaceRequest{
			Plugin:                   currentPlugin.Artifact.Ref(),
			ExpectedPluginRevision:   currentPlugin.Artifact.Revision,
			Artifact:                 currentSkill.Ref(),
			ExpectedArtifactRevision: currentSkill.Revision,
			SkillName:                skillName,
			SKILLMD:                  replacementDocument,
			Files: managedSkillFiles(
				replacementDocument,
				"Check catalog, migrations, and compatibility.\n",
			),
			Enabled: true,
		},
	)
	requireNoError(t, err)
	if replaced.Artifact.Ref() != created.Artifact.Ref() {
		t.Fatalf("replacement changed Artifact identity: %+v", replaced.Artifact.Ref())
	}

	updated, err := resolveWorkflowSkill(ctx, adapter, replaced.Artifact.Ref())
	requireNoError(t, err)
	if updated.Version == active.Version {
		t.Fatalf("runtime revision did not change after replacement: %q", updated.Version)
	}
	if !updated.Enabled {
		t.Fatal("replaced Skill resolved as disabled")
	}
	if !runtimeService.IsRegistered(skillRuntime.SkillRegistration{
		Definition: updated.Definition,
		Revision:   updated.Version,
	}) {
		t.Fatal("replacement revision was not registered")
	}

	session, err = adapter.ResolveSkillSession(ctx, request)
	requireNoError(t, err)
	if !artifactRefSetEquals(session.AvailableArtifacts, request.Artifacts) {
		t.Fatalf("refs after replacement=%+v, want %+v", session.AvailableArtifacts, request.Artifacts)
	}
	if !strings.Contains(session.Prompt, skillName) {
		t.Fatalf("replacement prompt does not mention Skill %q", skillName)
	}
}

func newSkillRuntimeAdapter(
	t *testing.T,
	fixture *skillWorkflowFixture,
) (*skillruntime.RuntimeAdapter, *skillRuntime.Service) {
	t.Helper()

	adapter, err := skillruntime.NewRuntimeAdapter(
		fixture.store.Artifacts,
		fixture.store.Catalog,
		fixture.store.Resources,
		fixture.store.TrustedNativeResources,
	)
	requireNoError(t, err)

	runtimeService, err := skillRuntime.New(
		skillRuntime.WithCatalogSource(adapter),
	)
	requireNoError(t, err)

	t.Cleanup(func() {
		if err := runtimeService.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Errorf("close Skill runtime: %v", err)
		}
	})
	requireNoError(t, adapter.BindRuntime(runtimeService))
	return adapter, runtimeService
}

func newWorkflowSkillSession(
	t *testing.T,
	runtimeService *skillRuntime.Service,
) agentskillsRuntimeSpec.SessionID {
	t.Helper()

	sessionID, _, err := runtimeService.NewSession(t.Context())
	requireNoError(t, err)
	t.Cleanup(func() {
		if err := runtimeService.CloseSession(
			context.WithoutCancel(t.Context()),
			sessionID,
		); err != nil {
			t.Errorf("close Skill session: %v", err)
		}
	})
	return sessionID
}

func resolveWorkflowSkill(
	ctx context.Context,
	adapter *skillruntime.RuntimeAdapter,
	ref artifactModel.ArtifactRef,
) (skillruntime.ResolvedArtifactSkill, error) {
	selected, err := adapter.ResolveSkills(ctx, []artifactModel.ArtifactRef{ref})
	if err != nil {
		return skillruntime.ResolvedArtifactSkill{}, err
	}
	if len(selected.Values) != 1 {
		return skillruntime.ResolvedArtifactSkill{}, fmt.Errorf(
			"expected one resolved Skill, got %d",
			len(selected.Values),
		)
	}
	return selected.Values[0], nil
}
