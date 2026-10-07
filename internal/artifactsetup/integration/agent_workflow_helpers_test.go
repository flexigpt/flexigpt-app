package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/agentcatalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/workspacecatalog/defaultpolicy"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/compose"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/llmsupport"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coredecoder "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/decoder"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	textAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text"
)

const workflowDependencySourceID sourceModel.SourceID = "0192c4c0-00f0-7000-8000-000000000001"

type workflowHarness struct {
	store *compose.Store
	api   *agentAPI.Service
	texts *textAPI.Service

	dependencyDirectory     string
	dependencySourceCreated bool
	dependencyFiles         map[workflowDependency]spec.Locator
	nextDependencyFile      int

	agentBootstrap *installFlow.Bootstrap
	agentInstaller *installFlow.CatalogInstaller
}

type workflowDependency struct {
	Type   declaration.Type
	Name   spec.LogicalName
	Insert declaration.InsertTarget
}

// workflowDirectCapabilityProvider models an application-supplied capability
// for Model and Tool references. It deliberately has no fabricated Artifact,
// Source, Definition, or local-state identity.
type workflowDirectCapabilityProvider struct{}

func (workflowDirectCapabilityProvider) ProviderIdentity() string {
	return "workflowtest"
}

func (p workflowDirectCapabilityProvider) ResolveDirectCapability(
	ctx context.Context,
	request composition.DirectCapabilityRequest,
) (composition.CapabilityTarget, bool, error) {
	if err := request.RootID.Validate(); err != nil {
		return composition.CapabilityTarget{}, false, err
	}
	if err := request.Name.Validate(); err != nil {
		return composition.CapabilityTarget{}, false, err
	}
	if err := request.Scope.Validate(); err != nil {
		return composition.CapabilityTarget{}, false, err
	}

	switch request.Type {
	case declaration.TypeModel, declaration.TypeTool:
	default:
		return composition.CapabilityTarget{}, false, nil
	}

	target := composition.CapabilityTarget{
		Form:             composition.TargetFormDirect,
		Type:             request.Type,
		Name:             request.Name,
		Provenance:       composition.TargetProvenanceDirectCapability,
		ProviderIdentity: p.ProviderIdentity(),
		ProviderLocalID:  "v1." + string(request.Name),
		Evidence: cryptoutil.DigestBytes([]byte(
			string(request.RootID) + "\x00" +
				string(request.Type) + "\x00" +
				string(request.Name),
		)),
	}
	if err := target.Validate(); err != nil {
		return composition.CapabilityTarget{}, false, err
	}
	return target, true, nil
}

func workflowDirectCapabilities() []composition.DirectCapabilityProvider {
	return []composition.DirectCapabilityProvider{
		workflowDirectCapabilityProvider{},
	}
}

func newWorkflowHarness(
	t *testing.T,
) *workflowHarness {
	t.Helper()

	ctx := t.Context()
	requireNoError(t, topology.ValidateApplicationTopology())

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
	workspaceFS, err := artifactbuiltin.EmbeddedWorkspacePackages()
	requireNoError(t, err)

	store, err := local.Open(
		ctx,
		local.Config{
			BaseDirectory: t.TempDir(),
			EmbeddedProviders: map[string]iofs.ProviderRegistration{
				defaultpolicy.ProviderKey: {
					Filesystem: workspaceFS,
				},
			},
			SchemaCodecs: schemaCodecs,
			Decoders:     decoders,

			ProtectedRootIDs: topology.ProtectedRootIDs(),
			RetainedRoots:    topology.RetainedRootDrafts(),
		},
	)
	requireNoError(t, err)
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close workflow Artifact Store: %v", closeErr)
		}
	})

	llm, err := llmartifactory.Open(llmartifactory.Config{
		Artifacts:        store.Artifacts,
		Catalog:          store.Catalog,
		Resources:        store.Resources,
		Interpretations:  registry,
		LocatorFactories: locatorFactories,
		Scope: composition.ScopeBinding{
			BuiltinRoot: topology.BuiltinRootID(),
		},
		DirectCapabilities: workflowDirectCapabilities(),
	})
	requireNoError(t, err)

	agentSupport, err := llmsupport.Agent()
	requireNoError(t, err)

	api, err := agentAPI.New(
		store.Sources,
		store.Refresh,
		store.Artifacts,
		store.Catalog,
		store.Resources,
		store.ManagedPackages,
		store.Protection,
		store.Definitions,
		agentAPI.WithCompositionResolver(llm.Composition()),
		agentAPI.WithDeclarationInterpretations(registry),
		agentAPI.WithSupport(agentSupport),
	)
	requireNoError(t, err)

	texts, err := textAPI.New(
		store.Resources,
		store.Protection,
	)
	requireNoError(t, err)

	return &workflowHarness{
		store:           store,
		api:             api,
		texts:           texts,
		dependencyFiles: make(map[workflowDependency]spec.Locator),
	}
}

