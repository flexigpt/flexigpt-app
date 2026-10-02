package consumerapi_test

import (
	"errors"
	"testing"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
)

func TestSkillStoreWorkflowAggregateHonorsBuiltinSkillEnablement(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)

	ctx := t.Context()

	builtinSkills, err := fixture.api.ListSkills(
		ctx,
		skillConsumerAPI.ListSkillsRequest{
			RootID: documentTopology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)

	markdownOutput, found := findSkillByName(
		builtinSkills,
		"markdown-output",
	)
	if !found {
		t.Fatal("markdown-output built-in Skill was not installed")
	}

	markdownOutputRecord, err := fixture.api.GetSkill(
		ctx,
		markdownOutput.Ref,
	)
	requireNoError(t, err)

	sourceBefore, err := fixture.store.Sources.Get(
		ctx,
		markdownOutputRecord.RootID,
		markdownOutputRecord.Binding.SourceID,
	)
	requireNoError(t, err)

	aggregateService, _ := newSkillAggregateService(t, fixture)

	initial, err := aggregateService.ResolveArtifactSkill(
		ctx,
		markdownOutput.Ref,
	)
	requireNoError(t, err)
	if !initial.Enabled {
		t.Fatal("aggregate resolved built-in Skill as disabled")
	}

	disabled, err := fixture.api.SetSkillEnabled(
		ctx,
		markdownOutput.Ref,
		markdownOutput.Revision,
		false,
	)
	requireNoError(t, err)
	if disabled.Enabled {
		t.Fatal("built-in Skill remains enabled after disable")
	}

	sourceAfterDisable, err := fixture.store.Sources.Get(
		ctx,
		markdownOutputRecord.RootID,
		markdownOutputRecord.Binding.SourceID,
	)
	requireNoError(t, err)
	if sourceAfterDisable.Revision != sourceBefore.Revision {
		t.Fatalf(
			"disabling built-in Skill changed Source revision from %d to %d",
			sourceBefore.Revision,
			sourceAfterDisable.Revision,
		)
	}

	_, err = aggregateService.ResolveArtifactSkill(ctx, disabled.Ref())
	if !errors.Is(err, basespec.ErrReferenceUnresolved) {
		t.Fatalf(
			"aggregate resolution after built-in disable error=%v, want ErrReferenceUnresolved",
			err,
		)
	}

	reenabled, err := fixture.api.SetSkillEnabled(
		ctx,
		disabled.Ref(),
		disabled.Revision,
		true,
	)
	requireNoError(t, err)
	if !reenabled.Enabled {
		t.Fatal("built-in Skill remains disabled after enable")
	}

	restored, err := aggregateService.ResolveArtifactSkill(
		ctx,
		reenabled.Ref(),
	)
	requireNoError(t, err)
	if !restored.Enabled {
		t.Fatal("aggregate resolved re-enabled built-in Skill as disabled")
	}
	if restored.Artifact != markdownOutput.Ref {
		t.Fatalf(
			"restored built-in Artifact ref=%+v, want %+v",
			restored.Artifact,
			markdownOutput.Ref,
		)
	}

	sourceAfterEnable, err := fixture.store.Sources.Get(
		ctx,
		markdownOutputRecord.RootID,
		markdownOutputRecord.Binding.SourceID,
	)
	requireNoError(t, err)
	if sourceAfterEnable.Revision != sourceBefore.Revision {
		t.Fatalf(
			"re-enabling built-in Skill changed Source revision from %d to %d",
			sourceBefore.Revision,
			sourceAfterEnable.Revision,
		)
	}
}
