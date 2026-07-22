package contextadapter

import (
	"strings"

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
	agentsFileName = "AGENTS.md"
	claudeFileName = "CLAUDE.md"
	readmeFileName = "README.md"
)

const (
	contextRoleAgentInstructions     = "agent-instructions"
	contextRoleAssistantInstructions = "assistant-instructions"
	contextRoleProjectReadme         = "project-readme"
	contextRoleProjectContext        = "project-context"
	contextRoleLabelKey              = "context.role"
	contextMarkdownMediaType         = "text/markdown"

	contextPreferenceNone          = ""
	contextPreferenceIncludeReadme = "include-readme"

	contextPromptSeparator   = "\n\n"
	contextPromptStartFormat = "<<<WORKSPACE_CONTEXT name=%q role=%q source=%q>>>\n"
	contextPromptEndMarker   = "\n<<<END_WORKSPACE_CONTEXT>>>"
)

type contextFileSupport struct {
	FileName         string
	Role             string
	DefaultDiscovery bool
	Preference       string
	RuntimeOrder     int
}

var contextConventionRegistry = []contextFileSupport{
	{
		FileName:         agentsFileName,
		Role:             contextRoleAgentInstructions,
		DefaultDiscovery: true,
		RuntimeOrder:     100,
	},
	{
		FileName:         claudeFileName,
		Role:             contextRoleAssistantInstructions,
		DefaultDiscovery: true,
		RuntimeOrder:     200,
	},
	{
		FileName:     readmeFileName,
		Role:         contextRoleProjectReadme,
		Preference:   contextPreferenceIncludeReadme,
		RuntimeOrder: 300,
	},
}

func contextConventionFor(
	locator artifactstore.Locator,
) (contextFileSupport, bool) {
	value := string(locator)
	for _, convention := range contextConventionRegistry {
		if strings.EqualFold(value, convention.FileName) {
			return convention, true
		}
	}
	return contextFileSupport{}, false
}

func supportedContextRole(role string) bool {
	switch role {
	case contextRoleAgentInstructions,
		contextRoleAssistantInstructions,
		contextRoleProjectReadme,
		contextRoleProjectContext:
		return true
	default:
		return false
	}
}

var artifactSupport = engine.ArtifactSupport{
	Kind:      contextKind,
	SchemaID:  contextSchemaID,
	DecoderID: contextDecoderID,
	Validator: ValidateContextDefinition,
}

type contextDefinition struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	MediaType string `json:"mediaType"`
	Content   string `json:"content"`
}
