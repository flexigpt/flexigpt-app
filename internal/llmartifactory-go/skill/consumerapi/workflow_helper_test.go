package consumerapi_test

import (
	"fmt"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/skillcatalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/compose"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/registration/canonical"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/consumerapi"
	skillProviderAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/providerapi"
)

type skillWorkflowFixture struct {
	store *compose.Store
	api   *skillConsumerAPI.API

	bootstrap       *install.Bootstrap
	baselineEnsurer skillConsumerAPI.BaselineEnsurer
}

func newSkillWorkflowFixture(t *testing.T) *skillWorkflowFixture {
	t.Helper()

	canonicalRegistration, err := canonical.NewRegistration()
	requireNoError(t, err)

	skillRegistration, err := skillProviderAPI.NewRegistration()
	requireNoError(t, err)

	locatorRegistry, err := locator.NewRegistry(
		canonicalRegistration.LocatorFactories()...,
	)
	requireNoError(t, err)

	schemaCodecs := canonicalRegistration.SchemaCodecs()

	decoders := canonicalRegistration.Decoders()
	decoders = append(
		decoders,
		skillRegistration.Decoders()...,
	)

	store, err := local.Open(
		t.Context(),
		local.Config{
			BaseDirectory: t.TempDir(),
			SchemaCodecs:  schemaCodecs,
			Decoders:      decoders,

			ProtectedRootIDs: topology.ProtectedRootIDs(),
			RetainedRoots:    topology.RetainedRootDrafts(),
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
			locatorRegistry.Factories(),
		),
	)
	requireNoError(t, err)

	installer, err := skillcatalog.NewInstaller(
		skillcatalog.InstallerDependencies{
			Hydrator: store.Topology,
		},
	)
	requireNoError(t, err)

	bootstrap, err := install.NewBootstrap(
		topology.BuiltinTopologyDeclaration(),
		store.Topology,
		installer,
	)
	requireNoError(t, err)

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
	requireNoError(t, f.bootstrap.Ensure(root.WithInstallerPrivilege(t.Context())))
}

func (f *skillWorkflowFixture) ensureUserBaseline(
	t *testing.T,
) plugin.CollectionView {
	t.Helper()

	value, err := f.baselineEnsurer.EnsureSkillBaselineCollection(
		t.Context(),
		topology.UserRootID(),
	)
	requireNoError(t, err)
	return value
}

func createManagedSkillInCollection(
	t *testing.T,
	api *skillConsumerAPI.API,
	collectionValue plugin.CollectionView,
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
) []managedpackageModel.ManagedPackageFile {
	return []managedpackageModel.ManagedPackageFile{
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
	values []plugin.ListItem,
	name string,
) (plugin.ListItem, bool) {
	for _, value := range values {
		if string(value.Name) == name {
			return value, true
		}
	}
	return plugin.ListItem{}, false
}

func skillRevisionSnapshot(
	values []skillConsumerAPI.SkillListItem,
) map[artifactModel.ArtifactRef]uint64 {
	output := make(map[artifactModel.ArtifactRef]uint64, len(values))
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
