package toolruntime

import (
	"encoding/json"
	"testing"
)

func TestStableChoiceIDIsDeterministicAndDelimited(t *testing.T) {
	first := StableChoiceID("tool", "a", "bc")
	second := StableChoiceID("tool", "ab", "c")
	again := StableChoiceID("tool", "a", "bc")

	if first == second {
		t.Fatalf("delimited inputs collided: %q", first)
	}
	if first != again {
		t.Fatalf("stable choice ID changed: %q != %q", first, again)
	}
}

func TestInputSchemaCanonicalizesObjectSchemas(t *testing.T) {
	schema, err := InputSchema(
		json.RawMessage(`{"properties":{"name":{"type":"string"}},"type":"object"}`),
	)
	if err != nil {
		t.Fatalf("InputSchema: %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("schema type = %#v, want object", schema["type"])
	}

	if _, err := InputSchema(json.RawMessage(`false`)); err == nil {
		t.Fatal("false schema was accepted")
	}
}
