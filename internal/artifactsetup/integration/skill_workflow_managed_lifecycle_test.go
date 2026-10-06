package integration

import (
	"bytes"
	"errors"
	"path"
	"testing"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill"
)

func TestSkillStoreWorkflowManagedPluginAndSkillLifecycle(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	baseline := fixture.ensureUserBaseline(t)

	ctx := t.Context()
	pluginValue, err := fixture.api.CreateSkillPlugin(
		ctx,
		pluginAPI.CreateRequest{
			RootID:      topology.UserRootID(),
			Name:        spec.LogicalName("release-workflow"),
			DisplayName: "Release workflow",
			Description: "Skills used to prepare release notes.",
		},
	)
	requireNoError(t, err)

	if pluginValue.Baseline {
		t.Fatal("new user Skill Plugin is incorrectly a baseline")
	}
	if !pluginValue.Editable {
		t.Fatal("new user Skill Plugin is not editable")
	}
	if !pluginValue.Deletable {
		t.Fatal("new user Skill Plugin is not deletable")
	}
	if len(pluginValue.Members) != 0 {
		t.Fatalf(
			"new user Skill Plugin members=%d, want 0",
			len(pluginValue.Members),
		)
	}

	pluginValue, err = fixture.api.SetSkillPluginEnabled(
		ctx,
		pluginValue.Artifact.Ref(),
		pluginValue.Artifact.Revision,
		false,
	)
	requireNoError(t, err)
	if pluginValue.Artifact.Enabled {
		t.Fatal("Plugin remains enabled after disable")
	}
	if !pluginValue.Editable {
		t.Fatal("disabling Plugin unexpectedly changed editability")
	}

	pluginValue, err = fixture.api.SetSkillPluginEnabled(
		ctx,
		pluginValue.Artifact.Ref(),
		pluginValue.Artifact.Revision,
		true,
	)
	requireNoError(t, err)
	if !pluginValue.Artifact.Enabled {
		t.Fatal("Plugin remains disabled after enable")
	}

	pluginValue, err = fixture.api.UpdateSkillPlugin(
		ctx,
		pluginAPI.UpdateRequest{
			Plugin:           pluginValue.Artifact.Ref(),
			ExpectedRevision: pluginValue.Artifact.Revision,
			DisplayName:      "Release workflow v2",
			Description:      "Updated release authoring Skill workflow.",
		},
	)
	requireNoError(t, err)
	if pluginValue.DisplayName != "Release workflow v2" {
		t.Fatalf(
			"Plugin display name=%q, want updated value",
			pluginValue.DisplayName,
		)
	}
	if pluginValue.Description !=
		"Updated release authoring Skill workflow." {
		t.Fatalf(
			"Plugin description=%q, want updated value",
			pluginValue.Description,
		)
	}

	const (
		skillName         = "release-notes"
		initialChecklist  = "Check dates and customer-facing changes.\n"
		replacedChecklist = "Check dates, migration notes, and known issues.\n"
	)

	initialDocument := workflowSkillMarkdown(
		skillName,
		"Draft release notes from verified changes.",
		"Summarize shipped changes, known issues, and upgrade notes.",
	)
	created, err := fixture.api.CreateManagedSkill(
		ctx,
		skillAPI.ManagedSkillCreateRequest{
			Plugin:                 pluginValue.Artifact.Ref(),
			ExpectedPluginRevision: pluginValue.Artifact.Revision,
			SkillName:              skillName,
			SKILLMD:                initialDocument,
			Files:                  managedSkillFiles(initialDocument, initialChecklist),
			Enabled:                true,
		},
	)
	requireNoError(t, err)

	if !created.MembershipCreated {
		t.Fatal("creating a new managed Skill did not create Plugin membership")
	}
	if created.Artifact.State != artifactModel.StateAvailable {
		t.Fatalf(
			"created Skill state=%q, want %q",
			created.Artifact.State,
			artifactModel.StateAvailable,
		)
	}
	if !created.Artifact.Enabled {
		t.Fatal("created Skill is disabled")
	}
	if created.Plugin.Artifact.Ref() !=
		pluginValue.Artifact.Ref() {
		t.Fatalf(
			"created Skill Plugin ref=%+v, want %+v",
			created.Plugin.Artifact.Ref(),
			pluginValue.Artifact.Ref(),
		)
	}
	if created.Plugin.Artifact.Revision <=
		pluginValue.Artifact.Revision {
		t.Fatalf(
			"Plugin revision after automatic membership=%d, want greater than %d",
			created.Plugin.Artifact.Revision,
			pluginValue.Artifact.Revision,
		)
	}

	gotPlugin, err := fixture.api.GetSkillPlugin(
		ctx,
		created.Plugin.Artifact.Ref(),
	)
	requireNoError(t, err)
	if len(gotPlugin.Members) != 1 {
		t.Fatalf(
			"Plugin members after Skill creation=%d, want 1",
			len(gotPlugin.Members),
		)
	}
	member := gotPlugin.Members[0]
	if member.Type != declaration.TypeSkill {
		t.Fatalf(
			"Plugin member type=%q, want %q",
			member.Type,
			declaration.TypeSkill,
		)
	}
	if string(member.Name) != skillName {
		t.Fatalf(
			"Plugin member name=%q, want %q",
			member.Name,
			skillName,
		)
	}
	if member.Locator == nil {
		t.Fatal("managed Skill Plugin member has no source-local locator")
	}

	listedSkills, err := fixture.api.ListSkills(
		ctx,
		skillAPI.ListSkillsRequest{
			RootID: topology.UserRootID(),
		},
	)
	requireNoError(t, err)
	listedSkill, found := findSkillByName(listedSkills, skillName)
	if !found {
		t.Fatalf("created Skill %q is absent from ListSkills", skillName)
	}
	if listedSkill.Ref != created.Artifact.Ref() {
		t.Fatalf(
			"listed Skill ref=%+v, want %+v",
			listedSkill.Ref,
			created.Artifact.Ref(),
		)
	}

	gotSkill, err := fixture.api.GetSkill(ctx, created.Artifact.Ref())
	requireNoError(t, err)
	if gotSkill.Ref() != created.Artifact.Ref() {
		t.Fatalf(
			"GetSkill ref=%+v, want %+v",
			gotSkill.Ref(),
			created.Artifact.Ref(),
		)
	}

	runtimeDirectory := path.Base(path.Dir(
		string(gotSkill.Binding.Locator),
	))
	if runtimeDirectory != skillName {
		t.Fatalf(
			"managed Skill runtime directory=%q, want Skill name %q",
			runtimeDirectory,
			skillName,
		)
	}

	managedDocument, err := fixture.api.GetManagedSkillDocument(
		ctx,
		gotSkill.Ref(),
	)
	requireNoError(t, err)
	if managedDocument.Artifact.Ref() != gotSkill.Ref() {
		t.Fatalf(
			"managed document Artifact ref=%+v, want %+v",
			managedDocument.Artifact.Ref(),
			gotSkill.Ref(),
		)
	}
	if managedDocument.Document.Name != skillName {
		t.Fatalf(
			"managed document name=%q, want %q",
			managedDocument.Document.Name,
			skillName,
		)
	}

	storedDocument, err := fixture.store.Resources.ReadSourceEntry(
		ctx,
		gotSkill.RootID,
		gotSkill.Binding.SourceID,
		gotSkill.Binding.Locator,
		spec.MaxCandidateBytes,
	)
	requireNoError(t, err)
	if !bytes.Equal(storedDocument.Content, initialDocument) {
		t.Fatal("managed Skill source document differs from the created SKILL.md")
	}

	assetLocator := spec.Locator(path.Join(
		path.Dir(string(gotSkill.Binding.Locator)),
		"references/checklist.md",
	))
	storedChecklist, err := fixture.store.Resources.ReadSourceEntry(
		ctx,
		gotSkill.RootID,
		gotSkill.Binding.SourceID,
		assetLocator,
		spec.MaxCandidateBytes,
	)
	requireNoError(t, err)
	if !bytes.Equal(storedChecklist.Content, []byte(initialChecklist)) {
		t.Fatal("managed Skill resource differs from the created package resource")
	}

	capabilities, err := fixture.api.ResolveSkillPlugin(
		ctx,
		gotPlugin.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !capabilities.Complete {
		t.Fatal("Plugin capability plan is incomplete for its managed Skill")
	}

	foundSkillCapability := false
	for _, occurrence := range capabilities.Occurrences {
		if occurrence.Target == nil ||
			occurrence.Target.Form != composition.TargetFormArtifact ||
			occurrence.Target.Artifact == nil {
			continue
		}
		if *occurrence.Target.Artifact == gotSkill.Ref() {
			foundSkillCapability = true
			break
		}
	}
	if !foundSkillCapability {
		t.Fatal("Plugin capability plan does not contain the created Skill")
	}

	memberships, err := fixture.api.ListSkillPluginMemberships(
		ctx,
		gotSkill.Ref(),
	)
	requireNoError(t, err)
	if len(memberships) != 1 {
		t.Fatalf(
			"Skill membership count=%d, want 1",
			len(memberships),
		)
	}
	if memberships[0].Plugin != gotPlugin.Artifact.Ref() {
		t.Fatalf(
			"membership Plugin=%+v, want %+v",
			memberships[0].Plugin,
			gotPlugin.Artifact.Ref(),
		)
	}
	if !memberships[0].ResolvedToArtifact ||
		memberships[0].ResolvedArtifact == nil ||
		*memberships[0].ResolvedArtifact != gotSkill.Ref() {
		t.Fatalf(
			"membership did not resolve to created Skill: %+v",
			memberships[0],
		)
	}

	disabledSkill, err := fixture.api.SetSkillEnabled(
		ctx,
		gotSkill.Ref(),
		gotSkill.Revision,
		false,
	)
	requireNoError(t, err)
	if disabledSkill.Enabled {
		t.Fatal("Skill remains enabled after disable")
	}

	disabledDocument, err := fixture.api.GetManagedSkillDocument(
		ctx,
		disabledSkill.Ref(),
	)
	requireNoError(t, err)
	if disabledDocument.Document.Name != skillName {
		t.Fatalf(
			"disabled Skill document name=%q, want %q",
			disabledDocument.Document.Name,
			skillName,
		)
	}

	reenabledSkill, err := fixture.api.SetSkillEnabled(
		ctx,
		disabledSkill.Ref(),
		disabledSkill.Revision,
		true,
	)
	requireNoError(t, err)
	if !reenabledSkill.Enabled {
		t.Fatal("Skill remains disabled after enable")
	}

	replacementDocument := workflowSkillMarkdown(
		skillName,
		"Draft release notes with migration guidance.",
		"Summarize shipped changes, migration steps, and known issues.",
	)
	replaced, err := fixture.api.ReplaceManagedSkill(
		ctx,
		skillAPI.ManagedSkillReplaceRequest{
			Plugin:                   gotPlugin.Artifact.Ref(),
			ExpectedPluginRevision:   gotPlugin.Artifact.Revision,
			Artifact:                 reenabledSkill.Ref(),
			ExpectedArtifactRevision: reenabledSkill.Revision,
			SkillName:                skillName,
			SKILLMD:                  replacementDocument,
			Files: managedSkillFiles(
				replacementDocument,
				replacedChecklist,
			),
			Enabled: true,
		},
	)
	requireNoError(t, err)

	if replaced.Artifact.Ref() != reenabledSkill.Ref() {
		t.Fatalf(
			"replaced Skill ref=%+v, want stable ref %+v",
			replaced.Artifact.Ref(),
			reenabledSkill.Ref(),
		)
	}
	if replaced.Artifact.Revision <= reenabledSkill.Revision {
		t.Fatalf(
			"replaced Skill revision=%d, want greater than %d",
			replaced.Artifact.Revision,
			reenabledSkill.Revision,
		)
	}

	updatedSkill, err := fixture.api.GetSkill(ctx, replaced.Artifact.Ref())
	requireNoError(t, err)

	updatedDocument, err := fixture.api.GetManagedSkillDocument(
		ctx,
		updatedSkill.Ref(),
	)
	requireNoError(t, err)
	if updatedDocument.Document.Name != skillName {
		t.Fatalf(
			"updated Skill document name=%q, want %q",
			updatedDocument.Document.Name,
			skillName,
		)
	}

	updatedSourceDocument, err := fixture.store.Resources.ReadSourceEntry(
		ctx,
		updatedSkill.RootID,
		updatedSkill.Binding.SourceID,
		updatedSkill.Binding.Locator,
		spec.MaxCandidateBytes,
	)
	requireNoError(t, err)
	if !bytes.Equal(updatedSourceDocument.Content, replacementDocument) {
		t.Fatal("managed Skill replacement did not replace SKILL.md")
	}

	updatedChecklist, err := fixture.store.Resources.ReadSourceEntry(
		ctx,
		updatedSkill.RootID,
		updatedSkill.Binding.SourceID,
		assetLocator,
		spec.MaxCandidateBytes,
	)
	requireNoError(t, err)
	if !bytes.Equal(updatedChecklist.Content, []byte(replacedChecklist)) {
		t.Fatal("managed Skill replacement did not replace packaged resource")
	}

	pluginBeforeDetach, err := fixture.api.GetSkillPlugin(
		ctx,
		gotPlugin.Artifact.Ref(),
	)
	requireNoError(t, err)

	detachedPlugin, err := fixture.api.RemoveSkillPluginMember(
		ctx,
		pluginAPI.RemoveMemberRequest{
			Plugin:           pluginBeforeDetach.Artifact.Ref(),
			ExpectedRevision: pluginBeforeDetach.Artifact.Revision,
			Index:            0,
		},
	)
	requireNoError(t, err)
	if len(detachedPlugin.Members) != 0 {
		t.Fatalf(
			"Plugin members after detach=%d, want 0",
			len(detachedPlugin.Members),
		)
	}

	memberships, err = fixture.api.ListSkillPluginMemberships(
		ctx,
		updatedSkill.Ref(),
	)
	requireNoError(t, err)
	if len(memberships) != 0 {
		t.Fatalf(
			"Skill memberships after detach=%d, want 0",
			len(memberships),
		)
	}

	skillBeforePurge, err := fixture.api.GetSkill(ctx, updatedSkill.Ref())
	requireNoError(t, err)

	err = fixture.api.PurgeSkill(
		ctx,
		skillBeforePurge.Ref(),
		skillBeforePurge.Revision,
	)
	requireNoError(t, err)

	_, err = fixture.api.GetSkill(ctx, skillBeforePurge.Ref())
	if !errors.Is(err, spec.ErrArtifactNotFound) {
		t.Fatalf(
			"GetSkill after purge error=%v, want ErrArtifactNotFound",
			err,
		)
	}

	remainingSkills, err := fixture.api.ListSkills(
		ctx,
		skillAPI.ListSkillsRequest{
			RootID: topology.UserRootID(),
		},
	)
	requireNoError(t, err)
	if _, found := findSkillByName(remainingSkills, skillName); found {
		t.Fatalf("purged Skill %q remains in ListSkills", skillName)
	}

	pluginBeforeDelete, err := fixture.api.GetSkillPlugin(
		ctx,
		detachedPlugin.Artifact.Ref(),
	)
	requireNoError(t, err)

	err = fixture.api.DeleteSkillPlugin(
		ctx,
		pluginAPI.DeleteRequest{
			Plugin:           pluginBeforeDelete.Artifact.Ref(),
			ExpectedRevision: pluginBeforeDelete.Artifact.Revision,
		},
	)
	requireNoError(t, err)

	_, err = fixture.api.GetSkillPlugin(
		ctx,
		pluginBeforeDelete.Artifact.Ref(),
	)
	if !errors.Is(err, spec.ErrArtifactNotFound) {
		t.Fatalf(
			"GetSkillPlugin after delete error=%v, want ErrArtifactNotFound",
			err,
		)
	}

	remainingPlugins, err := fixture.api.ListSkillPlugins(
		ctx,
		topology.UserRootID(),
	)
	requireNoError(t, err)
	if len(remainingPlugins) != 1 {
		t.Fatalf(
			"user Plugins after cleanup=%d, want only baseline",
			len(remainingPlugins),
		)
	}

	remainingBaseline, found := findPluginByName(
		remainingPlugins,
		string(pluginAPI.SkillBaselinePluginName),
	)
	if !found {
		t.Fatal("baseline Plugin disappeared during managed Skill cleanup")
	}
	if remainingBaseline.Ref != baseline.Artifact.Ref() {
		t.Fatalf(
			"remaining baseline ref=%+v, want %+v",
			remainingBaseline.Ref,
			baseline.Artifact.Ref(),
		)
	}
}
