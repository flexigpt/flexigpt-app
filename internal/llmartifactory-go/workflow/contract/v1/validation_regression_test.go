package workflowv1

import (
	"encoding/json"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func workflowSource(steps, extra string) []byte {
	return []byte(`{
		"apiVersion":"v1",
		"type":"workflow",
		"name":"regression",
		"inputSchema":true,
		"outputSchema":true,
		"steps":` + steps + `,
		"output":null` + extra + `
	}`)
}

func TestLiteralNumbersPreservePrecision(t *testing.T) {
	value, known, err := literalValue(Value(`9007199254740993`))
	if err != nil || !known {
		t.Fatalf("known=%v err=%v", known, err)
	}
	number, ok := value.(json.Number)
	if !ok || number.String() != "9007199254740993" {
		t.Fatalf("literal changed: %#v", value)
	}
}

func TestSchemaTraversalIgnoresInstanceData(t *testing.T) {
	schema := map[string]any{
		"const": map[string]any{
			"$id":  "https://example.com/not-a-schema",
			"$ref": "https://example.com/business-data",
		},
	}
	ids := map[string]bool{}
	collectSchemaIDs(schema, ids)
	if ids["https://example.com/not-a-schema"] {
		t.Fatal("literal $id was treated as a schema resource")
	}
	if err := checkSchemaReferences(schema, ids); err != nil {
		t.Fatal(err)
	}

	schema["$ref"] = "https://example.com/not-a-schema"
	if err := checkSchemaReferences(schema, ids); err == nil {
		t.Fatal("literal $id authorized a non-local schema reference")
	}
}

func TestNullDefsAndBooleanSchemas(t *testing.T) {
	document := WorkflowDocument{
		Schemas: map[string]Schema{
			"local": Schema(`{"type":"string"}`),
		},
	}
	if _, err := document.schemaObject(Schema(`{"$defs":null}`)); err == nil {
		t.Fatal("accepted null $defs")
	}
	compiled, err := document.compileSchema(Schema(`false`))
	if err != nil {
		t.Fatal(err)
	}
	if err := jsonutil.ValidateJSONSchema(compiled, json.RawMessage(`null`), 1024); err == nil {
		t.Fatal("false schema accepted a value")
	}
}

func TestLiteralForeachItemIsNotReinterpreted(t *testing.T) {
	raw := workflowSource(`[
		{
			"id":"each",
			"foreach":{
				"items":[{"$literal":{"$expr":"business data"}}],
				"itemSchema":{"type":"string"},
				"do":{"steps":[]}
			}
		}
	]`, "")
	if _, err := DecodeWorkflowJSON(raw); err == nil {
		t.Fatal("object item was accepted by a string schema")
	}
}

func TestTransformCommonOutputSchema(t *testing.T) {
	raw := workflowSource(`[
		{
			"id":"transform",
			"outputSchema":{"type":"integer"},
			"transform":{"value":"not-an-integer"}
		}
	]`, "")
	if _, err := DecodeWorkflowJSON(raw); err == nil {
		t.Fatal("literal transform violated its step outputSchema")
	}
}

func TestLiteralCallChecksResolvedTargetSet(t *testing.T) {
	raw := workflowSource(`[
		{"id":"invoke","call":{"target":"writer","allowed":"choices","input":{}}}
	]`, `,
		"targets":{"writer":{"kind":"agent","ref":"writer"}},
		"targetSets":{
			"choices":{
				"resolved":{
					"value":{"$expr":"steps.missing.output"},
					"interface":{"inputSchema":true,"outputSchema":true}
				}
			}
		}`)
	if _, err := DecodeWorkflowJSON(raw); err == nil {
		t.Fatal("literal call accepted an invalid resolved target-set expression")
	}
}
