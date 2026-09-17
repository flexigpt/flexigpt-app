package codec

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/loopv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/skillv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/teamv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workflowv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workspacev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

var orderedSchemaKeys = []schema.Key{
	textv1.TextSchemaKey,
	modelv1.ModelSchemaKey,
	toolv1.ToolSchemaKey,
	skillv1.SkillSchemaKey,
	mcpv1.MCPSchemaKey,
	mcppolicyv1.MCPPolicySchemaKey,
	pluginv1.PluginSchemaKey,
	agentv1.AgentSchemaKey,
	teamv1.TeamSchemaKey,
	loopv1.LoopSchemaKey,
	workflowv1.WorkflowSchemaKey,
	workspacev1.WorkspaceSchemaKey,
}

var schemaKeysByType = map[declaration.Type]schema.Key{
	declaration.TypeText:      textv1.TextSchemaKey,
	declaration.TypeModel:     modelv1.ModelSchemaKey,
	declaration.TypeTool:      toolv1.ToolSchemaKey,
	declaration.TypeSkill:     skillv1.SkillSchemaKey,
	declaration.TypeMCP:       mcpv1.MCPSchemaKey,
	declaration.TypeMCPPolicy: mcppolicyv1.MCPPolicySchemaKey,
	declaration.TypePlugin:    pluginv1.PluginSchemaKey,
	declaration.TypeAgent:     agentv1.AgentSchemaKey,
	declaration.TypeTeam:      teamv1.TeamSchemaKey,
	declaration.TypeLoop:      loopv1.LoopSchemaKey,
	declaration.TypeWorkflow:  workflowv1.WorkflowSchemaKey,
	declaration.TypeWorkspace: workspacev1.WorkspaceSchemaKey,
}

func NewTextV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		textv1.TextSchemaKey,
		textv1.TextJSONSchema(),
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

func NewPluginV1SchemaCodec() providerapi.SchemaCodec {
	return NewPassthrough(
		pluginv1.PluginSchemaKey,
		pluginv1.PluginJSONSchema(),
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
		NewTextV1SchemaCodec(),
		NewModelV1SchemaCodec(),
		NewToolV1SchemaCodec(),
		NewSkillV1SchemaCodec(),
		NewMCPV1SchemaCodec(),
		NewMCPPolicyV1SchemaCodec(),
		NewPluginV1SchemaCodec(),
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