func (h *workflowHarness) installBundledAgents(
	t *testing.T,
) *installFlow.CatalogInstaller {
	t.Helper()

	if h.agentBootstrap == nil {
		installer, err := agentcatalog.NewInstaller(
			agentcatalog.InstallerDependencies{
				Hydrator: h.store.Topology,
			},
		)
		requireNoError(t, err)

		bootstrap, err := installFlow.NewBootstrap(
			topology.BuiltinTopologyDeclaration(),
			h.store.Topology,
			installer,
		)
		requireNoError(t, err)

		h.agentBootstrap = bootstrap
		h.agentInstaller = installer
	}
	ctx := root.WithInstallerPrivilege(t.Context())

	requireNoError(t, h.agentBootstrap.Ensure(ctx))
	h.addMissingBuiltinDependencies(t, ctx)
	return h.agentInstaller
}

func (h *workflowHarness) ensureBundledAgents(
	t *testing.T,
) {
	t.Helper()

	if h.agentBootstrap == nil {
		h.installBundledAgents(t)
		return
	}

	ctx := root.WithInstallerPrivilege(t.Context())
	requireNoError(t, h.agentBootstrap.Ensure(ctx))
	h.addMissingBuiltinDependencies(t, ctx)
}

func (h *workflowHarness) listAgentsForManagement(
	ctx context.Context,
) ([]agentAPI.AgentListItem, error) {
	roots, err := h.store.Roots.List(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(roots, func(left, right int) bool {
		return roots[left].ID < roots[right].ID
	})

	output := make([]agentAPI.AgentListItem, 0)
	for _, rootValue := range roots {
		values, err := h.api.ListAgents(ctx, agentAPI.ListAgentsRequest{
			RootID: rootValue.ID,
		})
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].Ref.RootID != output[right].Ref.RootID {
			return output[left].Ref.RootID < output[right].Ref.RootID
		}
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Ref.ArtifactID < output[right].Ref.ArtifactID
	})
	return output, nil
}

func (h *workflowHarness) listAgentImportDestinationsForManagement(
	ctx context.Context,
) ([]agentAPI.AgentImportDestination, error) {
	roots, err := h.store.Roots.List(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(roots, func(left, right int) bool {
		return roots[left].ID < roots[right].ID
	})

	output := make([]agentAPI.AgentImportDestination, 0)
	for _, rootValue := range roots {
		values, err := h.api.ListAgentImportDestinations(ctx, rootValue.ID)
		if err != nil {
			return nil, err
		}
		for index := range values {
			values[index].RootDisplayName = rootValue.DisplayName
		}
		output = append(output, values...)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].RootID != output[right].RootID {
			return output[left].RootID < output[right].RootID
		}
		if output[left].PluginName != output[right].PluginName {
			return output[left].PluginName < output[right].PluginName
		}
		return output[left].Plugin.ArtifactID <
			output[right].Plugin.ArtifactID
	})
	return output, nil
}

