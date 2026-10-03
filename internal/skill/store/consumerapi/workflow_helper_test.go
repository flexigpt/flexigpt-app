package consumerapi_test

import (
	"fmt"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/providercanonical"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	skillBuiltin "github.com/flexigpt/flexigpt-app/internal/skill/store/builtin"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
	skillProviderAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/providerapi"
)

type skillWorkflowFixture struct {
	store *local.Store
	api   *skillConsumerAPI.API

	bootstrap       *builtin.BootstrapRegistry
	baselineEnsurer skillConsumerAPI.BaselineEnsurer
}

func newSkillWorkflowFixture(t *testing.T) *skillWorkflowFixture {
	t.Helper()

	canonicalProvider, err := providercanonical.New()
	requireNoError(t, err)

	skillProvider, err := skillProviderAPI.NewProvider()
	requireNoError(t, err)

	store, err := local.Open(
		t.Context(),
		local.Config{
			BaseDirectory: t.TempDir(),
			Providers: []provider.Provider{
				canonicalProvider,
				skillProvider,
			},
			ProtectedRootIDs: documentTopology.ProtectedRootIDs(),
			RetainedRoots:    documentTopology.RetainedRootDrafts(),
		},
	)
	requireNoError(t, err)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close Skill workflow Artifact Store: %v", err)
		}
	})

	api, err := skillConsumerAPI.New(
		store.Sources,
		store.Refresh,
		store.Artifacts,
		store.Resources,
		store.ManagedPackages,
		store.Protection,
		store.Catalog,
		store.Definitions,
		skillConsumerAPI.WithLocatorResolvers(
			store.LocatorResolvers,
		),
	)
	requireNoError(t, err)

	requireNoError(t, err)

	installer, err := skillBuiltin.NewInstaller(
		skillBuiltin.InstallerDependencies{
			Hydrator: store.Topology,
		},
	)
	requireNoError(t, err)

	bootstrap, err := builtin.NewDefaultBootstrapRegistry(
		store.Topology,
		store.Topology,
	)
	requireNoError(t, err)
	requireNoError(t, bootstrap.Register(installer))

	baselineEnsurer, err := skillConsumerAPI.NewBaselineEnsurer(api)
	requireNoError(t, err)

	return &skillWorkflowFixture{
		store:           store,
		api:             api,
		bootstrap:       bootstrap,
		baselineEnsurer: baselineEnsurer,
	}
}

func (f *skillWorkflowFixture) bootstrapBuiltins(t *testing.T) {
	t.Helper()
	requireNoError(t, f.bootstrap.Ensure(t.Context()))
}

func (f *skillWorkflowFixture) ensureUserBaseline(
	t *testing.T,
) collection.CollectionView {
	t.Helper()

	value, err := f.baselineEnsurer.EnsureSkillBaselineCollection(
		t.Context(),
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	return value
}

func createManagedSkillInCollection(
	t *testing.T,
	api *skillConsumerAPI.API,
	collectionValue collection.CollectionView,
	name string,
	description string,
	body string,
	checklist string,
) (r skillConsumerAPI.ManagedSkillCreateResult, doc []byte) {
	t.Helper()

	document := workflowSkillMarkdown(name, description, body)
	result, err := api.CreateManagedSkill(
		t.Context(),
		skillConsumerAPI.ManagedSkillCreateRequest{
			Collection:                 collectionValue.Artifact.Ref(),
			ExpectedCollectionRevision: collectionValue.Artifact.Revision,
			SkillName:                  name,
			SKILLMD:                    document,
			Files:                      managedSkillFiles(document, checklist),
			Enabled:                    true,
		},
	)
	requireNoError(t, err)
	return result, document
}

func managedSkillFiles(
	document []byte,
	checklist string,
) []source.ManagedPackageFile {
	return []source.ManagedPackageFile{
		{
			Locator: spec.Locator("SKILL.md"),
			Content: append([]byte(nil), document...),
		},
		{
			Locator: spec.Locator("references/checklist.md"),
			Content: []byte(checklist),
		},
	}
}

func workflowSkillMarkdown(
	name string,
	description string,
	body string,
) []byte {
	return fmt.Appendf(nil,
		"---\n"+
			"name: %s\n"+
			"description: %s\n"+
			"insert: instructions\n"+
			"---\n\n"+
			"# %s\n\n"+
			"%s\n",
		name,
		description,
		name,
		body,
	)
}

func findSkillByName(
	values []skillConsumerAPI.SkillListItem,
	name string,
) (skillConsumerAPI.SkillListItem, bool) {
	for _, value := range values {
		if string(value.Name) == name {
			return value, true
		}
	}
	return skillConsumerAPI.SkillListItem{}, false
}

func findCollectionByName(
	values []collection.ListItem,
	name string,
) (collection.ListItem, bool) {
	for _, value := range values {
		if string(value.Name) == name {
			return value, true
		}
	}
	return collection.ListItem{}, false
}

func skillRevisionSnapshot(
	values []skillConsumerAPI.SkillListItem,
) map[artifact.ArtifactRef]uint64 {
	output := make(map[artifact.ArtifactRef]uint64, len(values))
	for _, value := range values {
		output[value.Ref] = value.Revision
	}
	return output
}

func requireNoError(
	t *testing.T,
	err error,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
