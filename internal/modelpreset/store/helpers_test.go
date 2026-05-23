package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

const (
	renamed                 = "RENAMED"
	ghostID                 = "ghost"
	nonexistentProviderName = "nonexistent_provider"
	invalidTagText          = "invalid tag"
)

const windowsGOOS = "windows"

func decodeProviderPageToken(t *testing.T, token string) spec.ProviderPageToken {
	t.Helper()

	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("base64 decode page token: %v", err)
	}

	var tok spec.ProviderPageToken
	if err := json.Unmarshal(raw, &tok); err != nil {
		t.Fatalf("unmarshal page token: %v", err)
	}
	return tok
}

func newStore(t *testing.T) *ModelPresetStore {
	t.Helper()
	return newStoreAtDir(t, t.TempDir())
}

func newStoreAtDir(t *testing.T, dir string) *ModelPresetStore {
	t.Helper()

	mustMkdirAll(t, dir)
	st, err := NewModelPresetStore(dir)
	if err != nil {
		t.Fatalf("NewModelPresetStore(%q): %v", dir, err)
	}
	t.Cleanup(func() { closeAndSleepOnWindows(t, st) })
	return st
}

func closeAndSleepOnWindows(t *testing.T, c interface{ Close() error }) {
	t.Helper()
	if c == nil {
		return
	}
	_ = c.Close()
	if isWindows() {
		// SQLite + filesystem handle release can be slightly delayed on Windows.
		time.Sleep(75 * time.Millisecond)
	}
}

func postUserProvider(t *testing.T, st *ModelPresetStore, name inferenceSpec.ProviderName, enabled bool) {
	t.Helper()

	body := &spec.PostProviderPresetRequestBody{
		DisplayName:              spec.ProviderDisplayName(strings.ToUpper(string(name))),
		SDKType:                  inferenceSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                enabled,
		Origin:                   "https://api." + string(name) + ".example.test",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		APIKeyHeaderKey:          spec.DefaultAuthorizationHeaderKey,
		DefaultHeaders:           spec.OpenAIChatCompletionsDefaultHeaders,
	}

	_, err := st.PostProviderPreset(t.Context(), &spec.PostProviderPresetRequest{
		ProviderName: name,
		Body:         body,
	})
	if err != nil {
		t.Fatalf("PostProviderPreset(%q): %v", name, err)
	}
}

func postUserModelPreset(
	t *testing.T,
	ctx context.Context,
	st *ModelPresetStore,
	provider inferenceSpec.ProviderName,
	modelID spec.ModelPresetID,
	enabled bool,
) {
	t.Helper()

	temp := 0.1
	_, err := st.PostModelPreset(ctx, &spec.PostModelPresetRequest{
		ProviderName:  provider,
		ModelPresetID: modelID,
		Body: &spec.PostModelPresetRequestBody{
			Name:        spec.ModelName(modelID),
			Slug:        spec.ModelSlug(modelID),
			DisplayName: spec.ModelDisplayName(strings.ToUpper(string(modelID))),
			IsEnabled:   enabled,
			ModelPresetPatch: spec.ModelPresetPatch{
				Temperature: &temp, // required (or reasoning)
			},
		},
	})
	if err != nil {
		t.Fatalf("PostModelPreset(%q/%q): %v", provider, modelID, err)
	}
}

func getProviderByName(
	t *testing.T,
	st *ModelPresetStore,
	ctx context.Context,
	name inferenceSpec.ProviderName,
	includeDisabled bool,
) spec.ProviderPreset {
	t.Helper()
	ps := listProvidersByNames(t, st, ctx, []inferenceSpec.ProviderName{name}, includeDisabled)
	if len(ps) != 1 {
		t.Fatalf("expected exactly 1 provider %q, got %d", name, len(ps))
	}
	return ps[0]
}

func listProvidersByNames(
	t *testing.T,
	st *ModelPresetStore,
	ctx context.Context,
	names []inferenceSpec.ProviderName,
	includeDisabled bool,
) []spec.ProviderPreset {
	t.Helper()
	resp, err := st.ListProviderPresets(ctx, &spec.ListProviderPresetsRequest{
		Names:           names,
		IncludeDisabled: includeDisabled,
	})
	if err != nil {
		t.Fatalf("ListProviderPresets(names=%v): %v", names, err)
	}
	return resp.Body.Providers
}

func anyBuiltInProviderFromStore(
	t *testing.T,
	st *ModelPresetStore,
) (inferenceSpec.ProviderName, spec.ProviderPreset) {
	t.Helper()
	if st == nil || st.builtinData == nil {
		t.Fatalf("store has no builtinData")
	}
	prov, _, err := st.builtinData.ListBuiltInPresets(t.Context())
	if err != nil {
		t.Fatalf("builtinData.ListBuiltInPresets: %v", err)
	}
	for n, p := range prov {
		return n, p
	}
	t.Fatalf("no built-in providers found")
	return "", spec.ProviderPreset{}
}

func builtInProviderWithAtLeastNModelsFromStore(
	t *testing.T,
	ctx context.Context,
	st *ModelPresetStore,
	n int,
) (inferenceSpec.ProviderName, spec.ProviderPreset) {
	t.Helper()
	if st == nil || st.builtinData == nil {
		t.Fatalf("store has no builtinData")
	}
	prov, _, err := st.builtinData.ListBuiltInPresets(ctx)
	if err != nil {
		t.Fatalf("builtinData.ListBuiltInPresets: %v", err)
	}
	for pn, p := range prov {
		if len(p.ModelPresets) >= n {
			return pn, p
		}
	}
	t.Skipf("no built-in provider has >= %d models", n)
	return "", spec.ProviderPreset{}
}

func anyModelID(pp spec.ProviderPreset) (spec.ModelPresetID, spec.ModelPreset) {
	for id, mp := range pp.ModelPresets {
		return id, mp
	}
	return "", spec.ModelPreset{}
}

func anotherModelID(
	pp spec.ProviderPreset,
	not spec.ModelPresetID,
) spec.ModelPresetID {
	for id := range pp.ModelPresets {
		if id != not {
			return id
		}
	}
	return ""
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dir, err)
	}
}

func mustFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %q: %v", path, err)
	}
}

func wantErrIs(t *testing.T, err, target error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %v, got nil", target)
	}
	if !errors.Is(err, target) {
		t.Fatalf("expected error Is(%v), got %v", target, err)
	}
}

func wantErrContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("expected error containing %q, got %v", substr, err)
	}
}

func PrintJSON(v any) {
	p, err := json.MarshalIndent(v, "", "")
	if err == nil {
		fmt.Print("request params", "json", string(p))
	}
}

func providerDisplayNamePtr(v spec.ProviderDisplayName) *spec.ProviderDisplayName { return new(v) }

func mpidPtr(v spec.ModelPresetID) *spec.ModelPresetID { return new(v) }

func isWindows() bool { return runtime.GOOS == windowsGOOS }
