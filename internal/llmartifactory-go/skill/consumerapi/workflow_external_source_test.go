package consumerapi_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/consumerapi"
)

func TestSkillStoreWorkflowReusesFilesystemSourceForSameSkillPath(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	const skillName = "reused-filesystem-skill"

	skillDirectory := filepath.Join(t.TempDir(), skillName)
	requireNoError(t, os.MkdirAll(skillDirectory, 0o700))

	document := workflowSkillMarkdown(
		skillName,
		"Skill used to verify filesystem Source reuse.",
		"Use the same filesystem Source on repeated registration.",
	)
	requireNoError(
		t,
		os.WriteFile(
			filepath.Join(skillDirectory, "SKILL.md"),
			document,
			0o600,
		),
	)

	ctx := t.Context()
	first, err := fixture.api.AddSkillPath(
		ctx,
		skillConsumerAPI.SkillPathRegistration{
			RootID:            topology.UserRootID(),
			Path:              skillDirectory,
			SourceDisplayName: "Reusable filesystem Skill",
			Enabled:           true,
		},
	)
	requireNoError(t, err)

	second, err := fixture.api.AddSkillPath(
		ctx,
		skillConsumerAPI.SkillPathRegistration{
			RootID:            topology.UserRootID(),
			Path:              skillDirectory,
			SourceDisplayName: "Reusable filesystem Skill",
			Enabled:           true,
		},
	)
	requireNoError(t, err)

	if second.Source.ID != first.Source.ID {
		t.Fatalf(
			"reused Source ID=%q, want %q",
			second.Source.ID,
			first.Source.ID,
		)
	}
	if second.Artifact.Ref() != first.Artifact.Ref() {
		t.Fatalf(
			"reused Artifact ref=%+v, want %+v",
			second.Artifact.Ref(),
			first.Artifact.Ref(),
		)
	}
	if second.Artifact.Revision != first.Artifact.Revision {
		t.Fatalf(
			"reused Artifact revision=%d, want %d",
			second.Artifact.Revision,
			first.Artifact.Revision,
		)
	}

	skills, err := fixture.api.ListSkills(
		ctx,
		skillConsumerAPI.ListSkillsRequest{
			RootID: topology.UserRootID(),
		},
	)
	requireNoError(t, err)
	if len(skills) != 1 {
		t.Fatalf(
			"user Skills after repeated AddSkillPath=%d, want 1",
			len(skills),
		)
	}

	reused, found := findSkillByName(skills, skillName)
	if !found {
		t.Fatalf("reused filesystem Skill %q is absent from listing", skillName)
	}
	if reused.Ref != first.Artifact.Ref() {
		t.Fatalf(
			"listed reused Skill ref=%+v, want %+v",
			reused.Ref,
			first.Artifact.Ref(),
		)
	}
}

