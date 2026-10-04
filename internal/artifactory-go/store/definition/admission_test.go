package definition

import (
	"bytes"
	"testing"

	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
)

func TestAdmitCanonicalizesDefinitionBeforePublication(t *testing.T) {
	t.Parallel()
	admitted, err := Admit(definitionModel.Definition{
		Kind:          "agent",
		SchemaID:      "agent",
		SchemaVersion: "v1",
		LogicalName:   "example",
		Body:          []byte(`{"b":2,"a":1}`),
	})
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if !bytes.Equal(admitted.Body, []byte(`{"a":1,"b":2}`)) {
		t.Fatalf("canonical body=%s", admitted.Body)
	}
	if err := definitionModel.ValidateAdmitted(admitted); err != nil {
		t.Fatalf("ValidateAdmitted: %v", err)
	}
}
