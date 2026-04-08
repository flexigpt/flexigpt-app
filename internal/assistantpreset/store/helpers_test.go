package store

import (
	"context"
	"encoding/json"
	"io/fs"
	"path"
	"testing"
	"testing/fstest"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	promptSpec "github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

func fixedTestTime() time.Time {
	return time.Date(2026, 3, 22, 10, 11, 12, 0, time.UTC)
}

func testBundleSlug(t *testing.T, suffix string) bundleitemutils.BundleSlug {
	t.Helper()

	candidates := []bundleitemutils.BundleSlug{
		bundleitemutils.BundleSlug("bundle-" + suffix),
		bundleitemutils.BundleSlug("bundle" + suffix),
		bundleitemutils.BundleSlug("b" + suffix),
	}

	for _, c := range candidates {
		if err := bundleitemutils.ValidateBundleSlug(c); err == nil {
			return c
		}
	}

	t.Fatalf("no valid bundle slug candidate for suffix %q", suffix)
	return ""
}

func testBundleID(t *testing.T, suffix string) bundleitemutils.BundleID {
	t.Helper()

	slug := testBundleSlug(t, suffix)
	candidates := []bundleitemutils.BundleID{
		bundleitemutils.BundleID("bundle-id-" + suffix),
		bundleitemutils.BundleID("bundle-" + suffix),
		bundleitemutils.BundleID("b-" + suffix),
	}

	for _, c := range candidates {
		if _, err := bundleitemutils.BuildBundleDir(c, slug); err == nil {
			return c
		}
	}

	t.Fatalf("no buildable bundle ID candidate for suffix %q", suffix)
	return ""
}

func testItemSlug(t *testing.T, suffix string) bundleitemutils.ItemSlug {
	t.Helper()

	candidates := []bundleitemutils.ItemSlug{
		bundleitemutils.ItemSlug("assistant-" + suffix),
		bundleitemutils.ItemSlug("assistant" + suffix),
		bundleitemutils.ItemSlug("a" + suffix),
	}

	for _, c := range candidates {
		if err := bundleitemutils.ValidateItemSlug(c); err == nil {
			return c
		}
	}

	t.Fatalf("no valid item slug candidate for suffix %q", suffix)
	return ""
}

func testItemVersion(t *testing.T) bundleitemutils.ItemVersion {
	t.Helper()

	candidates := []bundleitemutils.ItemVersion{
		"2026-03-22",
		"1.0.0",
		"v1",
		"1",
	}

	for _, c := range candidates {
		if err := bundleitemutils.ValidateItemVersion(c); err == nil {
			return c
		}
	}

	t.Fatal("no valid item version candidate found")
	return ""
}

func newTestBundle(t *testing.T, suffix string, enabled bool) spec.AssistantPresetBundle {
	t.Helper()

	now := fixedTestTime()

	return spec.AssistantPresetBundle{
		SchemaVersion: spec.SchemaVersion,
		ID:            testBundleID(t, suffix),
		Slug:          testBundleSlug(t, suffix),
		DisplayName:   "Bundle " + suffix,
		Description:   "Bundle description " + suffix,
		IsEnabled:     enabled,
		IsBuiltIn:     false,
		CreatedAt:     now,
		ModifiedAt:    now,
	}
}

func newTestPreset(t *testing.T, suffix string, enabled bool) spec.AssistantPreset {
	t.Helper()

	now := fixedTestTime()

	return spec.AssistantPreset{
		SchemaVersion: spec.SchemaVersion,
		ID:            bundleitemutils.ItemID("preset-id-" + suffix),
		Slug:          testItemSlug(t, suffix),
		Version:       testItemVersion(t),
		DisplayName:   "Preset " + suffix,
		Description:   "Preset description " + suffix,
		IsEnabled:     enabled,
		IsBuiltIn:     false,
		CreatedAt:     now,
		ModifiedAt:    now,
	}
}

func mustBuildBundleDir(
	t *testing.T,
	id bundleitemutils.BundleID,
	slug bundleitemutils.BundleSlug,
) bundleitemutils.BundleDirInfo {
	t.Helper()

	dirInfo, err := bundleitemutils.BuildBundleDir(id, slug)
	if err != nil {
		t.Fatalf("BuildBundleDir(%q, %q) error: %v", id, slug, err)
	}
	return dirInfo
}

func mustBuildItemFile(
	t *testing.T,
	slug bundleitemutils.ItemSlug,
	version bundleitemutils.ItemVersion,
) bundleitemutils.FileInfo {
	t.Helper()

	info, err := bundleitemutils.BuildItemFileInfo(slug, version)
	if err != nil {
		t.Fatalf("BuildItemFileInfo(%q, %q) error: %v", slug, version, err)
	}
	return info
}

func newBuiltInFS(
	t *testing.T,
	bundles map[bundleitemutils.BundleID]spec.AssistantPresetBundle,
	presets map[bundleitemutils.BundleID][]spec.AssistantPreset,
) fstest.MapFS {
	t.Helper()

	if bundles == nil {
		bundles = map[bundleitemutils.BundleID]spec.AssistantPresetBundle{}
	}
	if presets == nil {
		presets = map[bundleitemutils.BundleID][]spec.AssistantPreset{}
	}

	manifestRaw, err := json.Marshal(spec.AllBundles{
		SchemaVersion: spec.SchemaVersion,
		Bundles:       bundles,
	})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	fsys := fstest.MapFS{
		builtin.BuiltInAssistantPresetBundlesJSON: &fstest.MapFile{
			Data: manifestRaw,
		},
	}

	for bundleID, list := range presets {
		bundle, ok := bundles[bundleID]
		if !ok {
			t.Fatalf("preset bundle %q missing from manifest", bundleID)
		}

		dirInfo := mustBuildBundleDir(t, bundle.ID, bundle.Slug)

		for _, preset := range list {
			fileInfo := mustBuildItemFile(t, preset.Slug, preset.Version)

			raw, err := json.Marshal(preset)
			if err != nil {
				t.Fatalf("marshal built-in preset: %v", err)
			}

			fsys[path.Join(dirInfo.DirName, fileInfo.FileName)] = &fstest.MapFile{
				Data: raw,
			}
		}
	}

	return fsys
}

func newEmptyBuiltInFS(t *testing.T) fstest.MapFS {
	t.Helper()
	return newBuiltInFS(t, nil, nil)
}

type builtInFixture struct {
	fsys   fstest.MapFS
	bundle spec.AssistantPresetBundle
	preset spec.AssistantPreset
}

func newSingleBuiltInFixture(t *testing.T, bundleEnabled, presetEnabled bool) builtInFixture {
	t.Helper()

	bundle := newTestBundle(t, "builtin", bundleEnabled)
	bundle.DisplayName = "Built-in bundle"

	preset := newTestPreset(t, "builtin", presetEnabled)
	preset.DisplayName = "Built-in preset"

	return builtInFixture{
		fsys: newBuiltInFS(
			t,
			map[bundleitemutils.BundleID]spec.AssistantPresetBundle{
				bundle.ID: bundle,
			},
			map[bundleitemutils.BundleID][]spec.AssistantPreset{
				bundle.ID: {preset},
			},
		),
		bundle: bundle,
		preset: preset,
	}
}

func newTestStore(t *testing.T, builtins fs.FS, opts ...Option) *AssistantPresetStore {
	t.Helper()

	if builtins == nil {
		builtins = newEmptyBuiltInFS(t)
	}

	allOpts := append(
		[]Option{
			WithBuiltInDataOptions(WithBundlesFS(builtins, ".")),
		},
		opts...,
	)

	s, err := NewAssistantPresetStore(t.TempDir(), allOpts...)
	if err != nil {
		t.Fatalf("NewAssistantPresetStore() error: %v", err)
	}

	t.Cleanup(func() {
		_ = s.Close()
	})

	return s
}

func mustPutBundle(
	t *testing.T,
	s *AssistantPresetStore,
	bundleID bundleitemutils.BundleID,
	slug bundleitemutils.BundleSlug,
	enabled bool,
) {
	t.Helper()

	_, err := s.PutAssistantPresetBundle(t.Context(), &spec.PutAssistantPresetBundleRequest{
		BundleID: bundleID,
		Body: &spec.PutAssistantPresetBundleRequestBody{
			Slug:        slug,
			DisplayName: "bundle " + string(slug),
			Description: "desc " + string(slug),
			IsEnabled:   enabled,
		},
	})
	if err != nil {
		t.Fatalf("PutAssistantPresetBundle() error: %v", err)
	}
}

func mustPutPreset(
	t *testing.T,
	s *AssistantPresetStore,
	bundleID bundleitemutils.BundleID,
	slug bundleitemutils.ItemSlug,
	version bundleitemutils.ItemVersion,
	enabled bool,
) {
	t.Helper()

	_, err := s.PutAssistantPreset(t.Context(), &spec.PutAssistantPresetRequest{
		BundleID:            bundleID,
		AssistantPresetSlug: slug,
		Version:             version,
		Body: &spec.PutAssistantPresetRequestBody{
			DisplayName: "preset " + string(slug),
			Description: "desc " + string(slug),
			IsEnabled:   enabled,
		},
	})
	if err != nil {
		t.Fatalf("PutAssistantPreset() error: %v", err)
	}
}

func mustGetAssistantPreset(
	t *testing.T,
	s *AssistantPresetStore,
	bundleID bundleitemutils.BundleID,
	slug bundleitemutils.ItemSlug,
	version bundleitemutils.ItemVersion,
) spec.AssistantPreset {
	t.Helper()

	resp, err := s.GetAssistantPreset(t.Context(), &spec.GetAssistantPresetRequest{
		BundleID:            bundleID,
		AssistantPresetSlug: slug,
		Version:             version,
	})
	if err != nil {
		t.Fatalf("GetAssistantPreset() error: %v", err)
	}
	if resp == nil || resp.Body == nil {
		t.Fatal("GetAssistantPreset() returned nil body")
	}

	return *resp.Body
}

func boolPtr(v bool) *bool {
	return &v
}

func collectBundleIDs(items []spec.AssistantPresetBundle) map[bundleitemutils.BundleID]struct{} {
	out := make(map[bundleitemutils.BundleID]struct{}, len(items))
	for _, item := range items {
		out[item.ID] = struct{}{}
	}
	return out
}

func presetListKey(item spec.AssistantPresetListItem) string {
	return string(item.BundleID) + "|" + string(item.AssistantPresetSlug) + "|" + string(item.AssistantPresetVersion)
}

func collectPresetKeys(items []spec.AssistantPresetListItem) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[presetListKey(item)] = struct{}{}
	}
	return out
}

