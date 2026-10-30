package integration

import (
	"bytes"
	"testing"

	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func TestWorkflowRegistrationUsesCanonicalDefinitionSchema(t *testing.T) {
	registry, err := registration.NewLLMInterpretationRegistry()
	if err != nil {
		t.Fatal(err)
	}
	codecs, err := registration.LLMDeclarationSchemaCodecsForSchemaKeys(
		[]schemaModel.Key{workflowv1.WorkflowSchemaKey},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(codecs) != 1 {
		t.Fatalf("got %d codecs", len(codecs))
	}
	if !bytes.Equal(codecs[0].JSONSchema(), workflowv1.WorkflowDefinitionJSONSchema()) {
		t.Fatal("Workflow codec registered the standalone source schema")
	}
	raw := []byte(`{
		"apiVersion":"v1",
		"type":"workflow",
		"name":"registered",
		"inputSchema":true,
		"outputSchema":true,
		"steps":[],
		"output":null
	}`)
	canonical, err := registry.PrepareSourceDocument(workflowv1.WorkflowSchemaKey, raw)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(canonical, []byte("apiVersion")) {
		t.Fatal("source preparation retained dispatch metadata")
	}
	if _, err := codecs[0].Canonicalize(t.Context(), canonical); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.PrepareSourceDocument(
		workflowv1.WorkflowSchemaKey,
		bytes.Replace(raw, []byte(`"apiVersion":"v1",`), nil, 1),
	); err == nil {
		t.Fatal("Workflow admission accepted an omitted source version")
	}
}