// addMissingBuiltinDependencies keeps this package-level workflow test focused
// on Agent Store. It creates real source-backed declarations for unresolved
// cross-family references so Agent package finalization exercises the normal
// resolver rather than a mock.
//
// The application-level bootstrap workflow should separately install the real
// Skill and MCP built-in package families.
func (h *workflowHarness) addMissingBuiltinDependencies(
	t *testing.T,
	ctx context.Context,
) {
	t.Helper()

	for range 8 {
		dependencies, err := h.unresolvedBuiltinDependencies(ctx)
		requireNoError(t, err)
		if len(dependencies) == 0 {
			return
		}

		if !h.publishBuiltinDependencies(t, ctx, dependencies) {
			t.Fatalf(
				"built-in Agent capability closure is still unresolved after publishing dependencies: %#v",
				dependencies,
			)
		}
	}

	t.Fatal("built-in Agent dependency closure did not converge")
}

func (h *workflowHarness) unresolvedBuiltinDependencies(
	ctx context.Context,
) ([]workflowDependency, error) {
	agents, err := h.api.ListAgents(
		ctx,
		agentAPI.ListAgentsRequest{
			RootID: topology.BuiltinRootID(),
		},
	)
	if err != nil {
		return nil, err
	}

	missing := make(map[workflowDependency]struct{})
	for _, agent := range agents {
		plan, err := h.api.ResolveAgentCapabilities(
			ctx,
			agent.Ref,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"resolve bundled Agent %q: %w",
				agent.Name,
				err,
			)
		}

		for _, occurrence := range plan.Occurrences {
			if !occurrence.Required ||
				occurrence.Status == composition.ResolutionAvailable {
				continue
			}
			if occurrence.Name == "" {
				return nil, fmt.Errorf(
					"bundled Agent %q has unresolved unnamed %s capability at %q",
					agent.Name,
					occurrence.Type,
					occurrence.Path,
				)
			}
			switch occurrence.Type {
			case declaration.TypeModel, declaration.TypeTool:
				return nil, fmt.Errorf(
					"built-in %s fallback %q is unavailable at %q",
					occurrence.Type,
					occurrence.Name,
					occurrence.Path,
				)

			case declaration.TypeText:
				missing[workflowDependency{
					Type:   occurrence.Type,
					Name:   occurrence.Name,
					Insert: declaration.InsertInstructions,
				}] = struct{}{}
				missing[workflowDependency{
					Type:   occurrence.Type,
					Name:   occurrence.Name,
					Insert: declaration.InsertUserMessage,
				}] = struct{}{}

			default:
				missing[workflowDependency{
					Type: occurrence.Type,
					Name: occurrence.Name,
				}] = struct{}{}
			}
		}
	}

	output := make([]workflowDependency, 0, len(missing))
	for value := range missing {
		output = append(output, value)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].Type != output[right].Type {
			return output[left].Type < output[right].Type
		}
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Insert < output[right].Insert
	})
	return output, nil
}

func (h *workflowHarness) publishBuiltinDependencies(
	t *testing.T,
	ctx context.Context,
	dependencies []workflowDependency,
) bool {
	t.Helper()

	if h.dependencyDirectory == "" {
		h.dependencyDirectory = t.TempDir()
	}

	skillSupport, err := llmsupport.Skill()
	requireNoError(t, err)

	wrote := false
	for _, dependency := range dependencies {
		if _, found := h.dependencyFiles[dependency]; found {
			continue
		}

		locator, content, err := workflowDependencySourceFile(
			dependency,
			h.nextDependencyFile,
			skillSupport.Documents,
		)
		requireNoError(t, err)

		h.nextDependencyFile++

		location := filepath.Join(
			h.dependencyDirectory,
			string(locator),
		)
		requireNoError(
			t,
			os.MkdirAll(filepath.Dir(location), 0o700),
		)
		requireNoError(
			t,
			os.WriteFile(
				location,
				content,
				0o600,
			),
		)

		h.dependencyFiles[dependency] = locator
		wrote = true
	}

	if !wrote {
		return false
	}

	if !h.dependencySourceCreated {
		config, err := json.Marshal(map[string]string{
			"rootPath": h.dependencyDirectory,
		})
		requireNoError(t, err)

		discovery, err := workflowDependencyDiscovery(
			skillSupport.Documents,
		)
		requireNoError(t, err)

		_, err = h.store.Sources.Create(
			ctx,
			topology.BuiltinRootID(),
			sourceModel.Draft{
				ID:          workflowDependencySourceID,
				StorageKey:  "agent-test-dependencies",
				Kind:        fsdir.Kind,
				DisplayName: "Agent workflow test dependencies",
				Enabled:     true,
				Config:      json.RawMessage(config),
				Discovery:   discovery,
			},
		)
		requireNoError(t, err)
		h.dependencySourceCreated = true
	}

	refreshed, err := h.store.Refresh.RefreshSource(
		ctx,
		topology.BuiltinRootID(),
		workflowDependencySourceID,
	)
	requireNoError(t, err)
	for _, value := range refreshed.Diagnostics {
		if value.Severity != diagnostic.SeverityError {
			continue
		}
		t.Fatalf(
			"workflow dependency Source has an error diagnostic: %#v",
			refreshed.Diagnostics,
		)
	}

	return true
}

