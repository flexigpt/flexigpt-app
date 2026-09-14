package codec

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/collectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/contextv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/instructionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/loopv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/skillv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/teamv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workflowv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workspacev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

var orderedSchemaKeys = []schema.Key{
	instructionv1.InstructionSchemaKey,
	contextv1.ContextSchemaKey,
	toolv1.ToolSchemaKey,
	modelv1.ModelSchemaKey,
	skillv1.SkillSchemaKey,
	mcpv1.MCPSchemaKey,
	mcppolicyv1.MCPPolicySchemaKey,
	collectionv1.CollectionSchemaKey,
	agentv1.AgentSchemaKey,
	teamv1.TeamSchemaKey,
	loopv1.LoopSchemaKey,
	workflowv1.WorkflowSchemaKey,
	workspacev1.WorkspaceSchemaKey,
}

var schemaKeysByType = map[declaration.Type]schema.Key{
	declaration.TypeInstruction: instructionv1.InstructionSchemaKey,
	declaration.TypeContext:     contextv1.ContextSchemaKey,
	declaration.TypeTool:        toolv1.ToolSchemaKey,
	declaration.TypeModel:       modelv1.ModelSchemaKey,
	declaration.TypeSkill:       skillv1.SkillSchemaKey,
	declaration.TypeMCP:         mcpv1.MCPSchemaKey,
	declaration.TypeMCPPolicy:   mcppolicyv1.MCPPolicySchemaKey,
	declaration.TypeCollection:  collectionv1.CollectionSchemaKey,
	declaration.TypeAgent:       agentv1.AgentSchemaKey,
	declaration.TypeTeam:        teamv1.TeamSchemaKey,
	declaration.TypeLoop:        loopv1.LoopSchemaKey,
	declaration.TypeWorkflow:    workflowv1.WorkflowSchemaKey,
	declaration.TypeWorkspace:   workspacev1.WorkspaceSchemaKey,
}

func NewInstructionV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		instructionv1.InstructionSchemaKey,
		instructionv1.InstructionJSONSchema(),
	)
}

func NewContextV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		contextv1.ContextSchemaKey,
		contextv1.ContextJSONSchema(),
	)
}

func NewToolV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		toolv1.ToolSchemaKey,
		toolv1.ToolJSONSchema(),
	)
}

func NewModelV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		modelv1.ModelSchemaKey,
		modelv1.ModelJSONSchema(),
	)
}

func NewSkillV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		skillv1.SkillSchemaKey,
		skillv1.SkillJSONSchema(),
	)
}

func NewMCPV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		mcpv1.MCPSchemaKey,
		mcpv1.MCPJSONSchema(),
	)
}

func NewMCPPolicyV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		mcppolicyv1.MCPPolicySchemaKey,
		mcppolicyv1.MCPPolicyJSONSchema(),
	)
}

func NewCollectionV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		collectionv1.CollectionSchemaKey,
		collectionv1.CollectionJSONSchema(),
	)
}

func NewAgentV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		agentv1.AgentSchemaKey,
		agentv1.AgentJSONSchema(),
	)
}

func NewTeamV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		teamv1.TeamSchemaKey,
		teamv1.TeamJSONSchema(),
	)
}

func NewLoopV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		loopv1.LoopSchemaKey,
		loopv1.LoopJSONSchema(),
	)
}

func NewWorkflowV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		workflowv1.WorkflowSchemaKey,
		workflowv1.WorkflowJSONSchema(),
	)
}

func NewWorkspaceV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		workspacev1.WorkspaceSchemaKey,
		workspacev1.WorkspaceJSONSchema(),
	)
}

func AllSchemaCodecs() []providerapi.SchemaCodec {
	return []providerapi.SchemaCodec{
		NewInstructionV1SchemaCodec(),
		NewContextV1SchemaCodec(),
		NewToolV1SchemaCodec(),
		NewModelV1SchemaCodec(),
		NewSkillV1SchemaCodec(),
		NewMCPV1SchemaCodec(),
		NewMCPPolicyV1SchemaCodec(),
		NewCollectionV1SchemaCodec(),
		NewAgentV1SchemaCodec(),
		NewTeamV1SchemaCodec(),
		NewLoopV1SchemaCodec(),
		NewWorkflowV1SchemaCodec(),
		NewWorkspaceV1SchemaCodec(),
	}
}

func SchemaKeys() []schema.Key {
	return append([]schema.Key(nil), orderedSchemaKeys...)
}

func SchemaKeyForType(
	declarationType declaration.Type,
) (schema.Key, bool) {
	key, found := schemaKeysByType[declarationType]
	return key, found
}
