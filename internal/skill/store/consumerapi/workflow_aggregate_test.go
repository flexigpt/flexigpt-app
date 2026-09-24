package consumerapi_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
)

func TestSkillStoreWorkflowAggregateCatalogFollowsSkillLifecycle(
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
			Name:        "catalog-workflow",
			DisplayName: "Catalog workflow",
			Description: "Skill Collection used to verify aggregate catalog sync.",
		},
	)
	requireNoError(t, err)

	const skillName = "catalog-notes"
	created, _ := createManagedSkillInCollection(
		t,
		fixture.api,
		collectionValue,
		skillName,
		"Prepare catalog-aware release notes.",
		"Use the current verified Skill package.",
		"Check the release catalog.\n",
	)

	catalogSkill, err := fixture.api.GetSkill(
		ctx,
		created.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !catalogSkill.Enabled {
		t.Fatal("managed Skill is disabled before initial catalog synchronization")
	}

	inspection, err := fixture.store.Discovery.InspectSource(
		ctx,
		catalogSkill.RootID,
		catalogSkill.Binding.SourceID,
	)
	requireNoError(t, err)
	if !inspection.IsCurrent() {
		t.Fatalf(
			"managed Skill Source is stale before catalog synchronization: %+v",
			inspection,
		)
	}

	aggregateService, runtimeService := newSkillAggregateService(
		t,
		fixture,
	)

	requireNoError(
		t,
		aggregateService.SyncRootCatalog(
			ctx,
			documentTopology.UserRootID(),
		),
	)

	initial, err := aggregateService.ResolveArtifactSkill(
		ctx,
		created.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !initial.Enabled {
		t.Fatal("aggregate resolved newly created Skill as disabled")
	}
	if initial.Artifact != created.Artifact.Ref() {
		t.Fatalf(
			"aggregate resolved Artifact ref=%+v, want %+v",
			initial.Artifact,
			created.Artifact.Ref(),
		)
	}
	if initial.Definition.Name != skillName {
		t.Fatalf(
			"aggregate SkillDef name=%q, want %q",
			initial.Definition.Name,
			skillName,
		)
	}

	refs, err := aggregateService.ListArtifactSkillRefs(
		ctx,
		skillAggregate.ArtifactSkillFilter{
			AllowArtifacts: []artifact.ArtifactRef{
				created.Artifact.Ref(),
			},
		},
	)
	requireNoError(t, err)
	if len(refs) != 1 || refs[0] != created.Artifact.Ref() {
		t.Fatalf(
			"aggregate listed refs=%+v, want [%+v]",
			refs,
			created.Artifact.Ref(),
		)
	}

	summary, err := aggregateService.DescribeArtifactSkill(
		ctx,
		created.Artifact.Ref(),
	)
	requireNoError(t, err)
	if summary.Artifact != created.Artifact.Ref() {
		t.Fatalf(
			"aggregate summary Artifact ref=%+v, want %+v",
			summary.Artifact,
			created.Artifact.Ref(),
		)
	}
	if !summary.IsEnabled {
		t.Fatal("aggregate summary reports new Skill as disabled")
	}

	prompt, err := aggregateService.GetArtifactSkillsPrompt(
		ctx,
		skillAggregate.ArtifactSkillFilter{
			AllowArtifacts: []artifact.ArtifactRef{
				created.Artifact.Ref(),
			},
		},
	)
	requireNoError(t, err)
	if prompt == "" {
		t.Fatal("aggregate returned an empty prompt for an enabled Skill")
	}
	if !strings.Contains(prompt, skillName) {
		t.Fatalf(
			"aggregate prompt does not mention Skill %q: %q",
			skillName,
			prompt,
		)
	}

	disabled, err := fixture.api.SetSkillEnabled(
		ctx,
		created.Artifact.Ref(),
		created.Artifact.Revision,
		false,
	)
	requireNoError(t, err)
	if disabled.Enabled {
		t.Fatal("Skill remains enabled after disable")
	}

	_, err = aggregateService.ResolveArtifactSkill(ctx, disabled.Ref())
	if !errors.Is(err, basespec.ErrReferenceUnresolved) {
		t.Fatalf(
			"aggregate resolution after disable error=%v, want ErrReferenceUnresolved",
			err,
		)
	}

	runtimeRecords, err := runtimeService.ListAgentSkills(ctx, nil)
	requireNoError(t, err)
	for _, record := range runtimeRecords {
		if record.Def == initial.Definition {
			t.Fatalf(
				"disabled Skill definition %+v remains registered in runtime",
				initial.Definition,
			)
		}
	}

	reenabled, err := fixture.api.SetSkillEnabled(
		ctx,
		disabled.Ref(),
		disabled.Revision,
		true,
	)
	requireNoError(t, err)
	if !reenabled.Enabled {
		t.Fatal("Skill remains disabled after enable")
	}

	active, err := aggregateService.ResolveArtifactSkill(
		ctx,
		reenabled.Ref(),
	)
	requireNoError(t, err)
	if !active.Enabled {
		t.Fatal("aggregate resolved re-enabled Skill as disabled")
	}

	currentCollection, err := fixture.api.GetSkillCollection(
		ctx,
		created.Collection.Artifact.Ref(),
	)
	requireNoError(t, err)

	currentSkill, err := fixture.api.GetSkill(ctx, reenabled.Ref())
	requireNoError(t, err)

	replacementDocument := workflowSkillMarkdown(
		skillName,
		"Prepare catalog-aware release notes with migration details.",
		"Use the replacement Skill package after catalog reconciliation.",
	)
	replaced, err := fixture.api.ReplaceManagedSkill(
		ctx,
		skillConsumerAPI.ManagedSkillReplaceRequest{
			Collection:                 currentCollection.Artifact.Ref(),
			ExpectedCollectionRevision: currentCollection.Artifact.Revision,
			Artifact:                   currentSkill.Ref(),
			ExpectedArtifactRevision:   currentSkill.Revision,
			SkillName:                  skillName,
			SKILLMD:                    replacementDocument,
			Files: managedSkillFiles(
				replacementDocument,
				"Check catalog, migrations, and compatibility.\n",
			),
			Enabled: true,
		},
	)
	requireNoError(t, err)
	if replaced.Artifact.Ref() != created.Artifact.Ref() {
		t.Fatalf(
			"replacement changed Skill Artifact ref from %+v to %+v",
			created.Artifact.Ref(),
			replaced.Artifact.Ref(),
		)
	}

	updated, err := aggregateService.ResolveArtifactSkill(
		ctx,
		replaced.Artifact.Ref(),
	)
	requireNoError(t, err)
	if updated.Version == active.Version {
		t.Fatalf(
			"aggregate Skill runtime revision did not change after replacement: %q",
			updated.Version,
		)
	}
	if !updated.Enabled {
		t.Fatal("aggregate resolved replaced Skill as disabled")
	}

	refs, err = aggregateService.ListArtifactSkillRefs(
		ctx,
		skillAggregate.ArtifactSkillFilter{
			AllowArtifacts: []artifact.ArtifactRef{
				replaced.Artifact.Ref(),
			},
		},
	)
	requireNoError(t, err)
	if len(refs) != 1 || refs[0] != replaced.Artifact.Ref() {
		t.Fatalf(
			"aggregate refs after replacement=%+v, want [%+v]",
			refs,
			replaced.Artifact.Ref(),
		)
	}

	summary, err = aggregateService.DescribeArtifactSkill(
		ctx,
		replaced.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !summary.IsEnabled {
		t.Fatal("aggregate summary reports replaced Skill as disabled")
	}
}

func newSkillAggregateService(
	t *testing.T,
	fixture *skillWorkflowFixture,
) (aggSvc *skillAggregate.Service, runtimeSvc *skillRuntime.Service) {
	t.Helper()

	router, err := skillAggregate.NewArtifactRouter(
		fixture.store.Artifacts,
		fixture.store.Resources,
	)
	requireNoError(t, err)

	catalogSource, err := skillAggregate.NewCatalogSource(router)
	requireNoError(t, err)

	runtimeService, err := skillRuntime.New(
		skillRuntime.WithCatalogSource(catalogSource),
	)
	requireNoError(t, err)

	service, err := skillAggregate.New(router, runtimeService)
	if err != nil {
		_ = runtimeService.Close(context.WithoutCancel(t.Context()))
		t.Fatalf("create Skill aggregate service: %v", err)
	}

	t.Cleanup(func() {
		service.Close()
		if err := runtimeService.Close(
			context.WithoutCancel(t.Context()),
		); err != nil {
			t.Errorf("close Skill runtime service: %v", err)
		}
	})

	return service, runtimeService
}