func TestSkillStoreWorkflowAttachesExternalSkillWithoutTakingOwnership(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)
	fixture.ensureUserBaseline(t)

	const skillName = "attached-filesystem-skill"

	skillDirectory := filepath.Join(t.TempDir(), skillName)
	requireNoError(t, os.MkdirAll(skillDirectory, 0o700))

	document := workflowSkillMarkdown(
		skillName,
		"External Skill attached to a managed Plugin.",
		"The Plugin must not take ownership of this Skill package.",
	)
	requireNoError(
		t,
		os.WriteFile(
			filepath.Join(skillDirectory, "SKILL.md"),
			document,
			0o600,
		),
	)

	ctx := t.Context()
	external, err := fixture.api.AddSkillPath(
		ctx,
		skillConsumerAPI.SkillPathRegistration{
			RootID:            topology.UserRootID(),
			Path:              skillDirectory,
			SourceDisplayName: "Attached external Skill",
			Enabled:           true,
		},
	)
	requireNoError(t, err)

	collectionValue, err := fixture.api.CreateSkillPlugin(
		ctx,
		plugin.CreateRequest{
			RootID:      topology.UserRootID(),
			Name:        "external-attachment",
			DisplayName: "External attachment",
			Description: "Plugin for externally sourced Skills.",
		},
	)
	requireNoError(t, err)

	attached, err := fixture.api.AttachSkillArtifactToPlugin(
		ctx,
		plugin.AddArtifactMemberRequest{
			Plugin:           collectionValue.Artifact.Ref(),
			ExpectedRevision: collectionValue.Artifact.Revision,
			Artifact:         external.Artifact.Ref(),
		},
	)
	requireNoError(t, err)

	if len(attached.Members) != 1 {
		t.Fatalf(
			"attached Plugin members=%d, want 1",
			len(attached.Members),
		)
	}
	if string(attached.Members[0].Name) != skillName {
		t.Fatalf(
			"attached Plugin member name=%q, want %q",
			attached.Members[0].Name,
			skillName,
		)
	}

	capabilities, err := fixture.api.ResolveSkillPlugin(
		ctx,
		attached.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !capabilities.Complete {
		t.Fatal("Plugin containing external Skill is incomplete")
	}

	memberships, err := fixture.api.ListSkillPluginMemberships(
		ctx,
		external.Artifact.Ref(),
	)
	requireNoError(t, err)
	if len(memberships) != 1 {
		t.Fatalf(
			"external Skill memberships=%d, want 1",
			len(memberships),
		)
	}
	if !memberships[0].ResolvedToArtifact ||
		memberships[0].ResolvedArtifact == nil ||
		*memberships[0].ResolvedArtifact != external.Artifact.Ref() {
		t.Fatalf(
			"external Skill membership did not resolve correctly: %+v",
			memberships[0],
		)
	}

	_, err = fixture.api.ReplaceManagedSkill(
		ctx,
		skillConsumerAPI.ManagedSkillReplaceRequest{
			Plugin:                   attached.Artifact.Ref(),
			ExpectedPluginRevision:   attached.Artifact.Revision,
			Artifact:                 external.Artifact.Ref(),
			ExpectedArtifactRevision: external.Artifact.Revision,
			SkillName:                skillName,
			SKILLMD:                  document,
			Files: managedSkillFiles(
				document,
				"This package must not replace the external source.\n",
			),
			Enabled: true,
		},
	)
	if !errors.Is(err, spec.ErrUnsupported) {
		t.Fatalf(
			"replace external Skill error=%v, want ErrUnsupported",
			err,
		)
	}

	err = fixture.api.PurgeSkill(
		ctx,
		external.Artifact.Ref(),
		external.Artifact.Revision,
	)
	if !errors.Is(err, spec.ErrUnsupported) {
		t.Fatalf(
			"purge external Skill error=%v, want ErrUnsupported",
			err,
		)
	}

	detached, err := fixture.api.RemoveSkillPluginMember(
		ctx,
		plugin.RemoveMemberRequest{
			Plugin:           attached.Artifact.Ref(),
			ExpectedRevision: attached.Artifact.Revision,
			Index:            0,
		},
	)
	requireNoError(t, err)
	if len(detached.Members) != 0 {
		t.Fatalf(
			"Plugin members after detach=%d, want 0",
			len(detached.Members),
		)
	}

	requireNoError(
		t,
		fixture.api.DeleteSkillPlugin(
			ctx,
			plugin.DeleteRequest{
				Plugin:           detached.Artifact.Ref(),
				ExpectedRevision: detached.Artifact.Revision,
			},
		),
	)

	externalAfterCleanup, err := fixture.api.GetSkill(
		ctx,
		external.Artifact.Ref(),
	)
	requireNoError(t, err)
	if externalAfterCleanup.State != artifactModel.StateAvailable {
		t.Fatalf(
			"external Skill state after Plugin cleanup=%q, want %q",
			externalAfterCleanup.State,
			artifactModel.StateAvailable,
		)
	}
}
