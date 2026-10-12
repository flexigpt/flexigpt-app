package modelruntime

import (
	"errors"
	"testing"

	inferenceRuntimeSpec "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference/spec"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
)

func TestRuntimeRequestPatchPrepareAppliesTypedDefaultsAndClear(t *testing.T) {
	t.Parallel()

	stream := false
	systemPrompt := "Use concise answers."
	patch := inferenceRuntimeSpec.RuntimeRequestPatch{
		Defaults: &modelDomain.DefaultsPatch{
			Stream:       &stream,
			SystemPrompt: &systemPrompt,
		},
		Clear: []inferenceRuntimeSpec.RuntimeDefaultField{
			inferenceRuntimeSpec.RuntimeDefaultFieldTemperature,
		},
	}
	prepared, err := prepareRuntimeRequestPatch(&patch)
	if err != nil {
		t.Fatalf("prepare typed runtime request patch: %v", err)
	}

	defaults := map[string]any{
		"stream":       true,
		"temperature":  0.7,
		"systemPrompt": "old value",
	}
	if err := prepared.apply(defaults); err != nil {
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
	if prepared.digest == "" {
		t.Fatal("prepared patch has an empty digest")
	}
}

func TestNilRuntimeRequestPatchPrepareIsNoop(t *testing.T) {
	t.Parallel()

	var patch *inferenceRuntimeSpec.RuntimeRequestPatch
	prepared, err := prepareRuntimeRequestPatch(patch)
	if err != nil {
		t.Fatalf("prepare nil runtime request patch: %v", err)
	}
	if prepared.digest != "" {
		t.Fatalf("nil patch digest = %q, want empty", prepared.digest)
	}

	defaults := map[string]any{"temperature": 0.7}
	if err := prepared.apply(defaults); err != nil {
		t.Fatalf("apply nil runtime request patch: %v", err)
	}
	if defaults["temperature"] != 0.7 {
		t.Fatalf("defaults changed after nil patch: %#v", defaults)
	}
}

func TestRuntimeRequestPatchPrepareRejectsDuplicateClear(t *testing.T) {
	t.Parallel()

	patch := inferenceRuntimeSpec.RuntimeRequestPatch{
		Clear: []inferenceRuntimeSpec.RuntimeDefaultField{
			inferenceRuntimeSpec.RuntimeDefaultFieldReasoning,
			inferenceRuntimeSpec.RuntimeDefaultFieldReasoning,
		},
	}
	_, err := prepareRuntimeRequestPatch(&patch)
	if !errors.Is(err, storeSpec.ErrInvalid) {
		t.Fatalf("prepare duplicate clear fields: got %v, want ErrInvalid", err)
	}
}

func TestRuntimeRequestPatchPrepareRejectsUnknownClear(t *testing.T) {
	t.Parallel()

	patch := inferenceRuntimeSpec.RuntimeRequestPatch{
		Clear: []inferenceRuntimeSpec.RuntimeDefaultField{"unknown"},
	}
	_, err := prepareRuntimeRequestPatch(&patch)
	if !errors.Is(err, storeSpec.ErrInvalid) {
		t.Fatalf("prepare unknown clear field: got %v, want ErrInvalid", err)
	}
}

func TestRuntimeRequestPatchClearWinsOverDefaults(t *testing.T) {
	t.Parallel()

	temperature := 0.4
	patch := inferenceRuntimeSpec.RuntimeRequestPatch{
		Defaults: &modelDomain.DefaultsPatch{
			Temperature: &temperature,
		},
		Clear: []inferenceRuntimeSpec.RuntimeDefaultField{
			inferenceRuntimeSpec.RuntimeDefaultFieldTemperature,
		},
	}
	prepared, err := prepareRuntimeRequestPatch(&patch)
	if err != nil {
		t.Fatalf("prepare runtime request patch: %v", err)
	}

	defaults := map[string]any{"temperature": 0.7}
	if err := prepared.apply(defaults); err != nil {
		t.Fatalf("apply runtime request patch: %v", err)
	}
	if _, found := defaults["temperature"]; found {
		t.Fatal("request defaults overrode an explicit clear")
	}
}

func TestApplyRuntimeDefaultsPreservesLayerMergeSemantics(t *testing.T) {
	t.Parallel()

	defaults := map[string]any{
		"output": map[string]any{
			"verbosity": "low",
			"format": map[string]any{
				"kind": "text",
			},
		},
		"stopSequences": []any{"existing"},
		"adapterParameters": map[string]any{
			"old": true,
		},
	}
	patch := map[string]any{
		"output": map[string]any{
			"verbosity": "high",
		},
		"stopSequences": []any{},
		"adapterParameters": map[string]any{
			"new": true,
		},
	}
	if err := applyRuntimeDefaults(defaults, patch); err != nil {
		t.Fatalf("merge defaults: %v", err)
	}

	output, ok := defaults["output"].(map[string]any)
	if !ok || output["verbosity"] != "high" || output["format"] == nil {
		t.Fatalf("output was not shallow-merged: %#v", output)
	}
	stops, ok := defaults["stopSequences"].([]any)
	if !ok || len(stops) != 1 || stops[0] != "existing" {
		t.Fatalf("empty stopSequences replaced the existing value: %#v", stops)
	}
	parameters, ok := defaults["adapterParameters"].(map[string]any)
	if !ok || len(parameters) != 1 || parameters["new"] != true {
		t.Fatalf("adapterParameters did not replace atomically: %#v", parameters)
	}
}
