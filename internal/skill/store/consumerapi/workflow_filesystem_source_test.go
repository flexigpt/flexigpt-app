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

func TestSkillStoreWorkflowAddsAndRefreshesFilesystemSkill(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	const skillName = "filesystem-notes"

	skillDirectory := filepath.Join(t.TempDir(), skillName)
	requireNoError(t, os.MkdirAll(skillDirectory, 0o700))

	initialDocument := workflowSkillMarkdown(
		skillName,
		"Create notes from local filesystem sources.",
		"Use the local source as the authoritative input.",
	)
	skillDocumentPath := filepath.Join(skillDirectory, "SKILL.md")
	requireNoError(
		t,
		os.WriteFile(skillDocumentPath, initialDocument, 0o600),
	)

	ctx := t.Context()
	registered, err := fixture.api.AddSkillPath(
		ctx,
		skillConsumerAPI.SkillPathRegistration{
			RootID:            documentTopology.UserRootID(),
			Path:              skillDirectory,
			SourceDisplayName: "Filesystem notes Skill",
			Enabled:           true,
		},
	)
	requireNoError(t, err)

	if registered.Source.Kind != source.SourceKindFilesystemDirectory {
		t.Fatalf(
			"filesystem Skill Source kind=%q, want %q",
			registered.Source.Kind,
			source.SourceKindFilesystemDirectory,
		)
	}
	if registered.Artifact.State != artifact.StateAvailable {
		t.Fatalf(
			"filesystem Skill state=%q, want %q",
			registered.Artifact.State,
			artifact.StateAvailable,
		)
	}
	if !registered.Artifact.Enabled {
		t.Fatal("filesystem Skill is disabled after AddSkillPath")
	}
	if registered.Artifact.ResolvedDefinition == nil {
		t.Fatal("filesystem Skill has no resolved Definition")
	}

	initialDefinition := *registered.Artifact.ResolvedDefinition

	listedSkills, err := fixture.api.ListSkills(
		ctx,
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	listedSkill, found := findSkillByName(listedSkills, skillName)
	if !found {
		t.Fatalf("filesystem Skill %q is absent from ListSkills", skillName)
	}
	if listedSkill.Ref() != registered.Artifact.Ref() {
		t.Fatalf(
			"listed filesystem Skill ref=%+v, want %+v",
			listedSkill.Ref(),
			registered.Artifact.Ref(),
		)
	}

	replacementDocument := workflowSkillMarkdown(
		skillName,
		"Create refreshed notes from local filesystem sources.",
		"Use the refreshed local source as the authoritative input.",
	)
	requireNoError(
		t,
		os.WriteFile(skillDocumentPath, replacementDocument, 0o600),
	)

	requireNoError(
		t,
		fixture.api.RefreshSkillSource(
			ctx,
			registered.Artifact.RootID,
			registered.Source.ID,
		),
	)

	refreshed, err := fixture.api.GetSkill(
		ctx,
		registered.Artifact.Ref(),
	)
	requireNoError(t, err)

	if refreshed.Ref() != registered.Artifact.Ref() {
		t.Fatalf(
			"refreshed filesystem Skill ref=%+v, want %+v",
			refreshed.Ref(),
			registered.Artifact.Ref(),
		)
	}
	if refreshed.Revision <= registered.Artifact.Revision {
		t.Fatalf(
			"refreshed filesystem Skill revision=%d, want greater than %d",
			refreshed.Revision,
			registered.Artifact.Revision,
		)
	}
	if refreshed.ResolvedDefinition == nil {
		t.Fatal("refreshed filesystem Skill has no resolved Definition")
	}
	if *refreshed.ResolvedDefinition == initialDefinition {
		t.Fatal("refresh did not update the filesystem Skill Definition")
	}

	_, err = fixture.api.GetManagedSkillDocument(ctx, refreshed.Ref())
	if !errors.Is(err, basespec.ErrUnsupported) {
		t.Fatalf(
			"GetManagedSkillDocument for filesystem Skill error=%v, want ErrUnsupported",
			err,
		)
	}
}