type fakeModelPresetLookup func(context.Context, modelpresetSpec.ModelPresetRef) (ModelPresetSummary, error)

func (f fakeModelPresetLookup) GetModelPresetSummary(
	ctx context.Context,
	ref modelpresetSpec.ModelPresetRef,
) (ModelPresetSummary, error) {
	return f(ctx, ref)
}

type fakePromptTemplateLookup func(context.Context, promptSpec.PromptTemplateRef) (PromptTemplateSummary, error)

func (f fakePromptTemplateLookup) GetPromptTemplateSummary(
	ctx context.Context,
	ref promptSpec.PromptTemplateRef,
) (PromptTemplateSummary, error) {
	return f(ctx, ref)
}

type fakeToolSelectionLookup func(context.Context, toolSpec.ToolSelection) (ToolSummary, error)

func (f fakeToolSelectionLookup) GetToolSummaryForSelection(
	ctx context.Context,
	selection toolSpec.ToolSelection,
) (ToolSummary, error) {
	return f(ctx, selection)
}

type fakeSkillLookup func(context.Context, skillSpec.SkillSelection) (SkillSummary, error)

func (f fakeSkillLookup) GetSkillSummaryForSelection(
	ctx context.Context,
	selection skillSpec.SkillSelection,
) (SkillSummary, error) {
	return f(ctx, selection)
}
