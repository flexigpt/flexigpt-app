package store

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/fsutil"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	promptSpec "github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

func TestNewBuiltInData_InvalidOverlayDir(t *testing.T) {
	_, err := NewBuiltInData(
		t.Context(),
		"",
		0,
		ReferenceLookups{},
		WithBundlesFS(newEmptyBuiltInFS(t), "."),
	)
	if !errors.Is(err, spec.ErrInvalidDir) {
		t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrInvalidDir)
	}
}

func TestBuiltInData_EmptyManifest(t *testing.T) {
	d, err := NewBuiltInData(
		t.Context(),
		t.TempDir(),
		0,
		ReferenceLookups{},
		WithBundlesFS(newEmptyBuiltInFS(t), "."),
	)
	if err != nil {
		t.Fatalf("NewBuiltInData() error: %v", err)
	}
	defer func() { _ = d.Close() }()

	bundles, presets, err := d.ListBuiltInData(t.Context())
	if err != nil {
		t.Fatalf("ListBuiltInData() error: %v", err)
	}
	if len(bundles) != 0 {
		t.Fatalf("len(bundles) = %d, want 0", len(bundles))
	}
	if len(presets) != 0 {
		t.Fatalf("len(presets) = %d, want 0", len(presets))
	}
}

func TestBuiltInData_ListBuiltInData_DeepCopyAndGetters(t *testing.T) {
	bundle := newTestBundle(t, "deepcopy", true)
	preset := newTestPreset(t, "deepcopy", true)
	preset.StartingModelPresetRef = &modelpresetSpec.ModelPresetRef{
		ProviderName:  "provider-a",
		ModelPresetID: "mp-a",
	}
	preset.StartingIncludeModelSystemPrompt = boolPtr(true)
	preset.StartingInstructionTemplateRefs = []promptSpec.PromptTemplateRef{
		{
			BundleID:        bundleitemutils.BundleID("bundle-a"),
			TemplateSlug:    "tmpl-a",
			TemplateVersion: testItemVersion(t),
		},
	}
	preset.StartingToolSelections = []toolSpec.ToolSelection{
		{
			ToolRef: toolSpec.ToolRef{
				BundleID:    bundleitemutils.BundleID("bundle-a"),
				ToolSlug:    "tool-a",
				ToolVersion: testItemVersion(t),
			},
		},
	}
	preset.StartingSkillSelections = []skillSpec.SkillSelection{
		{
			SkillRef: skillSpec.SkillRef{
				BundleID:  bundleitemutils.BundleID("bundle-a"),
				SkillSlug: "skill-a",
				SkillID:   "skill-id-a",
			},
		},
	}

	lookups := ReferenceLookups{
		ModelPresets: fakeModelPresetLookup(
			func(context.Context, modelpresetSpec.ModelPresetRef) (ModelPresetSummary, error) {
				return ModelPresetSummary{IsEnabled: true}, nil
			},
		),
		PromptTemplates: fakePromptTemplateLookup(
			func(context.Context, promptSpec.PromptTemplateRef) (PromptTemplateSummary, error) {
				return PromptTemplateSummary{
					IsEnabled:  true,
					Kind:       promptSpec.PromptTemplateKindInstructionsOnly,
					IsResolved: true,
				}, nil
			},
		),
		ToolSelections: fakeToolSelectionLookup(func(context.Context, toolSpec.ToolSelection) (ToolSummary, error) {
			return ToolSummary{IsEnabled: true}, nil
		}),
		Skills: fakeSkillLookup(func(context.Context, skillSpec.SkillSelection) (SkillSummary, error) {
			return SkillSummary{IsEnabled: true}, nil
		}),
	}

	d, err := NewBuiltInData(
		t.Context(),
		t.TempDir(),
		0,
		lookups,
		WithBundlesFS(newBuiltInFS(
			t,
			map[bundleitemutils.BundleID]spec.AssistantPresetBundle{
				bundle.ID: bundle,
			},
			map[bundleitemutils.BundleID][]spec.AssistantPreset{
				bundle.ID: {preset},
			},
		), "."),
	)
	if err != nil {
		t.Fatalf("NewBuiltInData() error: %v", err)
	}
	defer func() { _ = d.Close() }()

	bundles, presets, err := d.ListBuiltInData(t.Context())
	if err != nil {
		t.Fatalf("ListBuiltInData() error: %v", err)
	}

	// Mutate returned snapshots.
	b := bundles[bundle.ID]
	b.DisplayName = "mutated bundle"
	bundles[bundle.ID] = b
	delete(bundles, bundle.ID)

	p := presets[bundle.ID][preset.ID]
	p.DisplayName = "mutated preset"
	*p.StartingIncludeModelSystemPrompt = false
	p.StartingInstructionTemplateRefs[0].TemplateSlug = "mutated-template"
	p.StartingToolSelections[0].ToolRef.ToolSlug = "mutated-tool"
	p.StartingSkillSelections[0].SkillRef.SkillSlug = "mutated-skill"
	presets[bundle.ID][preset.ID] = p
	delete(presets[bundle.ID], preset.ID)

	gotBundle, err := d.GetBuiltInBundle(t.Context(), bundle.ID)
	if err != nil {
		t.Fatalf("GetBuiltInBundle() error: %v", err)
	}
	if gotBundle.DisplayName != bundle.DisplayName {
		t.Fatalf("GetBuiltInBundle().DisplayName = %q, want %q", gotBundle.DisplayName, bundle.DisplayName)
	}

	gotPreset, err := d.GetBuiltInAssistantPreset(
		t.Context(),
		bundle.ID,
		preset.Slug,
		preset.Version,
	)
	if err != nil {
		t.Fatalf("GetBuiltInAssistantPreset() error: %v", err)
	}
	if gotPreset.DisplayName != preset.DisplayName {
		t.Fatalf("GetBuiltInAssistantPreset().DisplayName = %q, want %q", gotPreset.DisplayName, preset.DisplayName)
	}
	if gotPreset.StartingIncludeModelSystemPrompt == nil || *gotPreset.StartingIncludeModelSystemPrompt != true {
		t.Fatalf(
			"GetBuiltInAssistantPreset().StartingIncludeModelSystemPrompt = %v, want true",
			gotPreset.StartingIncludeModelSystemPrompt,
		)
	}
	if gotPreset.StartingInstructionTemplateRefs[0].TemplateSlug != "tmpl-a" {
		t.Fatalf("template slug = %q, want %q", gotPreset.StartingInstructionTemplateRefs[0].TemplateSlug, "tmpl-a")
	}
	if gotPreset.StartingToolSelections[0].ToolRef.ToolSlug != "tool-a" {
		t.Fatalf("tool slug = %q, want %q", gotPreset.StartingToolSelections[0].ToolRef.ToolSlug, "tool-a")
	}
	if gotPreset.StartingSkillSelections[0].SkillRef.SkillSlug != "skill-a" {
		t.Fatalf("skill slug = %q, want %q", gotPreset.StartingSkillSelections[0].SkillRef.SkillSlug, "skill-a")
	}
}

