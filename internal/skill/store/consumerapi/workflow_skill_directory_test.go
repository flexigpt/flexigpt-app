package consumerapi_test

import (
	"os"
	"path/filepath"
	"testing"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
)

func TestSkillStoreWorkflowRegistersAndRefreshesSkillDirectory(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	const (
		alphaName = "directory-alpha"
		betaName  = "directory-beta"
	)

	skillRoot := t.TempDir()
	writeWorkflowSkillFile(
		t,
		skillRoot,
		alphaName,
		workflowSkillMarkdown(
			alphaName,
			"Initial alpha directory Skill.",
			"Use the initial alpha instructions.",
		),
	)
	writeWorkflowSkillFile(
		t,
		skillRoot,
		betaName,
		workflowSkillMarkdown(
			betaName,
			"Initial beta directory Skill.",
			"Use the initial beta instructions.",
		),
	)

	ctx := t.Context()
	registered, err := fixture.api.RegisterSkillDirectory(
		ctx,
		skillConsumerAPI.SkillDirectoryRegistration{
			RootID:            documentTopology.UserRootID(),
			RootPath:          skillRoot,
			SourceDisplayName: "Directory workflow Skills",
		},
	)
	requireNoError(t, err)

	skills, err := fixture.api.ListSkills(
		ctx,
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	if len(skills) != 2 {
		t.Fatalf(
			"registered directory Skills=%d, want 2",
			len(skills),
		)
	}

	alpha, found := findSkillByName(skills, alphaName)
	if !found {
		t.Fatalf("directory Skill %q is absent from ListSkills", alphaName)
	}
	beta, found := findSkillByName(skills, betaName)
	if !found {
		t.Fatalf("directory Skill %q is absent from ListSkills", betaName)
	}
	if alpha.Binding.SourceID != registered.ID {
		t.Fatalf(
			"alpha Source ID=%q, want registered Source %q",
			alpha.Binding.SourceID,
			registered.ID,
		)
	}
	if beta.Binding.SourceID != registered.ID {
		t.Fatalf(
			"beta Source ID=%q, want registered Source %q",
			beta.Binding.SourceID,
			registered.ID,
		)
	}
	if alpha.State != artifact.StateAvailable {
		t.Fatalf(
			"alpha state=%q, want %q",
			alpha.State,
			artifact.StateAvailable,
		)
	}
	if beta.State != artifact.StateAvailable {
		t.Fatalf(
			"beta state=%q, want %q",
			beta.State,
			artifact.StateAvailable,
		)
	}

	reused, err := fixture.api.RegisterSkillDirectory(
		ctx,
		skillConsumerAPI.SkillDirectoryRegistration{
			RootID:            documentTopology.UserRootID(),
			RootPath:          skillRoot,
			SourceDisplayName: "Directory workflow Skills",
		},
	)
	requireNoError(t, err)
	if reused.ID != registered.ID {
		t.Fatalf(
			"reused Source ID=%q, want %q",
			reused.ID,
			registered.ID,
		)
	}

	disabledAlpha, err := fixture.api.SetSkillEnabled(
		ctx,
		alpha.Ref(),
		alpha.Revision,
		false,
	)
	requireNoError(t, err)
	if disabledAlpha.Enabled {
		t.Fatal("alpha Skill remains enabled after disable")
	}

	writeWorkflowSkillFile(
		t,
		skillRoot,
		alphaName,
		workflowSkillMarkdown(
			alphaName,
			"Updated alpha directory Skill.",
			"Use the updated alpha instructions after refresh.",
		),
	)

	requireNoError(
		t,
		fixture.api.RefreshSkillSource(
			ctx,
			documentTopology.UserRootID(),
			registered.ID,
		),
	)

	refreshedAlpha, err := fixture.api.GetSkill(ctx, alpha.Ref())
	requireNoError(t, err)
	if refreshedAlpha.State != artifact.StateAvailable {
		t.Fatalf(
			"refreshed alpha state=%q, want %q",
			refreshedAlpha.State,
			artifact.StateAvailable,
		)
	}
	if refreshedAlpha.Enabled {
		t.Fatal("source refresh overwrote alpha local enablement")
	}
	if refreshedAlpha.Revision <= disabledAlpha.Revision {
		t.Fatalf(
			"refreshed alpha revision=%d, want greater than disabled revision %d",
			refreshedAlpha.Revision,
			disabledAlpha.Revision,
		)
	}

	refreshedBeta, err := fixture.api.GetSkill(ctx, beta.Ref())
	requireNoError(t, err)
	if refreshedBeta.Ref() != beta.Ref() {
		t.Fatalf(
			"refreshed beta ref=%+v, want %+v",
			refreshedBeta.Ref(),
			beta.Ref(),
		)
	}
	if refreshedBeta.Revision != beta.Revision {
		t.Fatalf(
			"unmodified beta revision=%d, want %d",
			refreshedBeta.Revision,
			beta.Revision,
		)
	}

	finalSkills, err := fixture.api.ListSkills(
		ctx,
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	if len(finalSkills) != 2 {
		t.Fatalf(
			"directory Skills after refresh=%d, want 2",
			len(finalSkills),
		)
	}
}

func writeWorkflowSkillFile(
	t *testing.T,
	root string,
	name string,
	content []byte,
) {
	t.Helper()

	directory := filepath.Join(root, name)
	requireNoError(t, os.MkdirAll(directory, 0o700))
	requireNoError(
		t,
		os.WriteFile(
			filepath.Join(directory, "SKILL.md"),
			content,
			0o600,
		),
	)
}