// workflowDependencyDiscovery deliberately includes direct-root patterns as
// well as recursive patterns. A globstar descendant pattern alone must not be
// relied on to include a Source-root file.
//
// Skills are emitted as actual configured Skill package documents rather than
// canonical YAML declarations with a synthetic dangling locator. This keeps
// the Agent workflow fixture aligned with the real Skill source-format path.
func workflowDependencyDiscovery(
	documents support.Documents,
) (sourceModel.DiscoverySpec, error) {
	if err := documents.Validate(); err != nil {
		return sourceModel.DiscoverySpec{}, err
	}

	patterns := []string{
		"*.yaml",
		"**/*.yaml",
	}
	for _, document := range documents.Files {
		patterns = append(
			patterns,
			string(document),
			"**/"+string(document),
		)
	}

	decoderIDs := []spec.DecoderID{
		coredecoder.YAMLDecoderID,
	}
	if documents.Default.DecoderID != coredecoder.YAMLDecoderID {
		decoderIDs = append(
			decoderIDs,
			documents.Default.DecoderID,
		)
	}

	value := sourceModel.DiscoverySpec{
		DirectoryRoots: []sourceModel.DirectoryRoot{{
			Root:            ".",
			Recursive:       true,
			IncludePatterns: patterns,
		}},
		AllowedDecoderIDs: decoderIDs,
		Authoritative:     true,
	}.Normalized()
	return value, value.Validate()
}

func workflowDependencySourceFile(
	dependency workflowDependency,
	index int,
	documents support.Documents,
) (spec.Locator, []byte, error) {
	if dependency.Type == declaration.TypeSkill {
		locator := spec.Locator(path.Join(
			string(dependency.Name),
			string(documents.Default.Locator),
		))
		if err := locator.ValidatePortable(false); err != nil {
			return "", nil, err
		}
		content, err := workflowDependencySkillDocument(dependency)
		if err != nil {
			return "", nil, err
		}
		return locator, content, nil
	}

	locator := spec.Locator(fmt.Sprintf(
		"dependency-%04d.yaml",
		index,
	))
	if err := locator.ValidatePortable(false); err != nil {
		return "", nil, err
	}
	content, err := workflowDependencyDocument(dependency)
	if err != nil {
		return "", nil, err
	}
	return locator, content, nil
}

func workflowDependencySkillDocument(
	dependency workflowDependency,
) ([]byte, error) {
	if err := dependency.Name.Validate(); err != nil {
		return nil, err
	}
	return fmt.Appendf(
		nil,
		"---\n"+
			"name: %s\n"+
			"description: %s\n"+
			"insert: %s\n"+
			"---\n\n"+
			"# %s\n\n"+
			"Workflow fixture dependency Skill.\n",
		workflowYAMLQuote(string(dependency.Name)),
		workflowYAMLQuote("Workflow fixture dependency Skill."),
		workflowYAMLQuote(string(declaration.InsertInstructions)),
		dependency.Name,
	), nil
}

