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

	agentBuiltin "github.com/flexigpt/flexigpt-app/internal/agent/store/builtin"
	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/agent/store/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/providercanonical"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/providermarkdown"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	mcpProviderAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/providerapi"
	skillProviderAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/workspace/defaultpolicy"
)

const workflowDependencySourceID source.SourceID = "0192c4c0-00f0-7000-8000-000000000001"

type workflowHarness struct {
	store *compositionapi.Store
	api   *agentConsumerAPI.API

	dependencyDirectory     string
	dependencySourceCreated bool
	dependencyFiles         map[workflowDependency]basespec.Locator
	nextDependencyFile      int
}

type workflowDependency struct {
	Type   declaration.Type
	Name   basespec.LogicalName
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
	request resolve.FallbackRequest,
) (resolve.FallbackTarget, bool, error) {
	if ctx == nil {
		return resolve.FallbackTarget{}, false, fmt.Errorf(
			"%w: workflow fallback context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return resolve.FallbackTarget{}, false, err
	}
	if request.Type != f.declarationType {
		return resolve.FallbackTarget{}, false, nil
	}
	if err := request.Name.Validate(); err != nil {
		return resolve.FallbackTarget{}, false, err
	}

	return resolve.FallbackTarget{
		Mapped: &resolve.MappedTarget{
			Provider:   "workflow-test",
			Identifier: "v1." + string(request.Name),
			Type:       request.Type,
			Name:       request.Name,
			Builtin:    true,
		},
	}, true, nil
}

func workflowFallbackProviders() map[declaration.Type]resolve.FallbackProvider {
	return map[declaration.Type]resolve.FallbackProvider{
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
	requireNoError(t, documentTopology.ValidateApplicationTopology())

	canonicalProvider, err := providercanonical.New()
	requireNoError(t, err)

	markdownProvider, err := providermarkdown.NewProvider()
	requireNoError(t, err)

	skillProvider, err := skillProviderAPI.NewProvider()
	requireNoError(t, err)

	mcpProvider, err := mcpProviderAPI.NewProvider()
	requireNoError(t, err)

	workspaceFS, err := builtin.EmbeddedWorkspacePackages()
	requireNoError(t, err)

	store, err := compositionapi.Open(
		ctx,
		compositionapi.Config{
			BaseDirectory: t.TempDir(),
			EmbeddedProviders: map[string]fs.FS{
				defaultpolicy.ProviderKey: workspaceFS,
			},
			Providers: []providerapi.Provider{
				canonicalProvider,
				markdownProvider,
				skillProvider,
				mcpProvider,
			},
			ProtectedRootIDs: documentTopology.ProtectedRootIDs(),
			RetainedRoots:    documentTopology.RetainedRootDrafts(),
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
		store.Discovery,
		store.Artifacts,
		store.Resources,
		store.ManagedArtifacts,
		store.Protection,
		agentConsumerAPI.WithRoots(store.Roots),
		agentConsumerAPI.WithFallbackProviders(
			workflowFallbackProviders(),
		),
		agentConsumerAPI.WithLocatorResolvers(
			store.LocatorResolvers,
		),
	)
	requireNoError(t, err)

	return &workflowHarness{
		store:           store,
		api:             api,
		dependencyFiles: make(map[workflowDependency]basespec.Locator),
	}
}

func (h *workflowHarness) installBundledAgents(
	t *testing.T,
) *agentBuiltin.Installer {
	t.Helper()

	ctx := installerapi.WithPrivilege(t.Context())
	_, err := h.store.EnsureProtectedTopology(
		ctx,
		documentTopology.BuiltinTopologyDeclaration(),
	)
	requireNoError(t, err)

	packages, err := builtin.EmbeddedAgentPackages()
	requireNoError(t, err)

	builtinStore, err := agentConsumerAPI.NewBuiltinStore(h.api)
	requireNoError(t, err)

	installer, err := agentBuiltin.NewInstaller(
		agentBuiltin.InstallerDependencies{
			Agents:   builtinStore,
			Packages: packages,
		},
	)
	requireNoError(t, err)

	requireNoError(t, installer.EnsureBuiltInArtifacts(ctx))
	h.addMissingBuiltinDependencies(t, ctx)
	requireNoError(t, installer.FinalizeHydration(ctx))

	return installer
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
			RootID: documentTopology.BuiltinRootID(),
		},
	)
	if err != nil {
		return nil, err
	}

	missing := make(map[workflowDependency]struct{})
	for _, agent := range agents {
		plan, err := h.api.ResolveAgentCapabilities(
			ctx,
			agent.Artifact.Ref(),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"resolve bundled Agent %q: %w",
				agent.Artifact.LogicalName,
				err,
			)
		}

		for _, occurrence := range plan.Occurrences {
			if !occurrence.Required ||
				occurrence.Status == resolve.ResolutionAvailable {
				continue
			}
			if occurrence.Name == "" {
				return nil, fmt.Errorf(
					"bundled Agent %q has unresolved unnamed %s capability at %q",
					agent.Artifact.LogicalName,
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

		locator := basespec.Locator(fmt.Sprintf(
			"dependency-%04d.yaml",
			h.nextDependencyFile,
		))
		h.nextDependencyFile++

		requireNoError(
			t,
			os.WriteFile(
				filepath.Join(
					h.dependencyDirectory,
					string(locator),
				),
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

		discovery := source.DiscoverySpec{
			DirectoryRoots: []source.DirectoryRoot{{
				Root:            ".",
				Recursive:       true,
				IncludePatterns: []string{"**/*.yaml"},
			}},
			AllowedDecoderIDs: []basespec.DecoderID{
				decoder.YAMLDecoderID,
			},
			Authoritative: true,
		}.Normalized()

		_, err = h.store.Sources.Create(
			ctx,
			documentTopology.BuiltinRootID(),
			source.Draft{
				ID:          workflowDependencySourceID,
				StorageKey:  "agent-test-dependencies",
				Kind:        source.SourceKindFilesystemDirectory,
				DisplayName: "Agent workflow test dependencies",
				Enabled:     true,
				Config:      json.RawMessage(config),
				Discovery:   discovery,
			},
		)
		requireNoError(t, err)
		h.dependencySourceCreated = true
	}

	refreshed, err := h.store.Discovery.RefreshSource(
		ctx,
		documentTopology.BuiltinRootID(),
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
	before []agentConsumerAPI.AgentView,
	after []agentConsumerAPI.AgentView,
) {
	t.Helper()

	if len(before) != len(after) {
		t.Fatalf(
			"Agent count changed after idempotent ensure: before=%d after=%d",
			len(before),
			len(after),
		)
	}

	revisions := make(map[artifact.ArtifactRef]uint64, len(before))
	for _, value := range before {
		revisions[value.Artifact.Ref()] = value.Artifact.Revision
	}

	for _, value := range after {
		revision, found := revisions[value.Artifact.Ref()]
		if !found {
			t.Fatalf(
				"unexpected Agent %q after idempotent ensure",
				value.Artifact.Ref(),
			)
		}
		if value.Artifact.Revision != revision {
			t.Fatalf(
				"Agent %q revision changed after idempotent ensure: got %d want %d",
				value.Artifact.Ref(),
				value.Artifact.Revision,
				revision,
			)
		}
	}
}