func TestBuiltInData_SettersAndErrors(t *testing.T) {
	fixture := newSingleBuiltInFixture(t, true, true)

	d, err := NewBuiltInData(
		t.Context(),
		t.TempDir(),
		0,
		ReferenceLookups{},
		WithBundlesFS(fixture.fsys, "."),
	)
	if err != nil {
		t.Fatalf("NewBuiltInData() error: %v", err)
	}
	defer func() { _ = d.Close() }()

	t.Run("missing bundle errors", func(t *testing.T) {
		_, err := d.SetAssistantPresetBundleEnabled(t.Context(), "missing-bundle", false)
		if !errors.Is(err, spec.ErrBuiltInBundleNotFound) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBuiltInBundleNotFound)
		}

		_, err = d.GetBuiltInBundle(t.Context(), "missing-bundle")
		if !errors.Is(err, spec.ErrBundleNotFound) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBundleNotFound)
		}

		_, err = d.GetBuiltInAssistantPreset(
			t.Context(),
			"missing-bundle",
			fixture.preset.Slug,
			fixture.preset.Version,
		)
		if !errors.Is(err, spec.ErrBundleNotFound) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBundleNotFound)
		}
	})

	t.Run("missing preset error", func(t *testing.T) {
		_, err := d.GetBuiltInAssistantPreset(
			t.Context(),
			fixture.bundle.ID,
			testItemSlug(t, "other"),
			fixture.preset.Version,
		)
		if !errors.Is(err, spec.ErrAssistantPresetNotFound) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrAssistantPresetNotFound)
		}
	})

	t.Run("set bundle enabled", func(t *testing.T) {
		got, err := d.SetAssistantPresetBundleEnabled(t.Context(), fixture.bundle.ID, false)
		if err != nil {
			t.Fatalf("SetAssistantPresetBundleEnabled() error: %v", err)
		}
		if got.IsEnabled {
			t.Fatal("bundle should be disabled")
		}

		stored, err := d.GetBuiltInBundle(t.Context(), fixture.bundle.ID)
		if err != nil {
			t.Fatalf("GetBuiltInBundle() error: %v", err)
		}
		if stored.IsEnabled {
			t.Fatal("stored bundle should be disabled")
		}
	})

	t.Run("set preset enabled", func(t *testing.T) {
		got, err := d.SetAssistantPresetEnabled(
			t.Context(),
			fixture.bundle.ID,
			fixture.preset.Slug,
			fixture.preset.Version,
			false,
		)
		if err != nil {
			t.Fatalf("SetAssistantPresetEnabled() error: %v", err)
		}
		if got.IsEnabled {
			t.Fatal("preset should be disabled")
		}

		stored, err := d.GetBuiltInAssistantPreset(
			t.Context(),
			fixture.bundle.ID,
			fixture.preset.Slug,
			fixture.preset.Version,
		)
		if err != nil {
			t.Fatalf("GetBuiltInAssistantPreset() error: %v", err)
		}
		if stored.IsEnabled {
			t.Fatal("stored preset should be disabled")
		}
	})
}

