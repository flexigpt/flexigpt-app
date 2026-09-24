package consumerapi_test

import (
	"errors"
	"testing"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
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
		if value.ID == documentTopology.UserRootID() {
			userRootFound = true
			break
		}
	}
	if !userRootFound {
		t.Fatalf(
			"retained user Root %q was not created by composition startup",
			documentTopology.UserRootID(),
		)
	}

	_, err = fixture.store.Roots.Get(
		ctx,
		documentTopology.BuiltinRootID(),
	)
	if !errors.Is(err, basespec.ErrRootNotFound) {
		t.Fatalf(
			"built-in Root before hydration error=%v, want ErrRootNotFound",
			err,
		)
	}

	initialUserSkills, err := fixture.api.ListSkills(
		ctx,
		documentTopology.UserRootID(),
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
		documentTopology.BuiltinRootID(),
		documentTopology.BuiltinPackageSourceID(),
	)
	requireNoError(t, err)
	if builtinSource.Kind != source.SourceKindManagedDirectory {
		t.Fatalf(
			"built-in Source kind=%q, want %q",
			builtinSource.Kind,
			source.SourceKindManagedDirectory,
		)
	}
	if !builtinSource.Enabled {
		t.Fatal("built-in package Source is disabled after hydration")
	}

	builtinSkills, err := fixture.api.ListSkills(
		ctx,
		documentTopology.BuiltinRootID(),
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
	if markdownOutput.State != artifact.StateAvailable {
		t.Fatalf(
			"markdown-output state=%q, want %q",
			markdownOutput.State,
			artifact.StateAvailable,
		)
	}
	if !markdownOutput.Enabled {
		t.Fatal("markdown-output is disabled after built-in hydration")
	}

	builtinCollections, err := fixture.api.ListSkillCollections(
		ctx,
		documentTopology.BuiltinRootID(),
	)
	requireNoError(t, err)
	if len(builtinCollections) == 0 {
		t.Fatal("built-in hydration produced no visible Skill Collections")
	}
	for _, value := range builtinCollections {
		if value.Editable {
			t.Fatalf(
				"built-in Collection %q is unexpectedly editable",
				value.Name,
			)
		}
		if value.Deletable {
			t.Fatalf(
				"built-in Collection %q is unexpectedly deletable",
				value.Name,
			)
		}
	}

	baseline := fixture.ensureUserBaseline(t)
	if baseline.Name != collection.SkillBaselineCollectionName {
		t.Fatalf(
			"baseline name=%q, want %q",
			baseline.Name,
			collection.SkillBaselineCollectionName,
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

	userCollections, err := fixture.api.ListSkillCollections(
		ctx,
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	listedBaseline, found := findCollectionByName(
		userCollections,
		string(collection.SkillBaselineCollectionName),
	)
	if !found {
		t.Fatal("user Skill baseline is absent from Collection listing")
	}
	if listedBaseline.Artifact.Ref() != baseline.Artifact.Ref() {
		t.Fatalf(
			"listed baseline ref=%+v, want %+v",
			listedBaseline.Artifact.Ref(),
			baseline.Artifact.Ref(),
		)
	}

	err = fixture.api.DeleteSkillCollection(
		ctx,
		collection.DeleteRequest{
			Collection:       baseline.Artifact.Ref(),
			ExpectedRevision: baseline.Artifact.Revision,
		},
	)
	if !errors.Is(err, basespec.ErrProtected) {
		t.Fatalf(
			"delete baseline error=%v, want ErrProtected",
			err,
		)
	}

	beforeRehydrate := skillRevisionSnapshot(builtinSkills)

	fixture.bootstrapBuiltins(t)

	afterRehydrate, err := fixture.api.ListSkills(
		ctx,
		documentTopology.BuiltinRootID(),
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
