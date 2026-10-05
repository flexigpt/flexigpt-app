package consumerapi_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/consumerapi"
)

func TestSkillStoreWorkflowRejectsStalePluginAndSkillMutations(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	ctx := t.Context()

	pluginValue, err := fixture.api.CreateSkillPlugin(
		ctx,
		plugin.CreateRequest{
			RootID:      topology.UserRootID(),
			Name:        "concurrency-workflow",
			DisplayName: "Concurrency workflow",
			Description: "Initial Plugin description.",
		},
	)
	requireNoError(t, err)

	stalePlugin := pluginValue

	winnerPlugin, err := fixture.api.UpdateSkillPlugin(
		ctx,
		plugin.UpdateRequest{
			Plugin:           pluginValue.Artifact.Ref(),
			ExpectedRevision: pluginValue.Artifact.Revision,
			DisplayName:      "Concurrency workflow winner",
			Description:      "Winner Plugin description.",
		},
	)
	requireNoError(t, err)

	_, err = fixture.api.UpdateSkillPlugin(
		ctx,
		plugin.UpdateRequest{
			Plugin:           stalePlugin.Artifact.Ref(),
			ExpectedRevision: stalePlugin.Artifact.Revision,
			DisplayName:      "Concurrency workflow stale writer",
			Description:      "Stale writer must not win.",
		},
	)
	if !errors.Is(err, spec.ErrConflict) {
		t.Fatalf(
			"stale Plugin update error=%v, want ErrConflict",
			err,
		)
	}

	verifiedPlugin, err := fixture.api.GetSkillPlugin(
		ctx,
		winnerPlugin.Artifact.Ref(),
	)
	requireNoError(t, err)
	if verifiedPlugin.DisplayName !=
		"Concurrency workflow winner" {
		t.Fatalf(
			"Plugin display name after stale update=%q, want winner value",
			verifiedPlugin.DisplayName,
		)
	}
	if verifiedPlugin.Description !=
		"Winner Plugin description." {
		t.Fatalf(
			"Plugin description after stale update=%q, want winner value",
			verifiedPlugin.Description,
		)
	}

	const skillName = "concurrency-notes"

	initialDocument := workflowSkillMarkdown(
		skillName,
		"Create release notes with optimistic concurrency.",
		"Use the currently committed Skill package only.",
	)
	created, err := fixture.api.CreateManagedSkill(
		ctx,
		skillConsumerAPI.ManagedSkillCreateRequest{
			Plugin:                 winnerPlugin.Artifact.Ref(),
			ExpectedPluginRevision: winnerPlugin.Artifact.Revision,
			SkillName:              skillName,
			SKILLMD:                initialDocument,
			Files: managedSkillFiles(
				initialDocument,
				"Check concurrent edits before publishing.\n",
			),
			Enabled: true,
		},
	)
	requireNoError(t, err)

	stalePluginForCreate := winnerPlugin
	staleDocument := workflowSkillMarkdown(
		"stale-plugin-skill",
		"This Skill must not be created.",
		"Do not publish from a stale Plugin revision.",
	)
	_, err = fixture.api.CreateManagedSkill(
		ctx,
		skillConsumerAPI.ManagedSkillCreateRequest{
			Plugin: stalePluginForCreate.Artifact.Ref(),
			ExpectedPluginRevision: stalePluginForCreate.
				Artifact.Revision,
			SkillName: "stale-plugin-skill",
			SKILLMD:   staleDocument,
			Files: managedSkillFiles(
				staleDocument,
				"Stale Plugin write.\n",
			),
			Enabled: true,
		},
	)
	if !errors.Is(err, spec.ErrConflict) {
		t.Fatalf(
			"stale Plugin Skill create error=%v, want ErrConflict",
			err,
		)
	}

	staleSkill := created.Artifact

	disabledSkill, err := fixture.api.SetSkillEnabled(
		ctx,
		staleSkill.Ref(),
		staleSkill.Revision,
		false,
	)
	requireNoError(t, err)
	if disabledSkill.Enabled {
		t.Fatal("winner Skill update did not disable the Skill")
	}

	_, err = fixture.api.SetSkillEnabled(
		ctx,
		staleSkill.Ref(),
		staleSkill.Revision,
		true,
	)
	if !errors.Is(err, spec.ErrConflict) {
		t.Fatalf(
			"stale Skill enable error=%v, want ErrConflict",
			err,
		)
	}

	currentPlugin, err := fixture.api.GetSkillPlugin(
		ctx,
		created.Plugin.Artifact.Ref(),
	)
	requireNoError(t, err)

	replacementDocument := workflowSkillMarkdown(
		skillName,
		"This replacement must not be published.",
		"Do not replace from a stale Skill revision.",
	)
	_, err = fixture.api.ReplaceManagedSkill(
		ctx,
		skillConsumerAPI.ManagedSkillReplaceRequest{
			Plugin:                   currentPlugin.Artifact.Ref(),
			ExpectedPluginRevision:   currentPlugin.Artifact.Revision,
			Artifact:                 staleSkill.Ref(),
			ExpectedArtifactRevision: staleSkill.Revision,
			SkillName:                skillName,
			SKILLMD:                  replacementDocument,
			Files: managedSkillFiles(
				replacementDocument,
				"Stale replacement content.\n",
			),
			Enabled: true,
		},
	)
	if !errors.Is(err, spec.ErrConflict) {
		t.Fatalf(
			"stale Skill replacement error=%v, want ErrConflict",
			err,
		)
	}

	currentSkill, err := fixture.api.GetSkill(ctx, staleSkill.Ref())
	requireNoError(t, err)
	if currentSkill.Enabled {
		t.Fatal("stale Skill enable overwrote the winner's disabled state")
	}

	storedDocument, err := fixture.store.Resources.ReadSourceEntry(
		ctx,
		currentSkill.RootID,
		currentSkill.Binding.SourceID,
		currentSkill.Binding.Locator,
		spec.MaxCandidateBytes,
	)
	requireNoError(t, err)
	if !bytes.Equal(storedDocument.Content, initialDocument) {
		t.Fatal("stale Skill replacement changed SKILL.md")
	}

	userSkills, err := fixture.api.ListSkills(
		ctx,
		skillConsumerAPI.ListSkillsRequest{
			RootID: topology.UserRootID(),
		},
	)
	requireNoError(t, err)
	if _, found := findSkillByName(
		userSkills,
		"stale-plugin-skill",
	); found {
		t.Fatal("stale Plugin create published a new Skill")
	}
}

