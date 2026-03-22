package store

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	promptSpec "github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

type invalidJSONValue struct {
	N int
}

func (invalidJSONValue) MarshalJSON() ([]byte, error) {
	return []byte("{"), nil
}

func TestCloneJSONValue(t *testing.T) {
	t.Run("deep copy on marshalable value", func(t *testing.T) {
		type sample struct {
			Values  []int
			Enabled *bool
		}

		flag := true
		in := sample{
			Values:  []int{1, 2, 3},
			Enabled: &flag,
		}

		got := cloneJSONValue(in)

		in.Values[0] = 99
		*in.Enabled = false

		if got.Values[0] != 1 {
			t.Fatalf("got.Values[0] = %d, want 1", got.Values[0])
		}
		if got.Enabled == nil || *got.Enabled != true {
			t.Fatalf("got.Enabled = %v, want true", got.Enabled)
		}
	})

	t.Run("returns input when marshal fails", func(t *testing.T) {
		type sample struct {
			C chan int
		}

		in := sample{C: make(chan int)}
		got := cloneJSONValue(in)

		if got.C != in.C {
			t.Fatal("expected original value to be returned on marshal error")
		}
	})

	t.Run("returns input when unmarshal fails", func(t *testing.T) {
		in := invalidJSONValue{N: 42}
		got := cloneJSONValue(in)

		if !reflect.DeepEqual(got, in) {
			t.Fatalf("got %#v, want %#v", got, in)
		}
	})
}

func TestCloneAssistantPreset(t *testing.T) {
	orig := newTestPreset(t, "clone", true)
	orig.StartingModelPresetRef = &modelpresetSpec.ModelPresetRef{
		ProviderName:  "provider-1",
		ModelPresetID: "mp-1",
	}
	orig.StartingIncludeModelSystemPrompt = boolPtr(true)
	orig.StartingInstructionTemplateRefs = []promptSpec.PromptTemplateRef{
		{
			BundleID:        bundleitemutils.BundleID("bundle-a"),
			TemplateSlug:    "tmpl-a",
			TemplateVersion: testItemVersion(t),
		},
	}
	orig.StartingToolSelections = []toolSpec.ToolSelection{
		{
			ToolRef: toolSpec.ToolRef{
				BundleID:    bundleitemutils.BundleID("bundle-a"),
				ToolSlug:    "tool-a",
				ToolVersion: testItemVersion(t),
			},
		},
	}
	orig.StartingEnabledSkillRefs = []skillSpec.SkillRef{
		{
			BundleID:  bundleitemutils.BundleID("bundle-a"),
			SkillSlug: "skill-a",
			SkillID:   "skill-id-a",
		},
	}

	got := cloneAssistantPreset(orig)

	orig.StartingModelPresetRef.ProviderName = "changed-provider"
	*orig.StartingIncludeModelSystemPrompt = false
	orig.StartingInstructionTemplateRefs[0].TemplateSlug = "changed-template"
	orig.StartingToolSelections[0].ToolRef.ToolSlug = "changed-tool"
	orig.StartingEnabledSkillRefs[0].SkillSlug = "changed-skill"

	if got.StartingModelPresetRef == nil || got.StartingModelPresetRef.ProviderName != "provider-1" {
		t.Fatalf("cloned StartingModelPresetRef = %#v", got.StartingModelPresetRef)
	}
	if got.StartingIncludeModelSystemPrompt == nil || *got.StartingIncludeModelSystemPrompt != true {
		t.Fatalf("cloned StartingIncludeModelSystemPrompt = %v", got.StartingIncludeModelSystemPrompt)
	}
	if got.StartingInstructionTemplateRefs[0].TemplateSlug != "tmpl-a" {
		t.Fatalf("cloned template slug = %q, want %q", got.StartingInstructionTemplateRefs[0].TemplateSlug, "tmpl-a")
	}
	if got.StartingToolSelections[0].ToolRef.ToolSlug != "tool-a" {
		t.Fatalf("cloned tool slug = %q, want %q", got.StartingToolSelections[0].ToolRef.ToolSlug, "tool-a")
	}
	if got.StartingEnabledSkillRefs[0].SkillSlug != "skill-a" {
		t.Fatalf("cloned skill slug = %q, want %q", got.StartingEnabledSkillRefs[0].SkillSlug, "skill-a")
	}
}