func workflowDependencyDocument(
	dependency workflowDependency,
) ([]byte, error) {
	name := workflowYAMLQuote(string(dependency.Name))

	switch dependency.Type {
	case declaration.TypeText:
		if err := dependency.Insert.Validate(); err != nil {
			return nil, err
		}
		return fmt.Appendf(nil,
			"type: %s\nname: %s\ninsert: %s\ncontent: %s\n",
			workflowYAMLQuote(string(declaration.TypeText)),
			name,
			workflowYAMLQuote(string(dependency.Insert)),
			workflowYAMLQuote("workflow dependency text"),
		), nil

	case declaration.TypeSkill:
		return nil, fmt.Errorf(
			"workflow dependency Skill %q must be published as a Skill package document",
			dependency.Name,
		)

	case declaration.TypeMCP:
		return fmt.Appendf(nil,
			"type: %s\nname: %s\ntransport: %s\ncommand: %s\n",
			workflowYAMLQuote(string(declaration.TypeMCP)),
			name,
			workflowYAMLQuote("stdio"),
			workflowYAMLQuote("workflow-dependency"),
		), nil

	case declaration.TypeMCPPolicy:
		return fmt.Appendf(nil,
			"type: %s\nname: %s\n",
			workflowYAMLQuote(string(declaration.TypeMCPPolicy)),
			name,
		), nil

	case declaration.TypePlugin:
		return fmt.Appendf(nil,
			"type: %s\nname: %s\n",
			workflowYAMLQuote(string(declaration.TypePlugin)),
			name,
		), nil

	case declaration.TypeAgent:
		return fmt.Appendf(nil,
			"type: %s\nname: %s\n",
			workflowYAMLQuote(string(declaration.TypeAgent)),
			name,
		), nil

	case declaration.TypeTeam:
		return fmt.Appendf(nil,
			"type: %s\nname: %s\n",
			workflowYAMLQuote(string(declaration.TypeTeam)),
			name,
		), nil

	case declaration.TypeLoop:
		bodyName, err := declaration.DeriveNestedLogicalName(
			dependency.Name,
			"body",
		)
		if err != nil {
			return nil, err
		}
		return fmt.Appendf(nil,
			"type: %s\nname: %s\nbody:\n  type: %s\n  name: %s\n  parameters: {}\n",
			workflowYAMLQuote(string(declaration.TypeLoop)),
			name,
			workflowYAMLQuote(string(declaration.TypeAgent)),
			workflowYAMLQuote(string(bodyName)),
		), nil

	case declaration.TypeWorkflow:
		return fmt.Appendf(nil,
			"type: %s\nname: %s\n",
			workflowYAMLQuote(string(declaration.TypeWorkflow)),
			name,
		), nil

	default:
		return nil, fmt.Errorf(
			"workflow dependency type %q is unsupported",
			dependency.Type,
		)
	}
}

func workflowYAMLQuote(
	value string,
) string {
	return fmt.Sprintf("%q", value)
}

func requireNoError(
	t *testing.T,
	err error,
) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireErrorIs(
	t *testing.T,
	err error,
	target error,
) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("error = %v, want errors.Is(..., %v)", err, target)
	}
}

func requireSameAgentRevisions(
	t *testing.T,
	before []agentAPI.AgentListItem,
	after []agentAPI.AgentListItem,
) {
	t.Helper()

	if len(before) != len(after) {
		t.Fatalf(
			"Agent count changed after idempotent ensure: before=%d after=%d",
			len(before),
			len(after),
		)
	}

	revisions := make(map[artifactModel.ArtifactRef]uint64, len(before))
	for _, value := range before {
		revisions[value.Ref] = value.Revision
	}

	for _, value := range after {
		revision, found := revisions[value.Ref]
		if !found {
			t.Fatalf(
				"unexpected Agent %q after idempotent ensure",
				value.Ref,
			)
		}
		if value.Revision != revision {
			t.Fatalf(
				"Agent %q revision changed after idempotent ensure: got %d want %d",
				value.Ref,
				value.Revision,
				revision,
			)
		}
	}
}
