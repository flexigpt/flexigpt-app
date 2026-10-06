package skill_test

import (
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill"
)

func TestSkillStoreWorkflowBootstrapsBuiltinsAndUserBaseline(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	ctx := t.Context()

	roots, err := fixture.store.Roots.List(ctx)
	requireNoError(t, err)

	userRootFound := false
	for _, value := range roots {
		if value.ID == topology.UserRootID() {
			userRootFound = true
			break
		}
	}
	if !userRootFound {
		t.Fatalf(
			"retained user Root %q was not created by composition startup",
			topology.UserRootID(),
		)
	}

	_, err = fixture.store.Roots.Get(
		ctx,
		topology.BuiltinRootID(),
	)
	if !errors.Is(err, spec.ErrRootNotFound) {
		t.Fatalf(
			"built-in Root before hydration error=%v, want ErrRootNotFound",
			err,
		)
	}

	initialUserSkills, err := fixture.api.ListSkills(
		ctx,
		skillAPI.ListSkillsRequest{
			RootID: topology.UserRootID(),
		},
	)
	requireNoError(t, err)
	if len(initialUserSkills) != 0 {
		t.Fatalf(
			"user Skills before baseline provisioning=%d, want 0",
			len(initialUserSkills),
		)
	}

	fixture.bootstrapBuiltins(t)

	builtinSource, err := fixture.store.Sources.Get(
		ctx,
		topology.BuiltinRootID(),
		topology.BuiltinPackageSourceID(),
	)
	requireNoError(t, err)
	if builtinSource.Kind != managedfs.Kind {
		t.Fatalf(
			"built-in Source kind=%q, want %q",
			builtinSource.Kind,
			managedfs.Kind,
		)
	}
	if !builtinSource.Enabled {
		t.Fatal("built-in package Source is disabled after hydration")
	}

	builtinSkills, err := fixture.api.ListSkills(
		ctx,
		skillAPI.ListSkillsRequest{
			RootID: topology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)
	if len(builtinSkills) == 0 {
		t.Fatal("built-in hydration produced no Skill Artifacts")
	}

	markdownOutput, found := findSkillByName(
		builtinSkills,
		"markdown-output",
	)
	if !found {
		t.Fatal("embedded markdown-output Skill was not installed")
	}
	if markdownOutput.State != artifactModel.StateAvailable {
		t.Fatalf(
			"markdown-output state=%q, want %q",
			markdownOutput.State,
			artifactModel.StateAvailable,
		)
	}
	if !markdownOutput.Enabled {
		t.Fatal("markdown-output is disabled after built-in hydration")
	}

	builtinPlugins, err := fixture.api.ListSkillPlugins(
		ctx,
		topology.BuiltinRootID(),
	)
	requireNoError(t, err)
	if len(builtinPlugins) == 0 {
		t.Fatal("built-in hydration produced no visible Skill Plugins")
	}
	for _, value := range builtinPlugins {
		if value.Editable {
			t.Fatalf(
				"built-in Plugin %q is unexpectedly editable",
				value.Name,
			)
		}
		if value.Deletable {
			t.Fatalf(
				"built-in Plugin %q is unexpectedly deletable",
				value.Name,
			)
		}
	}

	baseline := fixture.ensureUserBaseline(t)
	if baseline.Name != pluginAPI.SkillBaselinePluginName {
		t.Fatalf(
			"baseline name=%q, want %q",
			baseline.Name,
			pluginAPI.SkillBaselinePluginName,
		)
	}
	if !baseline.Baseline {
		t.Fatal("user Skill baseline is not marked as baseline")
	}
	if !baseline.Editable {
		t.Fatal("user Skill baseline is not editable")
	}
	if baseline.Deletable {
		t.Fatal("user Skill baseline is unexpectedly deletable")
	}
	if len(baseline.Members) != 0 {
		t.Fatalf(
			"user Skill baseline members=%d, want 0",
			len(baseline.Members),
		)
	}

	userPlugins, err := fixture.api.ListSkillPlugins(
		ctx,
		topology.UserRootID(),
	)
	requireNoError(t, err)
	listedBaseline, found := findPluginByName(
		userPlugins,
		string(pluginAPI.SkillBaselinePluginName),
	)
	if !found {
		t.Fatal("user Skill baseline is absent from Plugin listing")
	}
	if listedBaseline.Ref != baseline.Artifact.Ref() {
		t.Fatalf(
			"listed baseline ref=%+v, want %+v",
			listedBaseline.Ref,
			baseline.Artifact.Ref(),
		)
	}

	err = fixture.api.DeleteSkillPlugin(
		ctx,
		pluginAPI.DeleteRequest{
			Plugin:           baseline.Artifact.Ref(),
			ExpectedRevision: baseline.Artifact.Revision,
		},
	)
	if !errors.Is(err, spec.ErrProtected) {
		t.Fatalf(
			"delete baseline error=%v, want ErrProtected",
			err,
		)
	}

	beforeRehydrate := skillRevisionSnapshot(builtinSkills)

	fixture.bootstrapBuiltins(t)

	afterRehydrate, err := fixture.api.ListSkills(
		ctx,
		skillAPI.ListSkillsRequest{
			RootID: topology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)
	afterSnapshot := skillRevisionSnapshot(afterRehydrate)

	if len(afterSnapshot) != len(beforeRehydrate) {
		t.Fatalf(
			"built-in Skill count after idempotent hydration=%d, want %d",
			len(afterSnapshot),
			len(beforeRehydrate),
		)
	}
	for ref, revision := range beforeRehydrate {
		if afterSnapshot[ref] != revision {
			t.Fatalf(
				"built-in Skill %+v revision after idempotent hydration=%d, want %d",
				ref,
				afterSnapshot[ref],
				revision,
			)
		}
	}

	baselineAgain := fixture.ensureUserBaseline(t)
	if baselineAgain.Artifact.Ref() != baseline.Artifact.Ref() {
		t.Fatalf(
			"baseline ref after idempotent ensure=%+v, want %+v",
			baselineAgain.Artifact.Ref(),
			baseline.Artifact.Ref(),
		)
	}
	if baselineAgain.Artifact.Revision != baseline.Artifact.Revision {
		t.Fatalf(
			"baseline revision after idempotent ensure=%d, want %d",
			baselineAgain.Artifact.Revision,
			baseline.Artifact.Revision,
		)
	}
}
