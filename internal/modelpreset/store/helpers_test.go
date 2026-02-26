package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	inferencegoSpec "github.com/flexigpt/inference-go/spec"
)

const windowsGOOS = "windows"

func isWindows() bool { return runtime.GOOS == windowsGOOS }

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

func putUserProvider(t *testing.T, st *ModelPresetStore, name inferencegoSpec.ProviderName, enabled bool) {
	t.Helper()

	body := &spec.PutProviderPresetRequestBody{
		DisplayName:              spec.ProviderDisplayName(strings.ToUpper(string(name))),
		SDKType:                  inferencegoSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                enabled,
		Origin:                   "https://api." + string(name) + ".example.test",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		APIKeyHeaderKey:          spec.DefaultAuthorizationHeaderKey,
		DefaultHeaders:           spec.OpenAIChatCompletionsDefaultHeaders,
	}

	_, err := st.PutProviderPreset(t.Context(), &spec.PutProviderPresetRequest{
		ProviderName: name,
		Body:         body,
	})
	if err != nil {
		t.Fatalf("PutProviderPreset(%q): %v", name, err)
	}
}

func putUserModelPreset(
	t *testing.T,
	ctx context.Context,
	st *ModelPresetStore,
	provider inferencegoSpec.ProviderName,
	modelID spec.ModelPresetID,
	enabled bool,
) {
	t.Helper()

	temp := 0.1
	_, err := st.PutModelPreset(ctx, &spec.PutModelPresetRequest{
		ProviderName:  provider,
		ModelPresetID: modelID,
		Body: &spec.PutModelPresetRequestBody{
			Name:        spec.ModelName(modelID),
			Slug:        spec.ModelSlug(modelID),
			DisplayName: spec.ModelDisplayName(strings.ToUpper(string(modelID))),
			IsEnabled:   enabled,
			Temperature: &temp, // required (or reasoning)
		},
	})
	if err != nil {
		t.Fatalf("PutModelPreset(%q/%q): %v", provider, modelID, err)
	}
}