func TestCloneAllAssistantPresets(t *testing.T) {
	t.Run("nil source becomes empty map", func(t *testing.T) {
		got := cloneAllAssistantPresets(nil)
		if got == nil {
			t.Fatal("got nil map, want empty non-nil map")
		}
		if len(got) != 0 {
			t.Fatalf("len(got) = %d, want 0", len(got))
		}
	})

	t.Run("deep copies nested maps and preset contents", func(t *testing.T) {
		preset := newTestPreset(t, "all", true)
		preset.StartingIncludeModelSystemPrompt = boolPtr(true)

		src := map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.AssistantPreset{
			bundleitemutils.BundleID("bundle-a"): {
				preset.ID: preset,
			},
		}

		got := cloneAllAssistantPresets(src)

		src[bundleitemutils.BundleID("bundle-a")][preset.ID] = spec.AssistantPreset{}
		*got[bundleitemutils.BundleID("bundle-a")][preset.ID].StartingIncludeModelSystemPrompt = false

		if src[bundleitemutils.BundleID("bundle-a")][preset.ID].ID != "" {
			t.Fatal("expected source mutation to not affect cloned map structure")
		}
	})
}

func TestSlugLocks_LockKey(t *testing.T) {
	l := newSlugLocks()

	t.Run("same key returns same mutex", func(t *testing.T) {
		a := l.lockKey(bundleitemutils.BundleID("bundle-a"), bundleitemutils.ItemSlug("slug-a"))
		b := l.lockKey(bundleitemutils.BundleID("bundle-a"), bundleitemutils.ItemSlug("slug-a"))

		if a != b {
			t.Fatal("same bundleID+slug returned different mutex pointers")
		}
	})

	t.Run("different key returns different mutex", func(t *testing.T) {
		a := l.lockKey(bundleitemutils.BundleID("bundle-a"), bundleitemutils.ItemSlug("slug-a"))
		b := l.lockKey(bundleitemutils.BundleID("bundle-b"), bundleitemutils.ItemSlug("slug-a"))

		if a == b {
			t.Fatal("different keys returned same mutex pointer")
		}
	})
}

func TestSlugLocks_Concurrency(t *testing.T) {
	l := newSlugLocks()

	t.Run("same key blocks until unlock", func(t *testing.T) {
		a := l.lockKey(bundleitemutils.BundleID("bundle-a"), bundleitemutils.ItemSlug("slug-a"))
		b := l.lockKey(bundleitemutils.BundleID("bundle-a"), bundleitemutils.ItemSlug("slug-a"))

		a.Lock()
		acquired := make(chan struct{})

		go func() {
			b.Lock()
			defer b.Unlock()
			close(acquired)
		}()

		select {
		case <-acquired:
			t.Fatal("lock acquired before first lock was released")
		case <-time.After(100 * time.Millisecond):
		}

		a.Unlock()

		select {
		case <-acquired:
		case <-time.After(time.Second):
			t.Fatal("lock was not acquired after release")
		}
	})

	t.Run("different keys do not block each other", func(t *testing.T) {
		a := l.lockKey(bundleitemutils.BundleID("bundle-a"), bundleitemutils.ItemSlug("slug-a"))
		b := l.lockKey(bundleitemutils.BundleID("bundle-b"), bundleitemutils.ItemSlug("slug-a"))

		a.Lock()
		defer a.Unlock()

		acquired := make(chan struct{})

		go func() {
			b.Lock()
			defer b.Unlock()
			close(acquired)
		}()

		select {
		case <-acquired:
		case <-time.After(time.Second):
			t.Fatal("different key should not block")
		}
	})
}

