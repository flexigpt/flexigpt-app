package consumerapi_test

import (
	"bytes"
	"errors"
	"path"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
)

func TestSkillStoreWorkflowManagedCollectionAndSkillLifecycle(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	baseline := fixture.ensureUserBaseline(t)

	ctx := t.Context()
	collectionValue, err := fixture.api.CreateSkillCollection(
		ctx,
		collection.CreateRequest{
			RootID:      documentTopology.UserRootID(),
			Name:        basespec.LogicalName("release-workflow"),
			DisplayName: "Release workflow",
			Description: "Skills used to prepare release notes.",
		},
	)
	requireNoError(t, err)

	if collectionValue.Baseline {
		t.Fatal("new user Skill Collection is incorrectly a baseline")
	}
	if !collectionValue.Editable {
		t.Fatal("new user Skill Collection is not editable")
	}
	if !collectionValue.Deletable {
		t.Fatal("new user Skill Collection is not deletable")
	}
	if len(collectionValue.Members) != 0 {
		t.Fatalf(
			"new user Skill Collection members=%d, want 0",
			len(collectionValue.Members),
		)
	}

	collectionValue, err = fixture.api.SetSkillCollectionEnabled(
		ctx,
		collectionValue.Artifact.Ref(),
		collectionValue.Artifact.Revision,
		false,
	)
	requireNoError(t, err)
	if collectionValue.Artifact.Enabled {
		t.Fatal("Collection remains enabled after disable")
	}
	if !collectionValue.Editable {
		t.Fatal("disabling Collection unexpectedly changed editability")
	}

	collectionValue, err = fixture.api.SetSkillCollectionEnabled(
		ctx,
		collectionValue.Artifact.Ref(),
		collectionValue.Artifact.Revision,
		true,
	)
	requireNoError(t, err)
	if !collectionValue.Artifact.Enabled {
		t.Fatal("Collection remains disabled after enable")
	}

	collectionValue, err = fixture.api.UpdateSkillCollection(
		ctx,
		collection.UpdateRequest{
			Collection:       collectionValue.Artifact.Ref(),
			ExpectedRevision: collectionValue.Artifact.Revision,
			DisplayName:      "Release workflow v2",
			Description:      "Updated release authoring Skill workflow.",
		},
	)
	requireNoError(t, err)
	if collectionValue.DisplayName != "Release workflow v2" {
		t.Fatalf(
			"Collection display name=%q, want updated value",
			collectionValue.DisplayName,
		)
	}
	if collectionValue.Description !=
		"Updated release authoring Skill workflow." {
		t.Fatalf(
			"Collection description=%q, want updated value",
			collectionValue.Description,
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
		skillConsumerAPI.ManagedSkillCreateRequest{
			Collection:                 collectionValue.Artifact.Ref(),
			ExpectedCollectionRevision: collectionValue.Artifact.Revision,
			SkillName:                  skillName,
			SKILLMD:                    initialDocument,
			Files:                      managedSkillFiles(initialDocument, initialChecklist),
			Enabled:                    true,
		},
	)
	requireNoError(t, err)

	if !created.MembershipCreated {
		t.Fatal("creating a new managed Skill did not create Collection membership")
	}
	if created.Artifact.State != artifact.StateAvailable {
		t.Fatalf(
			"created Skill state=%q, want %q",
			created.Artifact.State,
			artifact.StateAvailable,
		)
	}
	if !created.Artifact.Enabled {
		t.Fatal("created Skill is disabled")
	}
	if created.Collection.Artifact.Ref() !=
		collectionValue.Artifact.Ref() {
		t.Fatalf(
			"created Skill Collection ref=%+v, want %+v",
			created.Collection.Artifact.Ref(),
			collectionValue.Artifact.Ref(),
		)
	}
	if created.Collection.Artifact.Revision <=
		collectionValue.Artifact.Revision {
		t.Fatalf(
			"Collection revision after automatic membership=%d, want greater than %d",
			created.Collection.Artifact.Revision,
			collectionValue.Artifact.Revision,
		)
	}

	gotCollection, err := fixture.api.GetSkillCollection(
		ctx,
		created.Collection.Artifact.Ref(),
	)
	requireNoError(t, err)
	if len(gotCollection.Members) != 1 {
		t.Fatalf(
			"Collection members after Skill creation=%d, want 1",
			len(gotCollection.Members),
		)
	}
	member := gotCollection.Members[0]
	if member.Type != declaration.TypeSkill {
		t.Fatalf(
			"Collection member type=%q, want %q",
			member.Type,
			declaration.TypeSkill,
		)
	}
	if string(member.Name) != skillName {
		t.Fatalf(
			"Collection member name=%q, want %q",
			member.Name,
			skillName,
		)
	}
	if member.Locator == nil {
		t.Fatal("managed Skill Collection member has no source-local locator")
	}

	listedSkills, err := fixture.api.ListSkills(
		ctx,
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	listedSkill, found := findSkillByName(listedSkills, skillName)
	if !found {
		t.Fatalf("created Skill %q is absent from ListSkills", skillName)
	}
	if listedSkill.Ref() != created.Artifact.Ref() {
		t.Fatalf(
			"listed Skill ref=%+v, want %+v",
			listedSkill.Ref(),
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
		basespec.MaxCandidateBytes,
	)
	requireNoError(t, err)
	if !bytes.Equal(storedDocument.Content, initialDocument) {
		t.Fatal("managed Skill source document differs from the created SKILL.md")
	}

	assetLocator := basespec.Locator(path.Join(
		path.Dir(string(gotSkill.Binding.Locator)),
		"references/checklist.md",
	))
	storedChecklist, err := fixture.store.Resources.ReadSourceEntry(
		ctx,
		gotSkill.RootID,
		gotSkill.Binding.SourceID,
		assetLocator,
		basespec.MaxCandidateBytes,
	)
	requireNoError(t, err)
	if !bytes.Equal(storedChecklist.Content, []byte(initialChecklist)) {
		t.Fatal("managed Skill resource differs from the created package resource")
	}

	capabilities, err := fixture.api.ResolveSkillCollection(
		ctx,
		gotCollection.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !capabilities.Complete {
		t.Fatal("Collection capability plan is incomplete for its managed Skill")
	}

	foundSkillCapability := false
	for _, occurrence := range capabilities.Occurrences {
		if occurrence.Artifact == nil {
			continue
		}
		if *occurrence.Artifact == gotSkill.Ref() {
			foundSkillCapability = true
			break
		}
	}
	if !foundSkillCapability {
		t.Fatal("Collection capability plan does not contain the created Skill")
	}

	memberships, err := fixture.api.ListSkillCollectionMemberships(
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
	if memberships[0].Collection != gotCollection.Artifact.Ref() {
		t.Fatalf(
			"membership Collection=%+v, want %+v",
			memberships[0].Collection,
			gotCollection.Artifact.Ref(),
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
		skillConsumerAPI.ManagedSkillReplaceRequest{
			Collection:                 gotCollection.Artifact.Ref(),
			ExpectedCollectionRevision: gotCollection.Artifact.Revision,
			Artifact:                   reenabledSkill.Ref(),
			ExpectedArtifactRevision:   reenabledSkill.Revision,
			SkillName:                  skillName,
			SKILLMD:                    replacementDocument,
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
		basespec.MaxCandidateBytes,
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
		basespec.MaxCandidateBytes,
	)
	requireNoError(t, err)
	if !bytes.Equal(updatedChecklist.Content, []byte(replacedChecklist)) {
		t.Fatal("managed Skill replacement did not replace packaged resource")
	}

	collectionBeforeDetach, err := fixture.api.GetSkillCollection(
		ctx,
		gotCollection.Artifact.Ref(),
	)
	requireNoError(t, err)

	detachedCollection, err := fixture.api.RemoveSkillCollectionMember(
		ctx,
		collection.RemoveMemberRequest{
			Collection:       collectionBeforeDetach.Artifact.Ref(),
			ExpectedRevision: collectionBeforeDetach.Artifact.Revision,
			Index:            0,
		},
	)
	requireNoError(t, err)
	if len(detachedCollection.Members) != 0 {
		t.Fatalf(
			"Collection members after detach=%d, want 0",
			len(detachedCollection.Members),
		)
	}

	memberships, err = fixture.api.ListSkillCollectionMemberships(
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
	if !errors.Is(err, basespec.ErrArtifactNotFound) {
		t.Fatalf(
			"GetSkill after purge error=%v, want ErrArtifactNotFound",
			err,
		)
	}

	remainingSkills, err := fixture.api.ListSkills(
		ctx,
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	if _, found := findSkillByName(remainingSkills, skillName); found {
		t.Fatalf("purged Skill %q remains in ListSkills", skillName)
	}

	collectionBeforeDelete, err := fixture.api.GetSkillCollection(
		ctx,
		detachedCollection.Artifact.Ref(),
	)
	requireNoError(t, err)

	err = fixture.api.DeleteSkillCollection(
		ctx,
		collection.DeleteRequest{
			Collection:       collectionBeforeDelete.Artifact.Ref(),
			ExpectedRevision: collectionBeforeDelete.Artifact.Revision,
		},
	)
	requireNoError(t, err)

	_, err = fixture.api.GetSkillCollection(
		ctx,
		collectionBeforeDelete.Artifact.Ref(),
	)
	if !errors.Is(err, basespec.ErrArtifactNotFound) {
		t.Fatalf(
			"GetSkillCollection after delete error=%v, want ErrArtifactNotFound",
			err,
		)
	}

	remainingCollections, err := fixture.api.ListSkillCollections(
		ctx,
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	if len(remainingCollections) != 1 {
		t.Fatalf(
			"user Collections after cleanup=%d, want only baseline",
			len(remainingCollections),
		)
	}

	remainingBaseline, found := findCollectionByName(
		remainingCollections,
		string(collection.SkillBaselineCollectionName),
	)
	if !found {
		t.Fatal("baseline Collection disappeared during managed Skill cleanup")
	}
	if remainingBaseline.Artifact.Ref() != baseline.Artifact.Ref() {
		t.Fatalf(
			"remaining baseline ref=%+v, want %+v",
			remainingBaseline.Artifact.Ref(),
			baseline.Artifact.Ref(),
		)
	}
}
