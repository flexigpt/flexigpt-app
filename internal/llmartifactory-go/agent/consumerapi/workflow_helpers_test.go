package consumerapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/agentcatalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/workspacecatalog/defaultpolicy"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/compose"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/decoder"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/registration/canonical"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/registration/markdown"
	mcpProviderAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/providerapi"
	skillProviderAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/providerapi"
)

const workflowDependencySourceID sourceModel.SourceID = "0192c4c0-00f0-7000-8000-000000000001"

type workflowHarness struct {
	store *compose.Store
	api   *agentConsumerAPI.API

	dependencyDirectory     string
	dependencySourceCreated bool
	dependencyFiles         map[workflowDependency]spec.Locator
	nextDependencyFile      int

	agentBootstrap *install.Bootstrap
	agentInstaller *agentcatalog.Installer
}

type workflowDependency struct {
	Type   declaration.Type
	Name   spec.LogicalName
	Insert declaration.InsertTarget
}

// workflowMappedFallback models the application-composition fallback route
// used by Tool and Model references. Built-in Agent packages are allowed to
// depend on these mapped runtime targets without requiring source-backed
// Tool or Model Artifacts.
type workflowMappedFallback struct {
	declarationType declaration.Type
}

func (f workflowMappedFallback) ResolveFallback(
	ctx context.Context,
	request composition.FallbackRequest,
) (composition.FallbackTarget, bool, error) {
	if ctx == nil {
		return composition.FallbackTarget{}, false, fmt.Errorf(
			"%w: workflow fallback context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return composition.FallbackTarget{}, false, err
	}
	if request.Type != f.declarationType {
		return composition.FallbackTarget{}, false, nil
	}
	if err := request.Name.Validate(); err != nil {
		return composition.FallbackTarget{}, false, err
	}

	return composition.FallbackTarget{
		Mapped: &composition.MappedTarget{
			Provider:   "workflow-test",
			Identifier: "v1." + string(request.Name),
			Type:       request.Type,
			Name:       request.Name,
			Builtin:    true,
		},
	}, true, nil
}

func workflowFallbackProviders() map[declaration.Type]composition.FallbackProvider {
	return map[declaration.Type]composition.FallbackProvider{
		declaration.TypeModel: workflowMappedFallback{
			declarationType: declaration.TypeModel,
		},
		declaration.TypeTool: workflowMappedFallback{
			declarationType: declaration.TypeTool,
		},
	}
}

func newWorkflowHarness(
	t *testing.T,
) *workflowHarness {
	t.Helper()

	ctx := t.Context()
	requireNoError(t, topology.ValidateApplicationTopology())

	canonicalRegistration, err := canonical.NewRegistration()
	requireNoError(t, err)

	markdownRegistration, err := markdown.NewRegistration()
	requireNoError(t, err)

	skillRegistration, err := skillProviderAPI.NewRegistration()
	requireNoError(t, err)

	mcpRegistration, err := mcpProviderAPI.NewRegistration()
	requireNoError(t, err)

	locatorRegistry, err := locator.NewRegistry(
		canonicalRegistration.LocatorFactories()...,
	)
	requireNoError(t, err)

	schemaCodecs := canonicalRegistration.SchemaCodecs()

	decoders := canonicalRegistration.Decoders()
	decoders = append(
		decoders,
		markdownRegistration.Decoders()...,
	)
	decoders = append(
		decoders,
		skillRegistration.Decoders()...,
	)
	decoders = append(
		decoders,
		mcpRegistration.Decoders()...,
	)
	workspaceFS, err := artifactbuiltin.EmbeddedWorkspacePackages()
	requireNoError(t, err)

	store, err := local.Open(
		ctx,
		local.Config{
			BaseDirectory: t.TempDir(),
			EmbeddedProviders: map[string]fs.FS{
				defaultpolicy.ProviderKey: workspaceFS,
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

	api, err := agentConsumerAPI.New(
		store.Sources,
		store.Refresh,
		store.Artifacts,
		store.Catalog,
		store.Resources,
		store.ManagedPackages,
		store.Protection,
		store.Definitions,
		agentConsumerAPI.WithRoots(store.Roots),
		agentConsumerAPI.WithFallbackProviders(
			workflowFallbackProviders(),
		),
		agentConsumerAPI.WithLocatorResolvers(
			locatorRegistry.Factories(),
		),
	)
	requireNoError(t, err)

	return &workflowHarness{
		store:           store,
		api:             api,
		dependencyFiles: make(map[workflowDependency]spec.Locator),
	}
}

func (h *workflowHarness) installBundledAgents(
	t *testing.T,
) *agentcatalog.Installer {
	t.Helper()

	if h.agentBootstrap == nil {
		installer, err := agentcatalog.NewInstaller(
			agentcatalog.InstallerDependencies{
				Hydrator: h.store.Topology,
			},
		)
		requireNoError(t, err)

		bootstrap, err := install.NewBootstrap(
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
		agentConsumerAPI.ListAgentsRequest{
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

	wrote := false
	for _, dependency := range dependencies {
		if _, found := h.dependencyFiles[dependency]; found {
			continue
		}

		content, err := workflowDependencyDocument(dependency)
		requireNoError(t, err)

		loc := spec.Locator(fmt.Sprintf(
			"dependency-%04d.yaml",
			h.nextDependencyFile,
		))
		h.nextDependencyFile++

		requireNoError(
			t,
			os.WriteFile(
				filepath.Join(
					h.dependencyDirectory,
					string(loc),
				),
				content,
				0o600,
			),
		)

		h.dependencyFiles[dependency] = loc
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

		discovery := sourceModel.DiscoverySpec{
			DirectoryRoots: []sourceModel.DirectoryRoot{{
				Root:            ".",
				Recursive:       true,
				IncludePatterns: []string{"**/*.yaml"},
			}},
			AllowedDecoderIDs: []spec.DecoderID{
				decoder.YAMLDecoderID,
			},
			Authoritative: true,
		}.Normalized()

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
		return fmt.Appendf(nil,
			"type: %s\nname: %s\nlocator: %s\n",
			workflowYAMLQuote(string(declaration.TypeSkill)),
			name,
			workflowYAMLQuote("./placeholder"),
		), nil

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
	before []agentConsumerAPI.AgentListItem,
	after []agentConsumerAPI.AgentListItem,
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
