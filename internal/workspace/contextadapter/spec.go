package contextadapter

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

const workspaceContextSchemaVersionV1 = "v1"

const (
	contextKind      artifactstore.ArtifactKind = "workspace.context"
	contextSchemaID  artifactstore.SchemaID     = "workspace.context.v1"
	contextDecoderID artifactstore.DecoderID    = "workspace.context-markdown"
)

const (
	agentsFileName                       = "AGENTS.md"
	claudeFileName                       = "CLAUDE.md"
	readmeFileName                       = "README.md"
	agentsLocator  artifactstore.Locator = agentsFileName
	claudeLocator  artifactstore.Locator = claudeFileName
	readmeLocator  artifactstore.Locator = readmeFileName
)

const (
	contextRoleAgentInstructions     = "agent-instructions"
	contextRoleAssistantInstructions = "assistant-instructions"
	contextRoleProjectReadme         = "project-readme"
	contextRoleProjectContext        = "project-context"
	contextRoleLabelKey              = "context.role"
	contextMarkdownMediaType         = "text/markdown"

	contextPromptSeparator   = "\n\n"
	contextPromptStartFormat = "<<<WORKSPACE_CONTEXT name=%q role=%q source=%q>>>\n"
	contextPromptEndMarker   = "\n<<<END_WORKSPACE_CONTEXT>>>"
)

type contextFileSupport struct {
	fileName string
	role     string
}

var contextFileSupportMatrix = []contextFileSupport{
	{
		fileName: agentsFileName,
		role:     contextRoleAgentInstructions,
	},
	{
		fileName: claudeFileName,
		role:     contextRoleAssistantInstructions,
	},
	{
		fileName: readmeFileName,
		role:     contextRoleProjectReadme,
	},
}

var artifactSupport = engine.ArtifactSupport{
	Kind:      contextKind,
	SchemaID:  contextSchemaID,
	DecoderID: contextDecoderID,
}

type contextDefinition struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	MediaType string `json:"mediaType"`
	Content   string `json:"content"`
}
