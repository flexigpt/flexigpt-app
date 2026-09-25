package consumerapi

import (
	"encoding/json"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

func TestToolViewDoesNotExposeInternalDocuments(t *testing.T) {
	input := json.RawMessage(`{"type":"object"}`)
	user := json.RawMessage(`false`)
	value := toolDomain.Tool{
		Document: toolv1.ToolDocument{
			Header: declaration.Header{
				Type: toolv1.ToolType,
				Name: "test-tool",
			},
			Version:       "1",
			InputSchema:   input,
			UserArgSchema: &user,
			Implementation: toolv1.ToolImplementation{
				Kind:        toolv1.ImplementationKindSDK,
				SDKType:     "test-sdk",
				SDKToolType: toolv1.SDKToolTypeWebSearch,
			},
		},
	}

	view := toolView(value)
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}

	for _, forbidden := range []string{
		"Artifact", "Definition", "Document",
		"definition", "document", "metadata",
	} {
		if _, found := fields[forbidden]; found {
			t.Fatalf("view exposes %q: %s", forbidden, raw)
		}
	}
	if string(fields["userArgSchema"]) != "false" {
		t.Fatalf("boolean schema was not preserved: %s", raw)
	}

	view.InputSchema[0] = ' '
	(*view.UserArgSchema)[0] = ' '
	if string(input) != `{"type":"object"}` || string(user) != "false" {
		t.Fatal("view schema slices alias internal material")
	}
}
