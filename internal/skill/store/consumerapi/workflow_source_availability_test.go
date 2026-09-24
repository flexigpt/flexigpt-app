package consumerapi_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
)

func TestSkillStoreWorkflowRecoversAfterSourceDisableAndReenable(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	const skillName = "availability-recovery"

	skillDirectory := filepath.Join(t.TempDir(), skillName)
	requireNoError(t, os.MkdirAll(skillDirectory, 0o700))

	skillDocumentPath := filepath.Join(skillDirectory, "SKILL.md")
	requireNoError(
		t,
		os.WriteFile(
			skillDocumentPath,
			workflowSkillMarkdown(
				skillName,
				"Skill used to verify Source availability recovery.",
				"Use the source only while it is available.",
			),
			0o600,
		),
	)

	ctx := t.Context()
	registered, err := fixture.api.AddSkillPath(
		ctx,
		skillConsumerAPI.SkillPathRegistration{
			RootID:            documentTopology.UserRootID(),
			Path:              skillDocumentPath,
			SourceDisplayName: "Availability recovery Skill",
			Enabled:           true,
		},
	)
	requireNoError(t, err)

	if registered.Artifact.Binding.Locator != "SKILL.md" {
		t.Fatalf(
			"file-path registration locator=%q, want SKILL.md",
			registered.Artifact.Binding.Locator,
		)
	}

	aggregateService, _ := newSkillAggregateService(t, fixture)

	initial, err := aggregateService.ResolveArtifactSkill(
		ctx,
		registered.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !initial.Enabled {
		t.Fatal("aggregate resolved initial external Skill as disabled")
	}

	disabledSource, err := fixture.store.Sources.Update(
		ctx,
		registered.Source.RootID,
		registered.Source.ID,
		source.Update{
			ExpectedRevision: registered.Source.Revision,
			DisplayName:      registered.Source.DisplayName,
			Enabled:          false,
		},
	)
	requireNoError(t, err)
	if disabledSource.Enabled {
		t.Fatal("Source remains enabled after disable")
	}

	missing, err := fixture.api.GetSkill(
		ctx,
		registered.Artifact.Ref(),
	)
	requireNoError(t, err)
	if missing.State != artifact.StateMissing {
		t.Fatalf(
			"Skill state after Source disable=%q, want %q",
			missing.State,
			artifact.StateMissing,
		)
	}
	if missing.ResolvedDefinition != nil {
		t.Fatal("missing Skill retained a resolved Definition")
	}
	if !missing.Enabled {
		t.Fatal("Source disable unexpectedly changed local Skill enablement")
	}

	listedSkills, err := fixture.api.ListSkills(
		ctx,
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	listedMissing, found := findSkillByName(listedSkills, skillName)
	if !found {
		t.Fatalf("missing Skill %q disappeared from ListSkills", skillName)
	}
	if listedMissing.State != artifact.StateMissing {
		t.Fatalf(
			"listed missing Skill state=%q, want %q",
			listedMissing.State,
			artifact.StateMissing,
		)
	}

	_, err = aggregateService.ResolveArtifactSkill(ctx, missing.Ref())
	if !errors.Is(err, basespec.ErrReferenceUnresolved) {
		t.Fatalf(
			"aggregate resolution after Source disable error=%v, want ErrReferenceUnresolved",
			err,
		)
	}

	reenabledSource, err := fixture.store.Sources.Update(
		ctx,
		disabledSource.RootID,
		disabledSource.ID,
		source.Update{
			ExpectedRevision: disabledSource.Revision,
			DisplayName:      disabledSource.DisplayName,
			Enabled:          true,
		},
	)
	requireNoError(t, err)
	if !reenabledSource.Enabled {
		t.Fatal("Source remains disabled after enable")
	}

	requireNoError(
		t,
		fixture.api.RefreshSkillSource(
			ctx,
			reenabledSource.RootID,
			reenabledSource.ID,
		),
	)

	restored, err := fixture.api.GetSkill(ctx, missing.Ref())
	requireNoError(t, err)
	if restored.State != artifact.StateAvailable {
		t.Fatalf(
			"Skill state after Source recovery=%q, want %q",
			restored.State,
			artifact.StateAvailable,
		)
	}
	if restored.ResolvedDefinition == nil {
		t.Fatal("restored Skill has no resolved Definition")
	}
	if !restored.Enabled {
		t.Fatal("Source recovery did not preserve local Skill enablement")
	}
	if restored.Revision <= missing.Revision {
		t.Fatalf(
			"restored Skill revision=%d, want greater than missing revision %d",
			restored.Revision,
			missing.Revision,
		)
	}

	recovered, err := aggregateService.ResolveArtifactSkill(
		ctx,
		restored.Ref(),
	)
	requireNoError(t, err)
	if !recovered.Enabled {
		t.Fatal("aggregate resolved recovered Skill as disabled")
	}
}