func listProvidersByNames(
	t *testing.T,
	st *ModelPresetStore,
	ctx context.Context,
	names []inferencegoSpec.ProviderName,
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

func getProviderByName(
	t *testing.T,
	st *ModelPresetStore,
	ctx context.Context,
	name inferencegoSpec.ProviderName,
	includeDisabled bool,
) spec.ProviderPreset {
	t.Helper()
	ps := listProvidersByNames(t, st, ctx, []inferencegoSpec.ProviderName{name}, includeDisabled)
	if len(ps) != 1 {
		t.Fatalf("expected exactly 1 provider %q, got %d", name, len(ps))
	}
	return ps[0]
}

func anyBuiltInProviderFromStore(
	t *testing.T,
	st *ModelPresetStore,
) (inferencegoSpec.ProviderName, spec.ProviderPreset) {
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
) (inferencegoSpec.ProviderName, spec.ProviderPreset) {
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

// Helper: schema with same model ID present in two different providers (scoped uniqueness).
func buildSchemaScopedDuplicateIDs(
	p1, p2 inferencegoSpec.ProviderName,
	commonID spec.ModelPresetID,
	extraID spec.ModelPresetID,
) []byte {
	mCommon := makeModelPreset(commonID)
	mExtra := makeModelPreset(extraID)

	pp1 := spec.ProviderPreset{
		SchemaVersion:            spec.SchemaVersion,
		Name:                     p1,
		DisplayName:              "Provider A",
		SDKType:                  inferencegoSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                true,
		CreatedAt:                time.Now(),
		ModifiedAt:               time.Now(),
		Origin:                   "https://example.com/a",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		DefaultModelPresetID:     commonID,
		ModelPresets: map[spec.ModelPresetID]spec.ModelPreset{
			commonID: mCommon,
			extraID:  mExtra,
		},
	}
	pp2 := spec.ProviderPreset{
		SchemaVersion:            spec.SchemaVersion,
		Name:                     p2,
		DisplayName:              "Provider B",
		SDKType:                  inferencegoSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                true,
		CreatedAt:                time.Now(),
		ModifiedAt:               time.Now(),
		Origin:                   "https://example.com/b",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		DefaultModelPresetID:     commonID,
		ModelPresets:             map[spec.ModelPresetID]spec.ModelPreset{commonID: mCommon},
	}

	s := spec.PresetsSchema{
		SchemaVersion:   spec.SchemaVersion,
		DefaultProvider: p1,
		ProviderPresets: map[inferencegoSpec.ProviderName]spec.ProviderPreset{p1: pp1, p2: pp2},
	}
	b, _ := json.Marshal(s)
	return b
}

func anyProvider(
	m map[inferencegoSpec.ProviderName]spec.ProviderPreset,
) (inferencegoSpec.ProviderName, spec.ProviderPreset) {
	for n, p := range m {
		return n, p
	}
	return "", spec.ProviderPreset{}
}

func anyModel(m map[inferencegoSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset,
) (inferencegoSpec.ProviderName, spec.ModelPresetID, spec.ModelPreset) {
	for pn, mm := range m {
		for mid, mp := range mm {
			return pn, mid, mp
		}
	}
	return "", "", spec.ModelPreset{}
}

func newPresetsFromFS(t *testing.T, mem fs.FS) (*BuiltInPresets, error) {
	t.Helper()
	ctx := t.Context()
	dir := t.TempDir()
	bi, err := NewBuiltInPresets(ctx, dir, time.Hour, WithModelPresetsFS(mem, "."))
	if err != nil {
		// If NewBuiltInPresets returned an error but allocated/started something,
		// make sure to close it immediately; otherwise tests on Windows can fail
		// during TempDir cleanup because the overlay sqlite file remains open.
		if bi != nil {
			_ = bi.Close()
			if runtime.GOOS == windows {
				// Give SQLite time to release handles on Windows.
				t.Log("modelpreset: sleeping in win")
				time.Sleep(time.Millisecond * 100)
			}
		}
		return nil, err
	}
	// Register normal cleanup for the successful case.
	t.Cleanup(func() {
		_ = bi.Close()
	})
	return bi, nil
}

func buildSchemaDefaultMissing(pn inferencegoSpec.ProviderName, mpid spec.ModelPresetID) []byte {
	model := makeModelPreset(mpid)
	pp := spec.ProviderPreset{
		SchemaVersion:            spec.SchemaVersion,
		Name:                     pn,
		DisplayName:              "Demo",
		SDKType:                  inferencegoSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                true,
		CreatedAt:                time.Now(),
		ModifiedAt:               time.Now(),
		Origin:                   "https://x",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		DefaultModelPresetID:     "ghost",
		ModelPresets:             map[spec.ModelPresetID]spec.ModelPreset{mpid: model},
	}
	s := spec.PresetsSchema{
		SchemaVersion:   spec.SchemaVersion,
		DefaultProvider: pn,
		ProviderPresets: map[inferencegoSpec.ProviderName]spec.ProviderPreset{pn: pp},
	}
	b, _ := json.Marshal(s)
	return b
}

func buildHappySchema(pn inferencegoSpec.ProviderName, mpid spec.ModelPresetID) []byte {
	model := makeModelPreset(mpid)
	pp := spec.ProviderPreset{
		SchemaVersion:            spec.SchemaVersion,
		Name:                     pn,
		DisplayName:              "Demo",
		SDKType:                  inferencegoSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                true,
		CreatedAt:                time.Now(),
		ModifiedAt:               time.Now(),
		Origin:                   "https://example.com",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		DefaultModelPresetID:     mpid,
		ModelPresets:             map[spec.ModelPresetID]spec.ModelPreset{mpid: model},
	}
	s := spec.PresetsSchema{
		SchemaVersion:   spec.SchemaVersion,
		DefaultProvider: pp.Name,
		ProviderPresets: map[inferencegoSpec.ProviderName]spec.ProviderPreset{pn: pp},
	}
	b, _ := json.Marshal(s)
	return b
}

func makeModelPreset(id spec.ModelPresetID) spec.ModelPreset {
	temp := 0.25
	now := time.Now().UTC()
	return spec.ModelPreset{
		SchemaVersion: spec.SchemaVersion,
		ID:            id,
		Name:          spec.ModelName(id),
		DisplayName:   spec.ModelDisplayName("Model " + string(id)),
		Slug:          spec.ModelSlug(id),
		IsEnabled:     true,

		Temperature: &temp, // validation requires reasoning or temp
		CreatedAt:   now,
		ModifiedAt:  now,
	}
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

func boolPtr(v bool) *bool { return &v }

func mpidPtr(v spec.ModelPresetID) *spec.ModelPresetID { return &v }

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
