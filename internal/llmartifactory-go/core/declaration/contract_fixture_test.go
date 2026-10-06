package declaration_test

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	loopv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/loop/contract/v1"
	mcpv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/contract/v1"
	mcppolicyv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcppolicy/contract/v1"
	modelv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/contract/v1"
	modelproviderv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/modelprovider/contract/v1"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
	skillv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/contract/v1"
	teamv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/team/contract/v1"
	textv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/contract/v1"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
	workspacev1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const contractFixtureDirectory = "testdata/contracts"

//go:embed testdata/contracts/*.yaml
var contractFixtures embed.FS

func TestContractFixturesValidateThroughCompletePipeline(t *testing.T) {
	schemas := schemasByType(t)
	registry := declarationTestRegistry(t)
	entries, err := fs.ReadDir(contractFixtures, contractFixtureDirectory)
	if err != nil {
		t.Fatalf("read contract fixtures err: %v", err)
	}

	names := make([]string, 0, len(entries))
	for _, value := range entries {
		if value.IsDir() || path.Ext(value.Name()) != ".yaml" {
			continue
		}
		names = append(names, value.Name())
	}
	sort.Strings(names)
	if len(names) != len(declaration.Types()) {
		t.Fatalf(
			"fixture count = %d, want %d portable declaration types",
			len(names),
			len(declaration.Types()),
		)
	}

	seenTypes := make(map[declaration.Type]string, len(names))
	for _, fixtureName := range names {
		name := fixtureName
		t.Run(name, func(t *testing.T) {
			rawYAML, err := fs.ReadFile(
				contractFixtures,
				path.Join(contractFixtureDirectory, name),
			)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			raw, err := yamlutil.CanonicalObjectJSON(
				rawYAML,
				spec.MaxDefinitionBytes,
			)
			if err != nil {
				t.Fatalf("canonicalize YAML: %v", err)
			}

			entry, err := declaration.DecodeCanonicalEntryJSON(raw)
			if err != nil {
				t.Fatalf("decode generic Entry: %v", err)
			}
			if previous, duplicate := seenTypes[entry.Header().Type]; duplicate {
				t.Fatalf(
					"fixture type %q duplicates %q",
					entry.Header().Type,
					previous,
				)
			}
			seenTypes[entry.Header().Type] = name

			schemaJSON, found := schemas[entry.Header().Type]
			if !found {
				t.Fatalf(
					"no codec schema for declaration type %q",
					entry.Header().Type,
				)
			}
			compiled, err := jsonutil.CompileJSONSchema(schemaJSON)
			if err != nil {
				t.Fatalf("compile JSON Schema: %v", err)
			}
			if err := jsonutil.ValidateJSONSchema(
				compiled,
				json.RawMessage(raw),
				spec.MaxDefinitionBytes,
			); err != nil {
				t.Fatalf("validate fixture JSON Schema: %v", err)
			}

			canonicalDocument, err := canonicalConcreteDocument(entry)
			if err != nil {
				t.Fatalf("decode and create concrete document: %v", err)
			}
			rebuilt, err := declaration.DecodeEntryJSON(canonicalDocument)
			if err != nil {
				t.Fatalf("rebuild declaration Entry: %v", err)
			}
			if rebuilt.Header().Type != entry.Header().Type ||
				rebuilt.Header().Name != entry.Header().Name {
				t.Fatalf(
					"rebuilt declaration identity = %s/%s, want %s/%s",
					rebuilt.Header().Type,
					rebuilt.Header().Name,
					entry.Header().Type,
					entry.Header().Name,
				)
			}

			if err := registry.ValidateTree(entry); err != nil {
				t.Fatalf("validate complete declaration tree err: %v", err)
			}
			named, err := registry.WalkNamedEntries(entry)
			if err != nil {
				t.Fatalf("walk named declarations: %v", err)
			}
			if len(named) == 0 {
				t.Fatal("walk named declarations returned no root declaration")
			}

			for index, value := range named {
				if _, err := canonicalConcreteDocument(value.Entry); err != nil {
					t.Fatalf(
						"decode contained declaration %q: %v",
						value.SubresourceLocator,
						err,
					)
				}

				if index == 0 {
					definitionValue, err := registry.DefinitionForEntry(value.Entry)
					if err != nil {
						t.Fatalf("create root Definition: %v", err)
					}
					if err := definitionValue.Validate(); err != nil {
						t.Fatalf("validate root Definition: %v", err)
					}
					continue
				}

				definitionValue, err := registry.DefinitionForEntry(value.Entry)
				if err != nil {
					t.Fatalf(
						"create contained Definition %q: %v",
						value.SubresourceLocator,
						err,
					)
				}
				if err := definitionValue.Validate(); err != nil {
					t.Fatalf(
						"validate contained Definition %q: %v",
						value.SubresourceLocator,
						err,
					)
				}
			}
		})
	}

	for _, declarationType := range declaration.Types() {
		if _, found := seenTypes[declarationType]; !found {
			t.Errorf("missing fixture for declaration type %q", declarationType)
		}
	}
}

