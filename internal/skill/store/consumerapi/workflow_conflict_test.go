package consumerapi_test

import (
	"bytes"
	"errors"
	"testing"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
)

func TestSkillStoreWorkflowRejectsStaleCollectionAndSkillMutations(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	ctx := t.Context()

	collectionValue, err := fixture.api.CreateSkillCollection(
		ctx,
		collection.CreateRequest{
			RootID:      documentTopology.UserRootID(),
			Name:        "concurrency-workflow",
			DisplayName: "Concurrency workflow",
			Description: "Initial Collection description.",
		},
	)
	requireNoError(t, err)

	staleCollection := collectionValue

	winnerCollection, err := fixture.api.UpdateSkillCollection(
		ctx,
		collection.UpdateRequest{
			Collection:       collectionValue.Artifact.Ref(),
			ExpectedRevision: collectionValue.Artifact.Revision,
			DisplayName:      "Concurrency workflow winner",
			Description:      "Winner Collection description.",
		},
	)
	requireNoError(t, err)

	_, err = fixture.api.UpdateSkillCollection(
		ctx,
		collection.UpdateRequest{
			Collection:       staleCollection.Artifact.Ref(),
			ExpectedRevision: staleCollection.Artifact.Revision,
			DisplayName:      "Concurrency workflow stale writer",
			Description:      "Stale writer must not win.",
		},
	)
	if !errors.Is(err, basespec.ErrConflict) {
		t.Fatalf(
			"stale Collection update error=%v, want ErrConflict",
			err,
		)
	}

	verifiedCollection, err := fixture.api.GetSkillCollection(
		ctx,
		winnerCollection.Artifact.Ref(),
	)
	requireNoError(t, err)
	if verifiedCollection.DisplayName !=
		"Concurrency workflow winner" {
		t.Fatalf(
			"Collection display name after stale update=%q, want winner value",
			verifiedCollection.DisplayName,
		)
	}
	if verifiedCollection.Description !=
		"Winner Collection description." {
		t.Fatalf(
			"Collection description after stale update=%q, want winner value",
			verifiedCollection.Description,
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
			Collection:                 winnerCollection.Artifact.Ref(),
			ExpectedCollectionRevision: winnerCollection.Artifact.Revision,
			SkillName:                  skillName,
			SKILLMD:                    initialDocument,
			Files: managedSkillFiles(
				initialDocument,
				"Check concurrent edits before publishing.\n",
			),
			Enabled: true,
		},
	)
	requireNoError(t, err)

	staleCollectionForCreate := winnerCollection
	staleDocument := workflowSkillMarkdown(
		"stale-collection-skill",
		"This Skill must not be created.",
		"Do not publish from a stale Collection revision.",
	)
	_, err = fixture.api.CreateManagedSkill(
		ctx,
		skillConsumerAPI.ManagedSkillCreateRequest{
			Collection: staleCollectionForCreate.Artifact.Ref(),
			ExpectedCollectionRevision: staleCollectionForCreate.
				Artifact.Revision,
			SkillName: "stale-collection-skill",
			SKILLMD:   staleDocument,
			Files: managedSkillFiles(
				staleDocument,
				"Stale Collection write.\n",
			),
			Enabled: true,
		},
	)
	if !errors.Is(err, basespec.ErrConflict) {
		t.Fatalf(
			"stale Collection Skill create error=%v, want ErrConflict",
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
	if !errors.Is(err, basespec.ErrConflict) {
		t.Fatalf(
			"stale Skill enable error=%v, want ErrConflict",
			err,
		)
	}

	currentCollection, err := fixture.api.GetSkillCollection(
		ctx,
		created.Collection.Artifact.Ref(),
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
			Collection:                 currentCollection.Artifact.Ref(),
			ExpectedCollectionRevision: currentCollection.Artifact.Revision,
			Artifact:                   staleSkill.Ref(),
			ExpectedArtifactRevision:   staleSkill.Revision,
			SkillName:                  skillName,
			SKILLMD:                    replacementDocument,
			Files: managedSkillFiles(
				replacementDocument,
				"Stale replacement content.\n",
			),
			Enabled: true,
		},
	)
	if !errors.Is(err, basespec.ErrConflict) {
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
		basespec.MaxCandidateBytes,
	)
	requireNoError(t, err)
	if !bytes.Equal(storedDocument.Content, initialDocument) {
		t.Fatal("stale Skill replacement changed SKILL.md")
	}

	userSkills, err := fixture.api.ListSkills(
		ctx,
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	if _, found := findSkillByName(
		userSkills,
		"stale-collection-skill",
	); found {
		t.Fatal("stale Collection create published a new Skill")
	}
}

func TestSkillStoreWorkflowManagedCreateReplayIsIdempotentAndDoesNotReplace(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	ctx := t.Context()

	collectionValue, err := fixture.api.CreateSkillCollection(
		ctx,
		collection.CreateRequest{
			RootID:      documentTopology.UserRootID(),
			Name:        "replay-workflow",
			DisplayName: "Replay workflow",
			Description: "Collection used to verify managed create replay.",
		},
	)
	requireNoError(t, err)

	const skillName = "replay-notes"

	initial, initialDocument := createManagedSkillInCollection(
		t,
		fixture.api,
		collectionValue,
		skillName,
		"Create replay-safe release notes.",
		"Use the original package unless an explicit replace succeeds.",
		"Check replay identity.\n",
	)

	replayed, err := fixture.api.CreateManagedSkill(
		ctx,
		skillConsumerAPI.ManagedSkillCreateRequest{
			Collection:                 initial.Collection.Artifact.Ref(),
			ExpectedCollectionRevision: initial.Collection.Artifact.Revision,
			SkillName:                  skillName,
			SKILLMD:                    initialDocument,
			Files: managedSkillFiles(
				initialDocument,
				"Check replay identity.\n",
			),
			Enabled: true,
		},
	)
	requireNoError(t, err)

	if replayed.MembershipCreated {
		t.Fatal("idempotent managed Skill create recreated Collection membership")
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
			Collection:                 replayed.Collection.Artifact.Ref(),
			ExpectedCollectionRevision: replayed.Collection.Artifact.Revision,
			SkillName:                  skillName,
			SKILLMD:                    replacementDocument,
			Files: managedSkillFiles(
				replacementDocument,
				"Implicit replacement.\n",
			),
			Enabled: true,
		},
	)
	if !errors.Is(err, basespec.ErrConflict) {
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
		basespec.MaxCandidateBytes,
	)
	requireNoError(t, err)
	if !bytes.Equal(storedDocument.Content, initialDocument) {
		t.Fatal("implicit managed Skill replacement changed SKILL.md")
	}
}
