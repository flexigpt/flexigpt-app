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
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/consumerapi"
)

type skillWorkflowFixture struct {
	store *compose.Store
	api   *skillConsumerAPI.API

	bootstrap       *install.Bootstrap
	baselineEnsurer skillConsumerAPI.BaselineEnsurer
}

func newSkillWorkflowFixture(t *testing.T) *skillWorkflowFixture {
	t.Helper()

	registry, err := registration.NewLLMInterpretationRegistry()
	requireNoError(t, err)
	schemaCodecs, err := registration.LLMDeclarationSchemaCodecs()
	requireNoError(t, err)
	canonicalDecoders, err := registration.LLMCanonicalDeclarationDecoders(
		registry,
	)
	requireNoError(t, err)
	sourceFormatDecoders, err := registration.LLMSourceFormatDecoders(
		registry,
	)
	requireNoError(t, err)
	locatorFactories, err := registration.LLMPathLocatorFactories(
		registry,
	)
	requireNoError(t, err)
	//nolint:gocritic // Ok assign.
	decoders := append(canonicalDecoders, sourceFormatDecoders...)

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

	llm, err := llmartifactory.Open(t.Context(), llmartifactory.Config{
		Store:            store,
		SchemaCodecs:     schemaCodecs,
		Decoders:         decoders,
		Interpretations:  registry,
		LocatorFactories: locatorFactories,
		Scope: composition.ScopeBinding{
			BuiltinRoot: topology.BuiltinRootID(),
		},
	})
	requireNoError(t, err)
	t.Cleanup(func() {
		if err := llm.Close(); err != nil {
			t.Errorf("close Skill workflow LLM Artifactory: %v", err)
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
		skillConsumerAPI.WithCompositionResolver(llm.Composition()),
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
) plugin.PluginView {
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
	collectionValue plugin.PluginView,
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
			Plugin:                     collectionValue.Artifact.Ref(),
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
