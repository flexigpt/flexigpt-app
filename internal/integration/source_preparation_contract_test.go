package integration

import (
	"bytes"
	"encoding/json"
	"testing"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	mcpv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/contract/v1"
	mcppolicyv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcppolicy/contract/v1"
	modelv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/contract/v1"
	modelproviderv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/modelprovider/contract/v1"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
	skillv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/contract/v1"
	textv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/contract/v1"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
	workspacev1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

type sourceCase struct {
	name         string
	registration coreinterpretation.Registration
	body         string
	decode       func([]byte) error
}

func sourceCases() []sourceCase {
	return []sourceCase{
		{
			name:         "agent",
			registration: agentv1.Interpretation(),
			body:         `{"type":"agent","name":"sample"}`,
			decode: func(raw []byte) error {
				_, err := agentv1.DecodeAgentJSON(raw)
				return err
			},
		},
		{
			name:         "mcp",
			registration: mcpv1.Interpretation(),
			body:         `{"type":"mcp","name":"sample","transport":"stdio","command":"example"}`,
			decode: func(raw []byte) error {
				_, err := mcpv1.DecodeMCPJSON(raw)
				return err
			},
		},
		{
			name:         "mcp-policy",
			registration: mcppolicyv1.Interpretation(),
			body:         `{"type":"mcp.policy","name":"sample"}`,
			decode: func(raw []byte) error {
				_, err := mcppolicyv1.DecodeMCPPolicyJSON(raw)
				return err
			},
		},
		{
			name:         "model",
			registration: modelv1.Interpretation(),
			body: `{"type":"model","name":"sample",` +
				`"provider":{"name":"provider"},"providerModelID":"vendor-model"}`,
			decode: func(raw []byte) error {
				_, err := modelv1.DecodeModelJSON(raw)
				return err
			},
		},
		{
			name:         "model-provider",
			registration: modelproviderv1.Interpretation(),
			body:         `{"type":"model.provider","name":"sample","adapter":"provider.adapter"}`,
			decode: func(raw []byte) error {
				_, err := modelproviderv1.DecodeModelProviderJSON(raw)
				return err
			},
		},
		{
			name:         "plugin",
			registration: pluginv1.Interpretation(),
			body:         `{"type":"plugin","name":"sample"}`,
			decode: func(raw []byte) error {
				_, err := pluginv1.DecodePluginJSON(raw)
				return err
			},
		},
		{
			name:         "skill",
			registration: skillv1.Interpretation(support.Documents{}),
			body:         `{"type":"skill","name":"sample","locator":"./SKILL.md"}`,
			decode: func(raw []byte) error {
				_, err := skillv1.DecodeSkillJSON(raw)
				return err
			},
		},
		{
			name:         "text",
			registration: textv1.Interpretation(),
			body:         `{"type":"text","name":"sample","insert":"instructions","content":"hello"}`,
			decode: func(raw []byte) error {
				_, err := textv1.DecodeTextJSON(raw)
				return err
			},
		},
		{
			name:         "tool",
			registration: toolv1.Interpretation(),
			body: `{"type":"tool","name":"sample","version":"v1",` +
				`"autoExecute":false,"inputSchema":true,` +
				`"implementation":{"kind":"go","function":"example.function"}}`,
			decode: func(raw []byte) error {
				_, err := toolv1.DecodeToolJSON(raw)
				return err
			},
		},
		{
			name:         "workflow",
			registration: workflowv1.Interpretation(),
			body: `{"type":"workflow","name":"sample","inputSchema":true,` +
				`"outputSchema":true,"steps":[],"output":null}`,
			decode: func(raw []byte) error {
				_, err := workflowv1.DecodeWorkflowJSON(raw)
				return err
			},
		},
		{
			name:         "workspace",
			registration: workspacev1.Interpretation(),
			body:         `{"type":"workspace","name":"sample"}`,
			decode: func(raw []byte) error {
				_, err := workspacev1.DecodeWorkspaceJSON(raw)
				return err
			},
		},
	}
}

func sourceWithVersion(
	t *testing.T,
	body string,
	version json.RawMessage,
) []byte {
	t.Helper()

	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &fields); err != nil {
		t.Fatal(err)
	}
	if version != nil {
		fields["apiVersion"] = version
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestStandaloneSourcePreparationForEveryFamily(t *testing.T) {
	for _, test := range sourceCases() {
		t.Run(test.name, func(t *testing.T) {
			registry, err := coreinterpretation.NewRegistry(test.registration)
			if err != nil {
				t.Fatal(err)
			}

			withoutVersion := sourceWithVersion(t, test.body, nil)
			withVersion := sourceWithVersion(t, test.body, json.RawMessage(`"v1"`))
			wrongVersion := sourceWithVersion(t, test.body, json.RawMessage(`"v2"`))
			nullVersion := sourceWithVersion(t, test.body, json.RawMessage(`null`))

			for _, input := range []struct {
				name string
				raw  []byte
				ok   bool
			}{
				{
					name: "omitted",
					raw:  withoutVersion,
					ok:   !test.registration.RequireAPIVersion,
				},
				{name: "supported", raw: withVersion, ok: true},
				{name: "unsupported", raw: wrongVersion, ok: false},
				{name: "null", raw: nullVersion, ok: false},
			} {
				t.Run(input.name, func(t *testing.T) {
					_, prepareErr := registry.PrepareSourceDocument(
						test.registration.SchemaKey,
						input.raw,
					)
					decodeErr := test.decode(input.raw)
					_, definitionsErr := registry.DefinitionsForSourceDocument(
						test.registration.SchemaKey,
						input.raw,
					)
					if (prepareErr == nil) != input.ok ||
						(decodeErr == nil) != input.ok ||
						(definitionsErr == nil) != input.ok {
						t.Fatalf(
							"prepare=%v decode=%v definitions=%v; expected success=%v",
							prepareErr,
							decodeErr,
							definitionsErr,
							input.ok,
						)
					}
				})
			}

			prepared, err := registry.PrepareSourceDocument(
				test.registration.SchemaKey,
				withVersion,
			)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(prepared, []byte(`"apiVersion"`)) {
				t.Fatal("prepared Definition body retained apiVersion")
			}

			// Canonical-body admission is distinct from standalone source.
			// This also permits a canonical Workflow body without apiVersion.
			canonical, err := declaration.PrepareSourceDocument(
				test.registration.SchemaKey,
				withoutVersion,
				false,
			)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(prepared, canonical) {
				t.Fatalf("version changed canonical body:\n%s\n%s", prepared, canonical)
			}
			definitions, err := registry.DefinitionsForCanonicalDocument(
				test.registration.SchemaKey,
				canonical,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(definitions) != 1 ||
				!bytes.Equal(definitions[0].Definition.Body, canonical) {
				t.Fatal("canonical Definition body changed")
			}
		})
	}
}

func TestAgentYAMLAndJSONPrepareIdentically(t *testing.T) {
	registry, err := coreinterpretation.NewRegistry(agentv1.Interpretation())
	if err != nil {
		t.Fatal(err)
	}
	yamlJSON, err := yamlutil.CanonicalObjectJSON(
		[]byte("apiVersion: v1\ntype: agent\nname: sample\n"),
		storeSpec.MaxDefinitionBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	yamlBody, err := registry.PrepareSourceDocument(agentv1.AgentSchemaKey, yamlJSON)
	if err != nil {
		t.Fatal(err)
	}
	jsonBody, err := registry.PrepareSourceDocument(
		agentv1.AgentSchemaKey,
		[]byte(`{"apiVersion":"v1","type":"agent","name":"sample"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(yamlBody, jsonBody) {
		t.Fatalf("YAML and JSON prepared differently: %s != %s", yamlBody, jsonBody)
	}
}

func TestExplicitSchemaVersionIsRetainedForRootInterpretation(t *testing.T) {
	first := agentv1.Interpretation()
	second := first
	second.SchemaKey = schemaModel.ArtifactKey(
		artifactModel.ArtifactKind(declaration.TypeAgent),
		schemaModel.SchemaID("artifact.agent.v2"),
		"v2",
	)

	validate := func(entry declaration.Entry) error {
		return declaration.ValidateEntryType(entry, declaration.TypeAgent)
	}
	first.ValidateEntry = validate
	first.ValidateAdmittedEntry = validate
	first.Relationships = nil
	second.ValidateEntry = validate
	second.ValidateAdmittedEntry = validate
	second.Relationships = nil

	registry, err := coreinterpretation.NewRegistry(first, second)
	if err != nil {
		t.Fatal(err)
	}
	values, err := registry.DefinitionsForSourceDocument(
		second.SchemaKey,
		[]byte(`{"type":"agent","apiVersion":"v2","name":"sample"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].Definition.SchemaVersion != "v2" {
		t.Fatal("explicit schema version was not retained")
	}

	entry, err := declaration.DecodeEntryJSON([]byte(`{"type":"agent","name":"sample"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.AdmittedRelationshipsForSchema(second.SchemaKey, entry); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.AdmittedRelationships(entry); err == nil {
		t.Fatal("unkeyed interpretation should be ambiguous")
	}
}
