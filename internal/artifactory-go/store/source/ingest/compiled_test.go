package ingest

import (
	"encoding/json"
	"testing"

	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestCompiledDocumentValidationAcceptsTrustedNonCanonicalDefinitionBody(
	t *testing.T,
) {
	t.Parallel()

	admitted, err := definitionModel.Canonicalize(definitionModel.Definition{
		Kind:          "agent",
		SchemaID:      "agent",
		SchemaVersion: "v1",
		LogicalName:   "compiled-agent",
		Body:          json.RawMessage(`{"a":1,"b":2}`),
	})
	if err != nil {
		t.Fatalf("canonicalize Definition: %v", err)
	}

	// This represents a generated JSON decode: semantic Definition content
	// and Digest are still correct, while the RawMessage byte form is not the
	// canonical representation retained during original admission.
	admitted.Body = json.RawMessage(`{"b":2, "a":1}`)

	document := CompiledDocument{
		Locator: "agent.json",
		Digest:  cryptoutil.DigestBytes([]byte("generated source witness")),
		Artifacts: []CompiledArtifact{{
			Definition: admitted,
		}},
	}

	if err := document.Validate(); err != nil {
		t.Fatalf(
			"trusted generated compiled document was rejected: %v",
			err,
		)
	}
}
