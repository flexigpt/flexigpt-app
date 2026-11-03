package modelruntime

import (
	"encoding/json"
	"testing"
)

func TestRuntimeRequestPatchFromJSONUsesModelRuntimeNormalization(
	t *testing.T,
) {
	t.Parallel()

	patch, err := RuntimeRequestPatchFromJSON(
		json.RawMessage(`{
			"temperature": 0.25,
			"clear": ["systemPrompt"]
		}`),
	)
	if err != nil {
		t.Fatalf("RuntimeRequestPatchFromJSON: %v", err)
	}
	if patch == nil {
		t.Fatal("RuntimeRequestPatchFromJSON returned nil")
	}

	prepared, err := prepareRuntimeRequestPatch(patch)
	if err != nil {
		t.Fatalf("prepareRuntimeRequestPatch: %v", err)
	}

	defaults := map[string]any{
		"systemPrompt": "old system prompt",
	}
	if err := prepared.apply(defaults); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if _, found := defaults["systemPrompt"]; found {
		t.Fatal("system prompt clear was not applied")
	}
	if value, found := defaults["temperature"].(float64); !found ||
		value != 0.25 {
		t.Fatalf("temperature = %#v, want 0.25", defaults["temperature"])
	}
}