func TestSkillStoreWorkflowManagedCreateReplayIsIdempotentAndDoesNotReplace(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	ctx := t.Context()

	pluginValue, err := fixture.api.CreateSkillPlugin(
		ctx,
		plugin.CreateRequest{
			RootID:      topology.UserRootID(),
			Name:        "replay-workflow",
			DisplayName: "Replay workflow",
			Description: "Plugin used to verify managed create replay.",
		},
	)
	requireNoError(t, err)

	const skillName = "replay-notes"

	initial, initialDocument := createManagedSkillInPlugin(
		t,
		fixture.api,
		pluginValue,
		skillName,
		"Create replay-safe release notes.",
		"Use the original package unless an explicit replace succeeds.",
		"Check replay identity.\n",
	)

	replayed, err := fixture.api.CreateManagedSkill(
		ctx,
		skillConsumerAPI.ManagedSkillCreateRequest{
			Plugin:                 initial.Plugin.Artifact.Ref(),
			ExpectedPluginRevision: initial.Plugin.Artifact.Revision,
			SkillName:              skillName,
			SKILLMD:                initialDocument,
			Files: managedSkillFiles(
				initialDocument,
				"Check replay identity.\n",
			),
			Enabled: true,
		},
	)
	requireNoError(t, err)

	if replayed.MembershipCreated {
		t.Fatal("idempotent managed Skill create recreated Plugin membership")
	}
	if replayed.Artifact.Ref() != initial.Artifact.Ref() {
		t.Fatalf(
			"replayed Skill ref=%+v, want %+v",
			replayed.Artifact.Ref(),
			initial.Artifact.Ref(),
		)
	}
	if replayed.Artifact.Revision != initial.Artifact.Revision {
		t.Fatalf(
			"replayed Skill revision=%d, want %d",
			replayed.Artifact.Revision,
			initial.Artifact.Revision,
		)
	}

	replacementDocument := workflowSkillMarkdown(
		skillName,
		"This implicit replacement must fail.",
		"CreateManagedSkill must not replace an existing package.",
	)
	_, err = fixture.api.CreateManagedSkill(
		ctx,
		skillConsumerAPI.ManagedSkillCreateRequest{
			Plugin:                 replayed.Plugin.Artifact.Ref(),
			ExpectedPluginRevision: replayed.Plugin.Artifact.Revision,
			SkillName:              skillName,
			SKILLMD:                replacementDocument,
			Files: managedSkillFiles(
				replacementDocument,
				"Implicit replacement.\n",
			),
			Enabled: true,
		},
	)
	if !errors.Is(err, spec.ErrConflict) {
		t.Fatalf(
			"implicit managed Skill replacement error=%v, want ErrConflict",
			err,
		)
	}

	currentSkill, err := fixture.api.GetSkill(
		ctx,
		initial.Artifact.Ref(),
	)
	requireNoError(t, err)
	if currentSkill.Revision != initial.Artifact.Revision {
		t.Fatalf(
			"implicit replacement changed Skill revision=%d, want %d",
			currentSkill.Revision,
			initial.Artifact.Revision,
		)
	}

	storedDocument, err := fixture.store.Resources.ReadSourceEntry(
		ctx,
		currentSkill.RootID,
		currentSkill.Binding.SourceID,
		currentSkill.Binding.Locator,
		spec.MaxCandidateBytes,
	)
	requireNoError(t, err)
	if !bytes.Equal(storedDocument.Content, initialDocument) {
		t.Fatal("implicit managed Skill replacement changed SKILL.md")
	}
}
