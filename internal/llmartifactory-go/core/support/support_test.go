package support

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func TestDocumentsMatchesDeclaredNamedPatterns(t *testing.T) {
	t.Parallel()

	documents := Documents{
		Default: Document{
			Locator:   "agent.yaml",
			DecoderID: "artifact-declaration-yaml",
		},
		Files: []spec.Locator{
			"agent.yaml",
			"agent.yml",
			"agent.json",
		},
		Patterns: []string{
			"*.agent.yaml",
			"*.agent.yml",
			"*.agent.json",
		},
	}
	if err := documents.Validate(); err != nil {
		t.Fatalf("validate document support: %v", err)
	}

	for _, locator := range []spec.Locator{
		"agent.yaml",
		"nested/reviewer.agent.yaml",
		"nested/reviewer.agent.yml",
		"nested/reviewer.agent.json",
	} {
		if !documents.Matches(locator) {
			t.Fatalf("configured document support does not match %q", locator)
		}
	}

	if documents.Matches("nested/reviewer.yaml") {
		t.Fatal("configured document support matched an undeclared filename")
	}
}
