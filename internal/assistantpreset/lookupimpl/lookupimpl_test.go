package lookupimpl

import (
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

const (
	testMissingBundleID = "missing bundle id"
	testNilReceiver     = "nil receiver"
	testNilStore        = "nil store"

	testErrNotConfigured = "not configured"
	testErrIncomplete    = "incomplete"

	testBundleIDA     = "bundle-a"
	testTemplateSlugA = "tmpl-a"
	testToolA         = "tool-a"
	testSkillA        = "skill-a"
)

func TestNewAssistantPresetReferenceLookups(t *testing.T) {
	got := NewAssistantPresetReferenceLookups(nil, nil, nil, nil, nil)

	if got.ModelPresets == nil {
		t.Fatal("ModelPresets is nil")
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
			name:            testNilReceiver,
			adapter:         nil,
			ref:             modelpresetSpec.ModelPresetRef{ProviderName: "p", ModelPresetID: "id"},
			wantErrContains: testErrNotConfigured,
		},
		{
			name:            testNilStore,
			adapter:         &modelPresetLookupAdapter{},
			ref:             modelpresetSpec.ModelPresetRef{ProviderName: "p", ModelPresetID: "id"},
			wantErrContains: testErrNotConfigured,
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
			name:    testNilReceiver,
			adapter: nil,
			selection: toolSpec.ToolSelection{
				ToolRef: toolSpec.ToolRef{BundleID: testBundleIDA, ToolSlug: testToolA, ToolVersion: validVersion},
			},
			wantErrContains: testErrNotConfigured,
		},
		{
			name:    testNilStore,
			adapter: &toolSelectionLookupAdapter{},
			selection: toolSpec.ToolSelection{
				ToolRef: toolSpec.ToolRef{BundleID: testBundleIDA, ToolSlug: testToolA, ToolVersion: validVersion},
			},
			wantErrContains: testErrNotConfigured,
		},
		{
			name:    testMissingBundleID,
			adapter: &toolSelectionLookupAdapter{store: &toolStore.ToolStore{}},
			selection: toolSpec.ToolSelection{
				ToolRef: toolSpec.ToolRef{ToolSlug: testToolA, ToolVersion: validVersion},
			},
			wantErrContains: testErrIncomplete,
		},
		{
			name:    "missing tool slug",
			adapter: &toolSelectionLookupAdapter{store: &toolStore.ToolStore{}},
			selection: toolSpec.ToolSelection{
				ToolRef: toolSpec.ToolRef{BundleID: testBundleIDA, ToolVersion: validVersion},
			},
			wantErrContains: testErrIncomplete,
		},
		{
			name:    "missing tool version",
			adapter: &toolSelectionLookupAdapter{store: &toolStore.ToolStore{}},
			selection: toolSpec.ToolSelection{
				ToolRef: toolSpec.ToolRef{BundleID: testBundleIDA, ToolSlug: testToolA},
			},
			wantErrContains: testErrIncomplete,
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
			name:    testNilReceiver,
			adapter: nil,
			selection: skillSpec.SkillSelection{
				SkillRef: skillSpec.SkillRef{BundleID: testBundleIDA, SkillSlug: testSkillA},
			},
			wantErrContains: testErrNotConfigured,
		},
		{
			name:    testNilStore,
			adapter: &skillLookupAdapter{},
			selection: skillSpec.SkillSelection{
				SkillRef: skillSpec.SkillRef{BundleID: testBundleIDA, SkillSlug: testSkillA},
			},
			wantErrContains: testErrNotConfigured,
		},
		{
			name:            testMissingBundleID,
			adapter:         &skillLookupAdapter{store: &skillStore.SkillStore{}},
			selection:       skillSpec.SkillSelection{SkillRef: skillSpec.SkillRef{SkillSlug: testSkillA}},
			wantErrContains: testErrIncomplete,
		},
		{
			name:            "missing skill slug",
			adapter:         &skillLookupAdapter{store: &skillStore.SkillStore{}},
			selection:       skillSpec.SkillSelection{SkillRef: skillSpec.SkillRef{BundleID: testBundleIDA}},
			wantErrContains: testErrIncomplete,
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
