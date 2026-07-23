package store

import (
	"context"

	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	skillstoreSpec "github.com/flexigpt/flexigpt-app/internal/skillstore/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

type ModelPresetSummary struct {
	IsEnabled bool
}

type ToolSummary struct {
	IsEnabled bool
}

type SkillSummary struct {
	IsEnabled bool
	Insert    skillstoreSpec.SkillInsert

	// HasArguments is used to reject preloaded instruction skills that need runtime input.
	HasArguments bool
	HasResources bool
}

// ModelPresetLookup validates/loads model preset refs without coupling this package
// to a concrete model preset store implementation.
type ModelPresetLookup interface {
	GetModelPresetSummary(
		ctx context.Context,
		ref modelpresetSpec.ModelPresetRef,
	) (ModelPresetSummary, error)
}

// ToolSelectionLookup validates/loads tool selections without coupling this package
// to a concrete tool store implementation.
type ToolSelectionLookup interface {
	GetToolSummaryForSelection(
		ctx context.Context,
		selection toolSpec.ToolSelection,
	) (ToolSummary, error)
}

// SkillLookup validates/loads skill selections without coupling this package
// to a concrete skill store implementation.
type SkillLookup interface {
	GetSkillSummaryForSelection(
		ctx context.Context,
		selection skillstoreSpec.SkillSelection,
	) (SkillSummary, error)
}

// MCPContextLookup validates MCP starter contexts without coupling this package
// to a concrete MCP store/runtime implementation.
//
// Implementations should validate persistent server config strictly, but should
// treat live discovery as best-effort because MCP capabilities are dynamic and
// may require an active connection/auth session.
type MCPContextLookup interface {
	ValidateMCPConversationContext(
		ctx context.Context,
		mcpContext mcpSpec.MCPConversationContext,
	) error
}

type ReferenceLookups struct {
	ModelPresets   ModelPresetLookup
	ToolSelections ToolSelectionLookup
	Skills         SkillLookup
	MCPContext     MCPContextLookup
}