func TestMCPRejectsRetiredSSETransport(t *testing.T) {
	raw := []byte(`{
		"type":"mcp",
		"name":"deprecated-sse",
		"transport":"sse",
		"url":"https://example.com/mcp"
	}`)
	if _, err := mcpv1.DecodeMCPJSON(raw); err == nil {
		t.Fatal("DecodeMCPJSON() error = nil, want retired SSE rejection")
	}
}

func schemasByType(t *testing.T) map[declaration.Type][]byte {
	t.Helper()

	output := map[declaration.Type][]byte{
		textv1.TextType:                   textv1.TextJSONSchema(),
		modelv1.ModelType:                 modelv1.ModelJSONSchema(),
		modelproviderv1.ModelProviderType: modelproviderv1.ModelProviderJSONSchema(),
		toolv1.ToolType:                   toolv1.ToolJSONSchema(),
		skillv1.SkillType:                 skillv1.SkillJSONSchema(),
		mcpv1.MCPType:                     mcpv1.MCPJSONSchema(),
		mcppolicyv1.MCPPolicyType:         mcppolicyv1.MCPPolicyJSONSchema(),
		pluginv1.PluginType:               pluginv1.PluginJSONSchema(),
		agentv1.AgentType:                 agentv1.AgentJSONSchema(),
		teamv1.TeamType:                   teamv1.TeamJSONSchema(),
		loopv1.LoopType:                   loopv1.LoopJSONSchema(),
		workflowv1.WorkflowType:           workflowv1.WorkflowJSONSchema(),
		workspacev1.WorkspaceType:         workspacev1.WorkspaceJSONSchema(),
	}
	return output
}

func declarationTestRegistry(t *testing.T) *coreinterpretation.Registry {
	t.Helper()

	registry, err := coreinterpretation.NewRegistry(
		textv1.Interpretation(),
		modelv1.Interpretation(),
		modelproviderv1.Interpretation(),
		toolv1.Interpretation(),
		skillv1.Interpretation(),
		mcpv1.Interpretation(),
		mcppolicyv1.Interpretation(),
		pluginv1.Interpretation(),
		agentv1.Interpretation(),
		teamv1.Interpretation(),
		loopv1.Interpretation(),
		workflowv1.Interpretation(),
		workspacev1.Interpretation(),
	)
	if err != nil {
		t.Fatalf("create declaration interpretation registry: %v", err)
	}
	return registry
}

func canonicalConcreteDocument(
	entry declaration.Entry,
) ([]byte, error) {
	switch entry.Header().Type {
	case declaration.TypeText:
		value, err := textv1.DecodeTextEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypeModelProvider:
		value, err := modelproviderv1.DecodeModelProviderEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypeModel:
		value, err := modelv1.DecodeModelEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypeTool:
		value, err := toolv1.DecodeToolEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypeSkill:
		value, err := skillv1.DecodeSkillEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypeMCP:
		value, err := mcpv1.DecodeMCPEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypeMCPPolicy:
		value, err := mcppolicyv1.DecodeMCPPolicyEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypePlugin:
		value, err := pluginv1.DecodePluginEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypeAgent:
		value, err := agentv1.DecodeAgentEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypeTeam:
		value, err := teamv1.DecodeTeamEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypeLoop:
		value, err := loopv1.DecodeLoopEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypeWorkflow:
		value, err := workflowv1.DecodeWorkflowEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	case declaration.TypeWorkspace:
		value, err := workspacev1.DecodeWorkspaceEntry(entry)
		if err != nil {
			return nil, err
		}
		return value.CanonicalJSON()

	default:
		return nil, fmt.Errorf(
			"unsupported fixture declaration type %q",
			entry.Header().Type,
		)
	}
}
