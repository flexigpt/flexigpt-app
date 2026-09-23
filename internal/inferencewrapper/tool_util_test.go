package inferencewrapper

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func TestDecodeToolArgSchemaUnwrapsWailsJSONString(t *testing.T) {
	raw := jsonutil.JSONRawString(
		`"{\"type\":\"object\",\"properties\":{\"path\":{\"type\":\"string\"}}}"`,
	)

	schema, err := decodeToolArgSchema(raw)
	if err != nil {
		t.Fatalf("decodeToolArgSchema() error = %v", err)
	}
	if got := schema["type"]; got != "object" {
		t.Fatalf("schema.type = %#v, want object", got)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema.properties = %#v, want object", schema["properties"])
	}
	if _, found := properties["path"]; !found {
		t.Fatalf("schema.properties = %#v, want path property", properties)
	}
}

func TestDecodeToolArgSchemaAcceptsDirectJSON(t *testing.T) {
	schema, err := decodeToolArgSchema(
		jsonutil.JSONRawString(`{"type":"object"}`),
	)
	if err != nil {
		t.Fatalf("decodeToolArgSchema() error = %v", err)
	}
	if got := schema["type"]; got != "object" {
		t.Fatalf("schema.type = %#v, want object", got)
	}
}

func TestDecodeToolArgSchemaUsesEmptyObjectForBlankInput(t *testing.T) {
	schema, err := decodeToolArgSchema("")
	if err != nil {
		t.Fatalf("decodeToolArgSchema() error = %v", err)
	}
	if got := schema["type"]; got != "object" {
		t.Fatalf("schema.type = %#v, want object", got)
	}
}
