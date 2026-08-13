package contextadapter

import (
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

const workspaceContextSchemaVersionV1 = "v1"

const (
	contextKind      basespec.ArtifactKind = "workspace.context"
	contextSchemaID  basespec.SchemaID     = "workspace.context.v1"
	contextDecoderID basespec.DecoderID    = "workspace.context-markdown"
)

const (
	contextRoleAgentInstructions     = builtinSchema.WorkspaceContextRoleAgentInstructions
	contextRoleAssistantInstructions = builtinSchema.WorkspaceContextRoleAssistantInstructions
	contextRoleProjectReadme         = builtinSchema.WorkspaceContextRoleProjectReadme
	contextRoleProjectContext        = "project-context"
	contextRoleLabelKey              = "context.role"
	contextMarkdownMediaType         = "text/markdown"

	contextPreferenceNone          = ""
	contextPreferenceIncludeReadme = builtinSchema.WorkspaceContextPreferenceIncludeReadme

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

var contextConventionRegistry = func() []contextFileSupport {
	input := builtinSchema.WorkspaceContextFileConventions()
	output := make([]contextFileSupport, 0, len(input))
	for _, value := range input {
		output = append(output, contextFileSupport{
			FileName:         string(value.FileName),
			Role:             value.Role,
			DefaultDiscovery: value.DefaultDiscovery,
			Preference:       value.Preference,
			RuntimeOrder:     value.RuntimeOrder,
		})
	}
	return output
}()

func contextConventionFor(
	locator basespec.Locator,
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

var artifactSupport = spec.ArtifactSupport{
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