func TestValidateAssistantPresetBundle(t *testing.T) {
	valid := newTestBundle(t, "valid", true)

	tests := []struct {
		name            string
		bundle          *spec.AssistantPresetBundle
		wantErrContains string
	}{
		{
			name:            "nil bundle",
			bundle:          nil,
			wantErrContains: "bundle is nil",
		},
		{
			name: "schema mismatch",
			bundle: func() *spec.AssistantPresetBundle {
				b := valid
				b.SchemaVersion = "wrong"
				return &b
			}(),
			wantErrContains: "schemaVersion",
		},
		{
			name: "empty id",
			bundle: func() *spec.AssistantPresetBundle {
				b := valid
				b.ID = ""
				return &b
			}(),
			wantErrContains: "bundle id is empty",
		},
		{
			name: "invalid slug",
			bundle: func() *spec.AssistantPresetBundle {
				b := valid
				b.Slug = "Bad Slug!"
				return &b
			}(),
			wantErrContains: "invalid bundle slug",
		},
		{
			name: "empty display name",
			bundle: func() *spec.AssistantPresetBundle {
				b := valid
				b.DisplayName = "   "
				return &b
			}(),
			wantErrContains: "bundle displayName is empty",
		},
		{
			name: "zero createdAt",
			bundle: func() *spec.AssistantPresetBundle {
				b := valid
				b.CreatedAt = time.Time{}
				return &b
			}(),
			wantErrContains: "bundle createdAt is zero",
		},
		{
			name: "zero modifiedAt",
			bundle: func() *spec.AssistantPresetBundle {
				b := valid
				b.ModifiedAt = time.Time{}
				return &b
			}(),
			wantErrContains: "bundle modifiedAt is zero",
		},
		{
			name:   "valid",
			bundle: &valid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAssistantPresetBundle(tt.bundle)

			if tt.wantErrContains == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErrContains)
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestValidateAssistantPresetStructure(t *testing.T) {
	valid := newTestPreset(t, "structure", true)

	dupPromptRef := promptSpec.PromptTemplateRef{
		BundleID:        bundleitemutils.BundleID("bundle-a"),
		TemplateSlug:    "tmpl-a",
		TemplateVersion: testItemVersion(t),
	}
	dupToolSelection := toolSpec.ToolSelection{
		ToolRef: toolSpec.ToolRef{
			BundleID:    bundleitemutils.BundleID("bundle-a"),
			ToolSlug:    "tool-a",
			ToolVersion: testItemVersion(t),
		},
	}
	dupSkillRef := skillSpec.SkillRef{
		BundleID:  bundleitemutils.BundleID("bundle-a"),
		SkillSlug: "skill-a",
		SkillID:   "skill-id-a",
	}

	tests := []struct {
		name            string
		preset          *spec.AssistantPreset
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:      "nil preset",
			preset:    nil,
			wantErrIs: spec.ErrNilAssistantPreset,
		},
		{
			name: "schema mismatch",
			preset: func() *spec.AssistantPreset {
				p := valid
				p.SchemaVersion = "wrong"
				return &p
			}(),
			wantErrContains: "schemaVersion",
		},
		{
			name: "empty id",
			preset: func() *spec.AssistantPreset {
				p := valid
				p.ID = ""
				return &p
			}(),
			wantErrContains: "assistant preset id is empty",
		},
		{
			name: "invalid slug",
			preset: func() *spec.AssistantPreset {
				p := valid
				p.Slug = "Bad Slug!"
				return &p
			}(),
			wantErrContains: "invalid assistant preset slug",
		},
		{
			name: "invalid version",
			preset: func() *spec.AssistantPreset {
				p := valid
				p.Version = "bad version!"
				return &p
			}(),
			wantErrContains: "invalid assistant preset version",
		},
		{
			name: "empty display name",
			preset: func() *spec.AssistantPreset {
				p := valid
				p.DisplayName = "   "
				return &p
			}(),
			wantErrContains: "displayName is empty",
		},
		{
			name: "zero createdAt",
			preset: func() *spec.AssistantPreset {
				p := valid
				p.CreatedAt = time.Time{}
				return &p
			}(),
			wantErrContains: "createdAt is zero",
		},
		{
			name: "zero modifiedAt",
			preset: func() *spec.AssistantPreset {
				p := valid
				p.ModifiedAt = time.Time{}
				return &p
			}(),
			wantErrContains: "modifiedAt is zero",
		},
		{
			name: "duplicate instruction refs",
			preset: func() *spec.AssistantPreset {
				p := valid
				p.StartingInstructionTemplateRefs = []promptSpec.PromptTemplateRef{dupPromptRef, dupPromptRef}
				return &p
			}(),
			wantErrContains: "startingInstructionTemplateRefs[1]: duplicate ref",
		},
		{
			name: "duplicate tool refs",
			preset: func() *spec.AssistantPreset {
				p := valid
				p.StartingToolSelections = []toolSpec.ToolSelection{dupToolSelection, dupToolSelection}
				return &p
			}(),
			wantErrContains: "startingToolSelections[1]: duplicate toolRef",
		},
		{
			name: "duplicate skill refs",
			preset: func() *spec.AssistantPreset {
				p := valid
				p.StartingEnabledSkillRefs = []skillSpec.SkillRef{dupSkillRef, dupSkillRef}
				return &p
			}(),
			wantErrContains: "startingEnabledSkillRefs[1]: duplicate ref",
		},
		{
			name:   "valid",
			preset: &valid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAssistantPresetStructure(tt.preset)

			if tt.wantErrIs == nil && tt.wantErrContains == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("error = %v, want errors.Is(..., %v)", err, tt.wantErrIs)
			}
			if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestValidateAssistantPresetReferences(t *testing.T) {
	ctx := t.Context()
	version := testItemVersion(t)

	modelRef := modelpresetSpec.ModelPresetRef{
		ProviderName:  "provider-a",
		ModelPresetID: "mp-a",
	}
	promptRef1 := promptSpec.PromptTemplateRef{
		BundleID:        bundleitemutils.BundleID("bundle-a"),
		TemplateSlug:    "tmpl-a",
		TemplateVersion: version,
	}
	promptRef2 := promptSpec.PromptTemplateRef{
		BundleID:        bundleitemutils.BundleID("bundle-b"),
		TemplateSlug:    "tmpl-b",
		TemplateVersion: version,
	}
	toolSel := toolSpec.ToolSelection{
		ToolRef: toolSpec.ToolRef{
			BundleID:    bundleitemutils.BundleID("bundle-a"),
			ToolSlug:    "tool-a",
			ToolVersion: version,
		},
	}
	skillRef := skillSpec.SkillRef{
		BundleID:  bundleitemutils.BundleID("bundle-a"),
		SkillSlug: "skill-a",
		SkillID:   "skill-id-a",
	}

	makeBase := func() *spec.AssistantPreset {
		p := newTestPreset(t, "refs", true)
		return &p
	}

	tests := []struct {
		name            string
		preset          *spec.AssistantPreset
		lookups         ReferenceLookups
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:      "nil preset",
			preset:    nil,
			wantErrIs: spec.ErrNilAssistantPreset,
		},
		{
			name: "model lookup missing",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingModelPresetRef = &modelRef
				return p
			}(),
			wantErrContains: "model preset lookup not configured",
		},
		{
			name: "model lookup error",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingModelPresetRef = &modelRef
				return p
			}(),
			lookups: ReferenceLookups{
				ModelPresets: fakeModelPresetLookup(
					func(context.Context, modelpresetSpec.ModelPresetRef) (ModelPresetSummary, error) {
						return ModelPresetSummary{}, errors.New("lookup boom")
					},
				),
			},
			wantErrContains: "startingModelPresetRef: lookup boom",
		},
		{
			name: "model disabled",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingModelPresetRef = &modelRef
				return p
			}(),
			lookups: ReferenceLookups{
				ModelPresets: fakeModelPresetLookup(
					func(context.Context, modelpresetSpec.ModelPresetRef) (ModelPresetSummary, error) {
						return ModelPresetSummary{IsEnabled: false}, nil
					},
				),
			},
			wantErrContains: "startingModelPresetRef references a disabled model preset",
		},
		{
			name: "prompt lookup missing",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingInstructionTemplateRefs = []promptSpec.PromptTemplateRef{promptRef1}
				return p
			}(),
			wantErrContains: "prompt template lookup not configured",
		},
		{
			name: "prompt lookup error",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingInstructionTemplateRefs = []promptSpec.PromptTemplateRef{promptRef1}
				return p
			}(),
			lookups: ReferenceLookups{
				PromptTemplates: fakePromptTemplateLookup(
					func(context.Context, promptSpec.PromptTemplateRef) (PromptTemplateSummary, error) {
						return PromptTemplateSummary{}, errors.New("prompt boom")
					},
				),
			},
			wantErrContains: "startingInstructionTemplateRefs[0]: prompt boom",
		},
		{
			name: "prompt disabled",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingInstructionTemplateRefs = []promptSpec.PromptTemplateRef{promptRef1}
				return p
			}(),
			lookups: ReferenceLookups{
				PromptTemplates: fakePromptTemplateLookup(
					func(context.Context, promptSpec.PromptTemplateRef) (PromptTemplateSummary, error) {
						return PromptTemplateSummary{
							IsEnabled:  false,
							Kind:       promptSpec.PromptTemplateKindInstructionsOnly,
							IsResolved: true,
						}, nil
					},
				),
			},
			wantErrContains: "startingInstructionTemplateRefs[0]: referenced template is disabled",
		},
		{
			name: "prompt wrong kind",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingInstructionTemplateRefs = []promptSpec.PromptTemplateRef{promptRef1}
				return p
			}(),
			lookups: ReferenceLookups{
				PromptTemplates: fakePromptTemplateLookup(
					func(context.Context, promptSpec.PromptTemplateRef) (PromptTemplateSummary, error) {
						return PromptTemplateSummary{
							IsEnabled:  true,
							Kind:       "not-instructions-only",
							IsResolved: true,
						}, nil
					},
				),
			},
			wantErrContains: "template kind must be",
		},
		{
			name: "prompt unresolved",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingInstructionTemplateRefs = []promptSpec.PromptTemplateRef{promptRef1}
				return p
			}(),
			lookups: ReferenceLookups{
				PromptTemplates: fakePromptTemplateLookup(
					func(context.Context, promptSpec.PromptTemplateRef) (PromptTemplateSummary, error) {
						return PromptTemplateSummary{
							IsEnabled:  true,
							Kind:       promptSpec.PromptTemplateKindInstructionsOnly,
							IsResolved: false,
						}, nil
					},
				),
			},
			wantErrContains: "template must be resolved",
		},
		{
			name: "prompt second item index in error",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingInstructionTemplateRefs = []promptSpec.PromptTemplateRef{promptRef1, promptRef2}
				return p
			}(),
			lookups: func() ReferenceLookups {
				calls := 0
				return ReferenceLookups{
					PromptTemplates: fakePromptTemplateLookup(
						func(context.Context, promptSpec.PromptTemplateRef) (PromptTemplateSummary, error) {
							calls++
							if calls == 1 {
								return PromptTemplateSummary{
									IsEnabled:  true,
									Kind:       promptSpec.PromptTemplateKindInstructionsOnly,
									IsResolved: true,
								}, nil
							}
							return PromptTemplateSummary{
								IsEnabled:  false,
								Kind:       promptSpec.PromptTemplateKindInstructionsOnly,
								IsResolved: true,
							}, nil
						},
					),
				}
			}(),
			wantErrContains: "startingInstructionTemplateRefs[1]",
		},
		{
			name: "tool lookup missing",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingToolSelections = []toolSpec.ToolSelection{toolSel}
				return p
			}(),
			wantErrContains: "tool selection lookup not configured",
		},
		{
			name: "tool lookup error",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingToolSelections = []toolSpec.ToolSelection{toolSel}
				return p
			}(),
			lookups: ReferenceLookups{
				ToolSelections: fakeToolSelectionLookup(
					func(context.Context, toolSpec.ToolSelection) (ToolSummary, error) {
						return ToolSummary{}, errors.New("tool boom")
					},
				),
			},
			wantErrContains: "startingToolSelections[0]: tool boom",
		},
		{
			name: "tool disabled",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingToolSelections = []toolSpec.ToolSelection{toolSel}
				return p
			}(),
			lookups: ReferenceLookups{
				ToolSelections: fakeToolSelectionLookup(
					func(context.Context, toolSpec.ToolSelection) (ToolSummary, error) {
						return ToolSummary{IsEnabled: false}, nil
					},
				),
			},
			wantErrContains: "startingToolSelections[0]: referenced tool is disabled",
		},
		{
			name: "skill lookup missing",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingEnabledSkillRefs = []skillSpec.SkillRef{skillRef}
				return p
			}(),
			wantErrContains: "skill lookup not configured",
		},
		{
			name: "skill lookup error",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingEnabledSkillRefs = []skillSpec.SkillRef{skillRef}
				return p
			}(),
			lookups: ReferenceLookups{
				Skills: fakeSkillLookup(func(context.Context, skillSpec.SkillRef) (SkillSummary, error) {
					return SkillSummary{}, errors.New("skill boom")
				}),
			},
			wantErrContains: "startingEnabledSkillRefs[0]: skill boom",
		},
		{
			name: "skill disabled",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingEnabledSkillRefs = []skillSpec.SkillRef{skillRef}
				return p
			}(),
			lookups: ReferenceLookups{
				Skills: fakeSkillLookup(func(context.Context, skillSpec.SkillRef) (SkillSummary, error) {
					return SkillSummary{IsEnabled: false}, nil
				}),
			},
			wantErrContains: "startingEnabledSkillRefs[0]: referenced skill is disabled",
		},
		{
			name: "all references valid",
			preset: func() *spec.AssistantPreset {
				p := makeBase()
				p.StartingModelPresetRef = &modelRef
				p.StartingInstructionTemplateRefs = []promptSpec.PromptTemplateRef{promptRef1}
				p.StartingToolSelections = []toolSpec.ToolSelection{toolSel}
				p.StartingEnabledSkillRefs = []skillSpec.SkillRef{skillRef}
				return p
			}(),
			lookups: ReferenceLookups{
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
				ToolSelections: fakeToolSelectionLookup(
					func(context.Context, toolSpec.ToolSelection) (ToolSummary, error) {
						return ToolSummary{IsEnabled: true}, nil
					},
				),
				Skills: fakeSkillLookup(func(context.Context, skillSpec.SkillRef) (SkillSummary, error) {
					return SkillSummary{IsEnabled: true}, nil
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAssistantPresetReferences(ctx, tt.preset, tt.lookups)

			if tt.wantErrIs == nil && tt.wantErrContains == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("error = %v, want errors.Is(..., %v)", err, tt.wantErrIs)
			}
			if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestValidateStartingModelPresetPatch_Nil(t *testing.T) {
	if err := validateStartingModelPresetPatch(nil); err != nil {
		t.Fatalf("validateStartingModelPresetPatch(nil) error: %v", err)
	}
}

func TestJSONHelpers(t *testing.T) {
	t.Run("jsonFieldPresentAndNonNull", func(t *testing.T) {
		tests := []struct {
			name      string
			value     any
			field     string
			want      bool
			expectErr bool
		}{
			{
				name:  "field absent",
				value: map[string]any{"other": "x"},
				field: "field",
				want:  false,
			},
			{
				name:  "field present null",
				value: map[string]any{"field": nil},
				field: "field",
				want:  false,
			},
			{
				name:  "field present non-null",
				value: map[string]any{"field": "x"},
				field: "field",
				want:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := jsonFieldPresentAndNonNull(tt.value, tt.field)
				if tt.expectErr {
					if err == nil {
						t.Fatal("expected error, got nil")
					}
					return
				}
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("normalizedJSONKey", func(t *testing.T) {
		a := map[string]any{"b": 2, "a": 1}
		b := map[string]any{"a": 1, "b": 2}

		keyA, err := normalizedJSONKey(a)
		if err != nil {
			t.Fatalf("normalizedJSONKey(a) error: %v", err)
		}
		keyB, err := normalizedJSONKey(b)
		if err != nil {
			t.Fatalf("normalizedJSONKey(b) error: %v", err)
		}

		if keyA != keyB {
			t.Fatalf("normalizedJSONKey mismatch: %q != %q", keyA, keyB)
		}
	})

	t.Run("toolSelectionRefKey stable for same toolRef", func(t *testing.T) {
		s1 := toolSpec.ToolSelection{
			ToolRef: toolSpec.ToolRef{
				BundleID:    bundleitemutils.BundleID("bundle-a"),
				ToolSlug:    "tool-a",
				ToolVersion: testItemVersion(t),
			},
		}
		s2 := toolSpec.ToolSelection{
			ToolRef: toolSpec.ToolRef{
				BundleID:    bundleitemutils.BundleID("bundle-a"),
				ToolSlug:    "tool-a",
				ToolVersion: testItemVersion(t),
			},
		}

		k1, err := toolSelectionRefKey(s1)
		if err != nil {
			t.Fatalf("toolSelectionRefKey(s1) error: %v", err)
		}
		k2, err := toolSelectionRefKey(s2)
		if err != nil {
			t.Fatalf("toolSelectionRefKey(s2) error: %v", err)
		}

		if k1 == "" {
			t.Fatal("empty key")
		}
		if k1 != k2 {
			t.Fatalf("keys differ: %q != %q", k1, k2)
		}
	})
}
