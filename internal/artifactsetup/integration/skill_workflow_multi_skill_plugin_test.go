package integration

import (
	"errors"
	"testing"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
)

func TestSkillStoreWorkflowKeepsRemainingManagedSkillAvailableDuringPartialCleanup(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	ctx := t.Context()

	pluginValue, err := fixture.api.CreateSkillPlugin(
		ctx,
		pluginAPI.CreateRequest{
			RootID:      topology.UserRootID(),
			Name:        "multi-skill-workflow",
			DisplayName: "Multi Skill workflow",
			Description: "Plugin used to verify partial Skill cleanup.",
		},
	)
	requireNoError(t, err)

	const (
		firstSkillName  = "first-multi-skill"
		secondSkillName = "second-multi-skill"
	)

	first, _ := createManagedSkillInPlugin(
		t,
		fixture.api,
		pluginValue,
		firstSkillName,
		"First managed Skill.",
		"Use the first initial instructions.",
		"First checklist.\n",
	)
	second, _ := createManagedSkillInPlugin(
		t,
		fixture.api,
		first.Plugin,
		secondSkillName,
		"Second managed Skill.",
		"Use the second initial instructions.",
		"Second checklist.\n",
	)

	pluginAfterCreate, err := fixture.api.GetSkillPlugin(
		ctx,
		second.Plugin.Artifact.Ref(),
	)
	requireNoError(t, err)
	if len(pluginAfterCreate.Members) != 2 {
		t.Fatalf(
			"Plugin members after two Skill creates=%d, want 2",
			len(pluginAfterCreate.Members),
		)
	}

	capabilities, err := fixture.api.ResolveSkillPlugin(
		ctx,
		pluginAfterCreate.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !capabilities.Complete {
		t.Fatal("Plugin with two managed Skills is incomplete")
	}
	if !capabilityPlanContainsArtifact(
		capabilities.Occurrences,
		first.Artifact.Ref(),
	) {
		t.Fatal("Plugin capability plan does not contain first Skill")
	}
	if !capabilityPlanContainsArtifact(
		capabilities.Occurrences,
		second.Artifact.Ref(),
	) {
		t.Fatal("Plugin capability plan does not contain second Skill")
	}

	aggregateService, _ := newSkillAggregateService(t, fixture)

	refs, err := aggregateService.ListArtifactSkillRefs(
		ctx,
		skillAggregate.ArtifactSkillFilter{
			AllowArtifacts: []artifactModel.ArtifactRef{
				first.Artifact.Ref(),
				second.Artifact.Ref(),
			},
		},
	)
	requireNoError(t, err)
	if !artifactRefSetEquals(
		refs,
		[]artifactModel.ArtifactRef{
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
		skillAPI.ManagedSkillReplaceRequest{
			Plugin:                   pluginAfterCreate.Artifact.Ref(),
			ExpectedPluginRevision:   pluginAfterCreate.Artifact.Revision,
			Artifact:                 firstCurrent.Ref(),
			ExpectedArtifactRevision: firstCurrent.Revision,
			SkillName:                firstSkillName,
			SKILLMD:                  replacementDocument,
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

	pluginBeforeDetach, err := fixture.api.GetSkillPlugin(
		ctx,
		replacedFirst.Plugin.Artifact.Ref(),
	)
	requireNoError(t, err)

	firstMemberIndex, found := pluginMemberIndexByName(
		pluginBeforeDetach.Members,
		firstSkillName,
	)
	if !found {
		t.Fatalf(
			"Plugin does not contain first Skill member %q",
			firstSkillName,
		)
	}

	pluginAfterDetach, err := fixture.api.RemoveSkillPluginMember(
		ctx,
		pluginAPI.RemoveMemberRequest{
			Plugin:           pluginBeforeDetach.Artifact.Ref(),
			ExpectedRevision: pluginBeforeDetach.Artifact.Revision,
			Index:            firstMemberIndex,
		},
	)
	requireNoError(t, err)
	if len(pluginAfterDetach.Members) != 1 {
		t.Fatalf(
			"Plugin members after first detach=%d, want 1",
			len(pluginAfterDetach.Members),
		)
	}
	if string(pluginAfterDetach.Members[0].Name) != secondSkillName {
		t.Fatalf(
			"remaining Plugin member=%q, want %q",
			pluginAfterDetach.Members[0].Name,
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
	if !errors.Is(err, spec.ErrReferenceUnresolved) {
		t.Fatalf(
			"aggregate resolution after first Skill purge error=%v, want ErrReferenceUnresolved",
			err,
		)
	}

	pluginAfterFirstPurge, err := fixture.api.GetSkillPlugin(
		ctx,
		pluginAfterDetach.Artifact.Ref(),
	)
	requireNoError(t, err)

	capabilities, err = fixture.api.ResolveSkillPlugin(
		ctx,
		pluginAfterFirstPurge.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !capabilities.Complete {
		t.Fatal("Plugin is incomplete after first Skill partial cleanup")
	}
	if capabilityPlanContainsArtifact(
		capabilities.Occurrences,
		first.Artifact.Ref(),
	) {
		t.Fatal("Plugin capability plan still contains purged first Skill")
	}
	if !capabilityPlanContainsArtifact(
		capabilities.Occurrences,
		second.Artifact.Ref(),
	) {
		t.Fatal("Plugin capability plan lost second Skill after partial cleanup")
	}

	secondAfterFirstPurge, err := fixture.api.GetSkill(
		ctx,
		second.Artifact.Ref(),
	)
	requireNoError(t, err)
	if secondAfterFirstPurge.State != artifactModel.StateAvailable {
		t.Fatalf(
			"second Skill state after first purge=%q, want %q",
			secondAfterFirstPurge.State,
			artifactModel.StateAvailable,
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

	secondMemberIndex, found := pluginMemberIndexByName(
		pluginAfterFirstPurge.Members,
		secondSkillName,
	)
	if !found {
		t.Fatalf(
			"Plugin does not contain second Skill member %q",
			secondSkillName,
		)
	}

	pluginAfterSecondDetach, err := fixture.api.RemoveSkillPluginMember(
		ctx,
		pluginAPI.RemoveMemberRequest{
			Plugin:           pluginAfterFirstPurge.Artifact.Ref(),
			ExpectedRevision: pluginAfterFirstPurge.Artifact.Revision,
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
		fixture.api.DeleteSkillPlugin(
			ctx,
			pluginAPI.DeleteRequest{
				Plugin: pluginAfterSecondDetach.Artifact.Ref(),
				ExpectedRevision: pluginAfterSecondDetach.
					Artifact.Revision,
			},
		),
	)

	remainingSkills, err := fixture.api.ListSkills(
		ctx,
		skillAPI.ListSkillsRequest{
			RootID: topology.UserRootID(),
		},
	)
	requireNoError(t, err)
	if len(remainingSkills) != 0 {
		t.Fatalf(
			"user Skills after multi-Skill cleanup=%d, want 0",
			len(remainingSkills),
		)
	}
}

func pluginMemberIndexByName(
	values []pluginAPI.MemberReference,
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
	values []composition.CapabilityOccurrence,
	ref artifactModel.ArtifactRef,
) bool {
	for _, value := range values {
		if value.Target != nil &&
			value.Target.Form == composition.TargetFormArtifact &&
			value.Target.Artifact != nil &&
			*value.Target.Artifact == ref {
			return true
		}
	}
	return false
}

func artifactRefSetEquals(
	actual []artifactModel.ArtifactRef,
	expected []artifactModel.ArtifactRef,
) bool {
	if len(actual) != len(expected) {
		return false
	}

	found := make(map[artifactModel.ArtifactRef]struct{}, len(actual))
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