func TestResolveBundlesFS(t *testing.T) {
	baseFS := fstest.MapFS{
		"root/file.txt": &fstest.MapFile{Data: []byte("x")},
		"plain.txt":     &fstest.MapFile{Data: []byte("y")},
	}

	tests := []struct {
		name      string
		dir       string
		readPath  string
		wantErr   bool
		wantBytes []byte
	}{
		{
			name:      "empty dir returns original fs",
			dir:       "",
			readPath:  "plain.txt",
			wantBytes: []byte("y"),
		},
		{
			name:      "dot dir returns original fs",
			dir:       ".",
			readPath:  "plain.txt",
			wantBytes: []byte("y"),
		},
		{
			name:      "subdir fs",
			dir:       "root",
			readPath:  "file.txt",
			wantBytes: []byte("x"),
		},
		{
			name:    "missing subdir",
			dir:     "missing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFS, err := fsutil.ResolveFS(baseFS, tt.dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("fsutil.ResolveFS() error: %v", err)
			}

			raw, err := fs.ReadFile(gotFS, tt.readPath)
			if err != nil {
				t.Fatalf("ReadFile() error: %v", err)
			}
			if !bytes.Equal(raw, tt.wantBytes) {
				t.Fatalf("ReadFile() = %q, want %q", string(raw), string(tt.wantBytes))
			}
		})
	}
}

func TestGetAssistantPresetKey(t *testing.T) {
	got := getAssistantPresetKey("bundle-a", "preset-a")
	want := builtInAssistantPresetID("bundle-a::preset-a")

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuiltInData_Close_NilSafe(t *testing.T) {
	var d *BuiltInData
	if err := d.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}
