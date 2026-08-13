package contextadapter

import (
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

const (
	contextPromptSeparator   = "\n\n"
	contextPromptStartFormat = "<<<WORKSPACE_CONTEXT name=%q role=%q source=%q>>>\n"
	contextPromptEndMarker   = "\n<<<END_WORKSPACE_CONTEXT>>>"
)

type contextFileSupport struct {
	FileName         string
	Role             artifactbuiltin.WorkspaceContextRole
	DefaultDiscovery bool
	Preference       artifactbuiltin.WorkspaceContextPreference
	RuntimeOrder     int
}

var contextConventionRegistry = func() []contextFileSupport {
	input := artifactbuiltin.WorkspaceContextFileConventions()
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

func supportedContextRole(role artifactbuiltin.WorkspaceContextRole) bool {
	switch role {
	case artifactbuiltin.WorkspaceContextRoleAgentInstructions,
		artifactbuiltin.WorkspaceContextRoleAssistantInstructions,
		artifactbuiltin.WorkspaceContextRoleProjectReadme,
		artifactbuiltin.WorkspaceContextRoleProjectContext:
		return true
	default:
		return false
	}
}

var artifactSupport = spec.ArtifactSupport{
	Kind:      artifactbuiltin.WorkspaceContextArtifactKind,
	SchemaID:  artifactbuiltin.WorkspaceContextSchemaID,
	DecoderID: artifactbuiltin.WorkspaceContextDecoderID,
	Validator: ValidateContextDefinition,
}

type contextDefinition struct {
	Name      string                                    `json:"name"`
	Role      artifactbuiltin.WorkspaceContextRole      `json:"role"`
	MediaType artifactbuiltin.WorkspaceContextMediaType `json:"mediaType"`
	Content   string                                    `json:"content"`
}
