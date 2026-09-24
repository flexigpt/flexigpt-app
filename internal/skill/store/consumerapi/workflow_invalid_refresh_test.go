package consumerapi_test

import (
	"os"
	"path/filepath"
	"testing"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
)

func TestSkillStoreWorkflowRefreshMarksExternalSkillInvalidAndRecovers(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	const skillName = "refreshable-skill"

	skillDirectory := filepath.Join(t.TempDir(), skillName)
	requireNoError(t, os.MkdirAll(skillDirectory, 0o700))

	skillDocumentPath := filepath.Join(skillDirectory, "SKILL.md")
	validDocument := workflowSkillMarkdown(
		skillName,
		"Recover from invalid external Skill source content.",
		"Use the restored Skill package after refresh.",
	)
	requireNoError(
		t,
		os.WriteFile(skillDocumentPath, validDocument, 0o600),
	)

	ctx := t.Context()
	registered, err := fixture.api.AddSkillPath(
		ctx,
		skillConsumerAPI.SkillPathRegistration{
			RootID:            documentTopology.UserRootID(),
			Path:              skillDirectory,
			SourceDisplayName: "Refreshable external Skill",
			Enabled:           true,
		},
	)
	requireNoError(t, err)

	invalidDocument := []byte(
		"---\n" +
			"name: invalid skill name with spaces\n" +
			"---\n\n" +
			"# Broken\n",
	)
	requireNoError(
		t,
		os.WriteFile(skillDocumentPath, invalidDocument, 0o600),
	)

	requireNoError(
		t,
		fixture.api.RefreshSkillSource(
			ctx,
			registered.Artifact.RootID,
			registered.Source.ID,
		),
	)

	invalid, err := fixture.api.GetSkill(
		ctx,
		registered.Artifact.Ref(),
	)
	requireNoError(t, err)
	if invalid.State != artifact.StateInvalid {
		t.Fatalf(
			"invalid external Skill state=%q, want %q",
			invalid.State,
			artifact.StateInvalid,
		)
	}
	if invalid.ResolvedDefinition != nil {
		t.Fatal("invalid external Skill retained a resolved Definition")
	}

	listedSkills, err := fixture.api.ListSkills(
		ctx,
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	listedInvalid, found := findSkillByName(listedSkills, skillName)
	if !found {
		t.Fatalf("invalid external Skill %q disappeared from ListSkills", skillName)
	}
	if listedInvalid.State != artifact.StateInvalid {
		t.Fatalf(
			"listed invalid external Skill state=%q, want %q",
			listedInvalid.State,
			artifact.StateInvalid,
		)
	}

	requireNoError(
		t,
		os.WriteFile(skillDocumentPath, validDocument, 0o600),
	)

	requireNoError(
		t,
		fixture.api.RefreshSkillSource(
			ctx,
			registered.Artifact.RootID,
			registered.Source.ID,
		),
	)

	restored, err := fixture.api.GetSkill(
		ctx,
		registered.Artifact.Ref(),
	)
	requireNoError(t, err)
	if restored.Ref() != registered.Artifact.Ref() {
		t.Fatalf(
			"restored Skill ref=%+v, want %+v",
			restored.Ref(),
			registered.Artifact.Ref(),
		)
	}
	if restored.State != artifact.StateAvailable {
		t.Fatalf(
			"restored external Skill state=%q, want %q",
			restored.State,
			artifact.StateAvailable,
		)
	}
	if restored.ResolvedDefinition == nil {
		t.Fatal("restored external Skill has no resolved Definition")
	}
	if restored.Revision <= invalid.Revision {
		t.Fatalf(
			"restored Skill revision=%d, want greater than invalid revision %d",
			restored.Revision,
			invalid.Revision,
		)
	}
}
