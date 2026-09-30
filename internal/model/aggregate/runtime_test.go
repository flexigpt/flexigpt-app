package aggregate

import "testing"

func TestRuntimeRequestPatchPrepareAppliesTypedDefaultsAndClear(
	t *testing.T,
) {
	t.Parallel()

	stream := false
	systemPrompt := "Use concise answers."
	patch := RuntimeRequestPatch{
		Defaults: &RuntimeDefaultsPatch{
			Stream:       &stream,
			SystemPrompt: &systemPrompt,
		},
		Clear: []RuntimeDefaultField{
			RuntimeDefaultFieldTemperature,
		},
	}
	prepared, err := patch.Prepare()
	if err != nil {
		t.Fatalf("prepare typed runtime request patch: %v", err)
	}

	defaults := map[string]any{
		"stream":       true,
		"temperature":  0.7,
		"systemPrompt": "old value",
	}
	if err := prepared.Apply(defaults); err != nil {
		t.Fatalf("apply prepared runtime request patch: %v", err)
	}

	if value, ok := defaults["stream"].(bool); !ok || value {
		t.Fatalf("stream = %#v, want false", defaults["stream"])
	}
	if value, ok := defaults["systemPrompt"].(string); !ok ||
		value != systemPrompt {
		t.Fatalf(
			"systemPrompt = %#v, want %q",
			defaults["systemPrompt"],
			systemPrompt,
		)
	}
	if _, found := defaults["temperature"]; found {
		t.Fatal("temperature was not cleared")
	}
	if prepared.Digest() == "" {
		t.Fatal("prepared patch has an empty digest")
	}
}

func TestNilRuntimeRequestPatchPrepareIsNoop(
	t *testing.T,
) {
	t.Parallel()

	var patch *RuntimeRequestPatch
	prepared, err := patch.Prepare()
	if err != nil {
		t.Fatalf("prepare nil runtime request patch: %v", err)
	}
	if prepared.Digest() != "" {
		t.Fatalf("nil patch digest = %q, want empty", prepared.Digest())
	}

	defaults := map[string]any{
		"temperature": 0.7,
	}
	if err := prepared.Apply(defaults); err != nil {
		t.Fatalf("apply nil runtime request patch: %v", err)
	}
	if defaults["temperature"] != 0.7 {
		t.Fatalf("defaults changed after nil patch: %#v", defaults)
	}
}

func TestRuntimeRequestPatchPrepareRejectsDuplicateClear(
	t *testing.T,
) {
	t.Parallel()

	patch := RuntimeRequestPatch{
		Clear: []RuntimeDefaultField{
			RuntimeDefaultFieldReasoning,
			RuntimeDefaultFieldReasoning,
		},
	}
	_, err := patch.Prepare()
	if err == nil {
		t.Fatal("Prepare() succeeded for duplicate clear fields")
	}
}
