package store

import (
	"context"

	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	promptSpec "github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

type ModelPresetSummary struct {
	IsEnabled bool
}

type PromptTemplateSummary struct {
	IsEnabled  bool
	Kind       promptSpec.PromptTemplateKind
	IsResolved bool
}

type ToolSummary struct {
	IsEnabled bool
}

type SkillSummary struct {
	IsEnabled bool
}

// ModelPresetLookup validates/loads model preset refs without coupling this package
// to a concrete model preset store implementation.
type ModelPresetLookup interface {
	GetModelPresetSummary(
		ctx context.Context,
		ref modelpresetSpec.ModelPresetRef,
	) (ModelPresetSummary, error)
}

// PromptTemplateLookup validates/loads prompt template refs without coupling this package
// to a concrete prompt store implementation.
type PromptTemplateLookup interface {
	GetPromptTemplateSummary(
		ctx context.Context,
		ref promptSpec.PromptTemplateRef,
	) (PromptTemplateSummary, error)
}

// ToolSelectionLookup validates/loads tool selections without coupling this package
// to a concrete tool store implementation.
type ToolSelectionLookup interface {
	GetToolSummaryForSelection(
		ctx context.Context,
		selection toolSpec.ToolSelection,
	) (ToolSummary, error)
}

// SkillLookup validates/loads skill refs without coupling this package
// to a concrete skill store implementation.
type SkillLookup interface {
	GetSkillSummary(
		ctx context.Context,
		ref skillSpec.SkillRef,
	) (SkillSummary, error)
}

type ReferenceLookups struct {
	ModelPresets    ModelPresetLookup
	PromptTemplates PromptTemplateLookup
	ToolSelections  ToolSelectionLookup
	Skills          SkillLookup
}
