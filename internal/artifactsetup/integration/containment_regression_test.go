package integration

import (
	"testing"

	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
	textv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/contract/v1"
)

func TestNestedContainedPathsIncludeTheirParent(t *testing.T) {
	raw := []byte(`{
		"type":"plugin",
		"name":"outer",
		"members":[
			{
				"type":"agent",
				"name":"first",
				"parameters":{
					"members":[{
						"type":"text",
						"name":"intro",
						"insert":"instructions",
						"parameters":{"content":"first"}
					}]
				}
			},
			{
				"type":"agent",
				"name":"second",
				"parameters":{
					"members":[{
						"type":"text",
						"name":"intro",
						"insert":"instructions",
						"parameters":{"content":"second"}
					}]
				}
			}
		]
	}`)
	entry, err := declaration.DecodeEntryJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := interpretation.NewRegistry(
		pluginv1.Interpretation(),
		agentv1.Interpretation(),
		textv1.Interpretation(),
	)
	if err != nil {
		t.Fatal(err)
	}
	values, err := registry.WalkNamedEntries(entry)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"":                    true,
		"members/agent/first": true,
		"members/agent/first/members/text/instructions/intro":  true,
		"members/agent/second":                                 true,
		"members/agent/second/members/text/instructions/intro": true,
	}
	if len(values) != len(want) {
		t.Fatalf("got %d entries, expected %d", len(values), len(want))
	}
	for _, value := range values {
		path := string(value.SubresourceLocator)
		if !want[path] {
			t.Fatalf("unexpected contained path %q", path)
		}
		delete(want, path)
	}
	if len(want) != 0 {
		t.Fatalf("missing contained paths: %v", want)
	}

	canonical, err := entry.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	definitions, err := registry.DefinitionsForCanonicalDocument(
		pluginv1.PluginSchemaKey,
		canonical,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 5 {
		t.Fatalf("got %d Definitions, expected 5", len(definitions))
	}
}
