package integration

import (
	"fmt"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/skillcatalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/compose"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill"
)

type skillWorkflowFixture struct {
	store *compose.Store
	api   *skillAPI.Service

	bootstrap       *installFlow.Bootstrap
	baselineEnsurer skillAPI.BaselineEnsurer
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

	api, err := skillAPI.New(
		store.Sources,
		store.Refresh,
		store.Artifacts,
		store.Resources,
		store.ManagedPackages,
		store.Protection,
		store.Catalog,
		store.Definitions,
		skillAPI.WithCompositionResolver(llm.Composition()),
	)
	requireNoError(t, err)

	installer, err := skillcatalog.NewInstaller(
		skillcatalog.InstallerDependencies{
			Hydrator: store.Topology,
		},
	)
	requireNoError(t, err)

	bootstrap, err := installFlow.NewBootstrap(
		topology.BuiltinTopologyDeclaration(),
		store.Topology,
		installer,
	)
	requireNoError(t, err)

	baselineEnsurer, err := skillAPI.NewBaselineEnsurer(api)
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
) pluginAPI.PluginView {
	t.Helper()

	value, err := f.baselineEnsurer.EnsureSkillBaselinePlugin(
		t.Context(),
		topology.UserRootID(),
	)
	requireNoError(t, err)
	return value
}

func createManagedSkillInPlugin(
	t *testing.T,
	api *skillAPI.Service,
	pluginValue pluginAPI.PluginView,
	name string,
	description string,
	body string,
	checklist string,
) (r skillAPI.ManagedSkillCreateResult, doc []byte) {
	t.Helper()

	document := workflowSkillMarkdown(name, description, body)
	result, err := api.CreateManagedSkill(
		t.Context(),
		skillAPI.ManagedSkillCreateRequest{
			Plugin:                 pluginValue.Artifact.Ref(),
			ExpectedPluginRevision: pluginValue.Artifact.Revision,
			SkillName:              name,
			SKILLMD:                document,
			Files:                  managedSkillFiles(document, checklist),
			Enabled:                true,
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
	values []skillAPI.SkillListItem,
	name string,
) (skillAPI.SkillListItem, bool) {
	for _, value := range values {
		if string(value.Name) == name {
			return value, true
		}
	}
	return skillAPI.SkillListItem{}, false
}

func findPluginByName(
	values []pluginAPI.ListItem,
	name string,
) (pluginAPI.ListItem, bool) {
	for _, value := range values {
		if string(value.Name) == name {
			return value, true
		}
	}
	return pluginAPI.ListItem{}, false
}

func skillRevisionSnapshot(
	values []skillAPI.SkillListItem,
) map[artifactModel.ArtifactRef]uint64 {
	output := make(map[artifactModel.ArtifactRef]uint64, len(values))
	for _, value := range values {
		output[value.Ref] = value.Revision
	}
	return output
}
