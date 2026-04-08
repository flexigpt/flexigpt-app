package lookupimpl

import (
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	promptSpec "github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	promptStore "github.com/flexigpt/flexigpt-app/internal/prompt/store"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

func TestNewAssistantPresetReferenceLookups(t *testing.T) {
	got := NewAssistantPresetReferenceLookups(nil, nil, nil, nil)

	if got.ModelPresets == nil {
		t.Fatal("ModelPresets is nil")
	}
	if got.PromptTemplates == nil {
		t.Fatal("PromptTemplates is nil")
	}
	if got.ToolSelections == nil {
		t.Fatal("ToolSelections is nil")
	}
	if got.Skills == nil {
		t.Fatal("Skills is nil")
	}

	if _, ok := got.ModelPresets.(*modelPresetLookupAdapter); !ok {
		t.Fatalf("ModelPresets type = %T, want *modelPresetLookupAdapter", got.ModelPresets)
	}
	if _, ok := got.PromptTemplates.(*promptTemplateLookupAdapter); !ok {
		t.Fatalf("PromptTemplates type = %T, want *promptTemplateLookupAdapter", got.PromptTemplates)
	}
	if _, ok := got.ToolSelections.(*toolSelectionLookupAdapter); !ok {
		t.Fatalf("ToolSelections type = %T, want *toolSelectionLookupAdapter", got.ToolSelections)
	}
	if _, ok := got.Skills.(*skillLookupAdapter); !ok {
		t.Fatalf("Skills type = %T, want *skillLookupAdapter", got.Skills)
	}
}

func TestModelPresetLookupAdapter_GetModelPresetSummary_Errors(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name            string
		adapter         *modelPresetLookupAdapter
		ref             modelpresetSpec.ModelPresetRef
		wantErrContains string
	}{
		{
			name:            "nil receiver",
			adapter:         nil,
			ref:             modelpresetSpec.ModelPresetRef{ProviderName: "p", ModelPresetID: "id"},
			wantErrContains: "not configured",
		},
		{
			name:            "nil store",
			adapter:         &modelPresetLookupAdapter{},
			ref:             modelpresetSpec.ModelPresetRef{ProviderName: "p", ModelPresetID: "id"},
			wantErrContains: "not configured",
		},
		{
			name:            "zero ref",
			adapter:         &modelPresetLookupAdapter{store: &modelpresetStore.ModelPresetStore{}},
			ref:             modelpresetSpec.ModelPresetRef{},
			wantErrContains: "ref is zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.adapter.GetModelPresetSummary(ctx, tt.ref)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestPromptTemplateLookupAdapter_GetPromptTemplateSummary_Errors(t *testing.T) {
	ctx := t.Context()
	validVersion := bundleitemutils.ItemVersion("v1")

	tests := []struct {
		name            string
		adapter         *promptTemplateLookupAdapter
		ref             promptSpec.PromptTemplateRef
		wantErrContains string
	}{
		{
			name:    "nil receiver",
			adapter: nil,
			ref: promptSpec.PromptTemplateRef{
				BundleID:        "bundle-a",
				TemplateSlug:    "tmpl-a",
				TemplateVersion: validVersion,
			},
			wantErrContains: "not configured",
		},
		{
			name:    "nil store",
			adapter: &promptTemplateLookupAdapter{},
			ref: promptSpec.PromptTemplateRef{
				BundleID:        "bundle-a",
				TemplateSlug:    "tmpl-a",
				TemplateVersion: validVersion,
			},
			wantErrContains: "not configured",
		},
		{
			name:            "missing bundle id",
			adapter:         &promptTemplateLookupAdapter{store: &promptStore.PromptTemplateStore{}},
			ref:             promptSpec.PromptTemplateRef{TemplateSlug: "tmpl-a", TemplateVersion: validVersion},
			wantErrContains: "incomplete",
		},
		{
			name:            "missing template slug",
			adapter:         &promptTemplateLookupAdapter{store: &promptStore.PromptTemplateStore{}},
			ref:             promptSpec.PromptTemplateRef{BundleID: "bundle-a", TemplateVersion: validVersion},
			wantErrContains: "incomplete",
		},
		{
			name:            "missing template version",
			adapter:         &promptTemplateLookupAdapter{store: &promptStore.PromptTemplateStore{}},
			ref:             promptSpec.PromptTemplateRef{BundleID: "bundle-a", TemplateSlug: "tmpl-a"},
			wantErrContains: "incomplete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.adapter.GetPromptTemplateSummary(ctx, tt.ref)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestToolSelectionLookupAdapter_GetToolSummaryForSelection_Errors(t *testing.T) {
	ctx := t.Context()
	validVersion := bundleitemutils.ItemVersion("v1")

	tests := []struct {
		name            string
		adapter         *toolSelectionLookupAdapter
		selection       toolSpec.ToolSelection
		wantErrContains string
	}{
		{
			name:    "nil receiver",
			adapter: nil,
			selection: toolSpec.ToolSelection{
				ToolRef: toolSpec.ToolRef{BundleID: "bundle-a", ToolSlug: "tool-a", ToolVersion: validVersion},
			},
			wantErrContains: "not configured",
		},
		{
			name:    "nil store",
			adapter: &toolSelectionLookupAdapter{},
			selection: toolSpec.ToolSelection{
				ToolRef: toolSpec.ToolRef{BundleID: "bundle-a", ToolSlug: "tool-a", ToolVersion: validVersion},
			},
			wantErrContains: "not configured",
		},
		{
			name:    "missing bundle id",
			adapter: &toolSelectionLookupAdapter{store: &toolStore.ToolStore{}},
			selection: toolSpec.ToolSelection{
				ToolRef: toolSpec.ToolRef{ToolSlug: "tool-a", ToolVersion: validVersion},
			},
			wantErrContains: "incomplete",
		},
		{
			name:    "missing tool slug",
			adapter: &toolSelectionLookupAdapter{store: &toolStore.ToolStore{}},
			selection: toolSpec.ToolSelection{
				ToolRef: toolSpec.ToolRef{BundleID: "bundle-a", ToolVersion: validVersion},
			},
			wantErrContains: "incomplete",
		},
		{
			name:    "missing tool version",
			adapter: &toolSelectionLookupAdapter{store: &toolStore.ToolStore{}},
			selection: toolSpec.ToolSelection{
				ToolRef: toolSpec.ToolRef{BundleID: "bundle-a", ToolSlug: "tool-a"},
			},
			wantErrContains: "incomplete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.adapter.GetToolSummaryForSelection(ctx, tt.selection)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestSkillLookupAdapter_GetSkillSummaryForSelection_Errors(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name            string
		adapter         *skillLookupAdapter
		selection       skillSpec.SkillSelection
		wantErrContains string
	}{
		{
			name:    "nil receiver",
			adapter: nil,
			selection: skillSpec.SkillSelection{
				SkillRef: skillSpec.SkillRef{BundleID: "bundle-a", SkillSlug: "skill-a"},
			},
			wantErrContains: "not configured",
		},
		{
			name:    "nil store",
			adapter: &skillLookupAdapter{},
			selection: skillSpec.SkillSelection{
				SkillRef: skillSpec.SkillRef{BundleID: "bundle-a", SkillSlug: "skill-a"},
			},
			wantErrContains: "not configured",
		},
		{
			name:            "missing bundle id",
			adapter:         &skillLookupAdapter{store: &skillStore.SkillStore{}},
			selection:       skillSpec.SkillSelection{SkillRef: skillSpec.SkillRef{SkillSlug: "skill-a"}},
			wantErrContains: "incomplete",
		},
		{
			name:            "missing skill slug",
			adapter:         &skillLookupAdapter{store: &skillStore.SkillStore{}},
			selection:       skillSpec.SkillSelection{SkillRef: skillSpec.SkillRef{BundleID: "bundle-a"}},
			wantErrContains: "incomplete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.adapter.GetSkillSummaryForSelection(ctx, tt.selection)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}
