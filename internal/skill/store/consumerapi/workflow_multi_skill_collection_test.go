package consumerapi_test

import (
	"errors"
	"testing"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
)

func TestSkillStoreWorkflowKeepsRemainingManagedSkillAvailableDuringPartialCleanup(
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
			Name:        "multi-skill-workflow",
			DisplayName: "Multi Skill workflow",
			Description: "Collection used to verify partial Skill cleanup.",
		},
	)
	requireNoError(t, err)

	const (
		firstSkillName  = "first-multi-skill"
		secondSkillName = "second-multi-skill"
	)

	first, _ := createManagedSkillInCollection(
		t,
		fixture.api,
		collectionValue,
		firstSkillName,
		"First managed Skill.",
		"Use the first initial instructions.",
		"First checklist.\n",
	)
	second, _ := createManagedSkillInCollection(
		t,
		fixture.api,
		first.Collection,
		secondSkillName,
		"Second managed Skill.",
		"Use the second initial instructions.",
		"Second checklist.\n",
	)

	collectionAfterCreate, err := fixture.api.GetSkillCollection(
		ctx,
		second.Collection.Artifact.Ref(),
	)
	requireNoError(t, err)
	if len(collectionAfterCreate.Members) != 2 {
		t.Fatalf(
			"Collection members after two Skill creates=%d, want 2",
			len(collectionAfterCreate.Members),
		)
	}

	capabilities, err := fixture.api.ResolveSkillCollection(
		ctx,
		collectionAfterCreate.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !capabilities.Complete {
		t.Fatal("Collection with two managed Skills is incomplete")
	}
	if !capabilityPlanContainsArtifact(
		capabilities.Occurrences,
		first.Artifact.Ref(),
	) {
		t.Fatal("Collection capability plan does not contain first Skill")
	}
	if !capabilityPlanContainsArtifact(
		capabilities.Occurrences,
		second.Artifact.Ref(),
	) {
		t.Fatal("Collection capability plan does not contain second Skill")
	}

	aggregateService, _ := newSkillAggregateService(t, fixture)

	refs, err := aggregateService.ListArtifactSkillRefs(
		ctx,
		skillAggregate.ArtifactSkillFilter{
			AllowArtifacts: []artifact.ArtifactRef{
				first.Artifact.Ref(),
				second.Artifact.Ref(),
			},
		},
	)
	requireNoError(t, err)
	if !artifactRefSetEquals(
		refs,
		[]artifact.ArtifactRef{
			first.Artifact.Ref(),
			second.Artifact.Ref(),
		},
	) {
		t.Fatalf(
			"aggregate refs=%+v, want first and second Skill refs",
			refs,
		)
	}

	firstCurrent, err := fixture.api.GetSkill(ctx, first.Artifact.Ref())
	requireNoError(t, err)

	replacementDocument := workflowSkillMarkdown(
		firstSkillName,
		"Updated first managed Skill.",
		"Use the first replacement instructions.",
	)
	replacedFirst, err := fixture.api.ReplaceManagedSkill(
		ctx,
		skillConsumerAPI.ManagedSkillReplaceRequest{
			Collection:                 collectionAfterCreate.Artifact.Ref(),
			ExpectedCollectionRevision: collectionAfterCreate.Artifact.Revision,
			Artifact:                   firstCurrent.Ref(),
			ExpectedArtifactRevision:   firstCurrent.Revision,
			SkillName:                  firstSkillName,
			SKILLMD:                    replacementDocument,
			Files: managedSkillFiles(
				replacementDocument,
				"Updated first checklist.\n",
			),
			Enabled: true,
		},
	)
	requireNoError(t, err)

	if replacedFirst.Artifact.Ref() != first.Artifact.Ref() {
		t.Fatalf(
			"first Skill replacement ref=%+v, want %+v",
			replacedFirst.Artifact.Ref(),
			first.Artifact.Ref(),
		)
	}

	secondAfterFirstReplacement, err := fixture.api.GetSkill(
		ctx,
		second.Artifact.Ref(),
	)
	requireNoError(t, err)
	if secondAfterFirstReplacement.Ref() != second.Artifact.Ref() {
		t.Fatalf(
			"second Skill ref after first replacement=%+v, want %+v",
			secondAfterFirstReplacement.Ref(),
			second.Artifact.Ref(),
		)
	}
	if secondAfterFirstReplacement.Revision != second.Artifact.Revision {
		t.Fatalf(
			"second Skill revision after unrelated replacement=%d, want %d",
			secondAfterFirstReplacement.Revision,
			second.Artifact.Revision,
		)
	}

	secondResolved, err := aggregateService.ResolveArtifactSkill(
		ctx,
		secondAfterFirstReplacement.Ref(),
	)
	requireNoError(t, err)
	if !secondResolved.Enabled {
		t.Fatal("aggregate resolved second Skill as disabled")
	}

	collectionBeforeDetach, err := fixture.api.GetSkillCollection(
		ctx,
		replacedFirst.Collection.Artifact.Ref(),
	)
	requireNoError(t, err)

	firstMemberIndex, found := collectionMemberIndexByName(
		collectionBeforeDetach.Members,
		firstSkillName,
	)
	if !found {
		t.Fatalf(
			"Collection does not contain first Skill member %q",
			firstSkillName,
		)
	}

	collectionAfterDetach, err := fixture.api.RemoveSkillCollectionMember(
		ctx,
		collection.RemoveMemberRequest{
			Collection:       collectionBeforeDetach.Artifact.Ref(),
			ExpectedRevision: collectionBeforeDetach.Artifact.Revision,
			Index:            firstMemberIndex,
		},
	)
	requireNoError(t, err)
	if len(collectionAfterDetach.Members) != 1 {
		t.Fatalf(
			"Collection members after first detach=%d, want 1",
			len(collectionAfterDetach.Members),
		)
	}
	if string(collectionAfterDetach.Members[0].Name) != secondSkillName {
		t.Fatalf(
			"remaining Collection member=%q, want %q",
			collectionAfterDetach.Members[0].Name,
			secondSkillName,
		)
	}

	firstBeforePurge, err := fixture.api.GetSkill(
		ctx,
		replacedFirst.Artifact.Ref(),
	)
	requireNoError(t, err)
	requireNoError(
		t,
		fixture.api.PurgeSkill(
			ctx,
			firstBeforePurge.Ref(),
			firstBeforePurge.Revision,
		),
	)

	_, err = aggregateService.ResolveArtifactSkill(
		ctx,
		firstBeforePurge.Ref(),
	)
	if !errors.Is(err, basespec.ErrReferenceUnresolved) {
		t.Fatalf(
			"aggregate resolution after first Skill purge error=%v, want ErrReferenceUnresolved",
			err,
		)
	}

	collectionAfterFirstPurge, err := fixture.api.GetSkillCollection(
		ctx,
		collectionAfterDetach.Artifact.Ref(),
	)
	requireNoError(t, err)

	capabilities, err = fixture.api.ResolveSkillCollection(
		ctx,
		collectionAfterFirstPurge.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !capabilities.Complete {
		t.Fatal("Collection is incomplete after first Skill partial cleanup")
	}
	if capabilityPlanContainsArtifact(
		capabilities.Occurrences,
		first.Artifact.Ref(),
	) {
		t.Fatal("Collection capability plan still contains purged first Skill")
	}
	if !capabilityPlanContainsArtifact(
		capabilities.Occurrences,
		second.Artifact.Ref(),
	) {
		t.Fatal("Collection capability plan lost second Skill after partial cleanup")
	}

	secondAfterFirstPurge, err := fixture.api.GetSkill(
		ctx,
		second.Artifact.Ref(),
	)
	requireNoError(t, err)
	if secondAfterFirstPurge.State != artifact.StateAvailable {
		t.Fatalf(
			"second Skill state after first purge=%q, want %q",
			secondAfterFirstPurge.State,
			artifact.StateAvailable,
		)
	}

	secondResolved, err = aggregateService.ResolveArtifactSkill(
		ctx,
		secondAfterFirstPurge.Ref(),
	)
	requireNoError(t, err)
	if !secondResolved.Enabled {
		t.Fatal("aggregate resolved remaining second Skill as disabled")
	}

	secondMemberIndex, found := collectionMemberIndexByName(
		collectionAfterFirstPurge.Members,
		secondSkillName,
	)
	if !found {
		t.Fatalf(
			"Collection does not contain second Skill member %q",
			secondSkillName,
		)
	}

	collectionAfterSecondDetach, err := fixture.api.RemoveSkillCollectionMember(
		ctx,
		collection.RemoveMemberRequest{
			Collection:       collectionAfterFirstPurge.Artifact.Ref(),
			ExpectedRevision: collectionAfterFirstPurge.Artifact.Revision,
			Index:            secondMemberIndex,
		},
	)
	requireNoError(t, err)

	secondBeforePurge, err := fixture.api.GetSkill(
		ctx,
		secondAfterFirstPurge.Ref(),
	)
	requireNoError(t, err)
	requireNoError(
		t,
		fixture.api.PurgeSkill(
			ctx,
			secondBeforePurge.Ref(),
			secondBeforePurge.Revision,
		),
	)

	requireNoError(
		t,
		fixture.api.DeleteSkillCollection(
			ctx,
			collection.DeleteRequest{
				Collection: collectionAfterSecondDetach.Artifact.Ref(),
				ExpectedRevision: collectionAfterSecondDetach.
					Artifact.Revision,
			},
		),
	)

	remainingSkills, err := fixture.api.ListSkills(
		ctx,
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	if len(remainingSkills) != 0 {
		t.Fatalf(
			"user Skills after multi-Skill cleanup=%d, want 0",
			len(remainingSkills),
		)
	}
}

func collectionMemberIndexByName(
	values []collection.MemberReference,
	name string,
) (int, bool) {
	for index, value := range values {
		if string(value.Name) == name {
			return index, true
		}
	}
	return 0, false
}

func capabilityPlanContainsArtifact(
	values []collection.CollectionCapabilityOccurrence,
	ref artifact.ArtifactRef,
) bool {
	for _, value := range values {
		if value.Artifact != nil && *value.Artifact == ref {
			return true
		}
	}
	return false
}

func artifactRefSetEquals(
	actual []artifact.ArtifactRef,
	expected []artifact.ArtifactRef,
) bool {
	if len(actual) != len(expected) {
		return false
	}

	found := make(map[artifact.ArtifactRef]struct{}, len(actual))
	for _, value := range actual {
		found[value] = struct{}{}
	}
	if len(found) != len(expected) {
		return false
	}

	for _, value := range expected {
		if _, exists := found[value]; !exists {
			return false
		}
	}
	return true
}
