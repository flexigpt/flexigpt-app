package skilladapter

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

const (
	skillKind      artifactstore.ArtifactKind = "workspace.skill"
	skillSchemaID  artifactstore.SchemaID     = "workspace.skill.v1"
	skillDecoderID artifactstore.DecoderID    = "workspace.skill-markdown"

	workspaceSkillsSchemaVersionV1 = "v1"

	workspaceSkillsDirectory      = ".skills"
	skillDefinitionFileName       = "SKILL.md"
	skillInsertLabelKey           = "skill.insert"
	workspaceSkillNamePatternText = `^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`
)

type skillArgumentDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type skillDefinition struct {
	Name           string                    `json:"name"`
	DisplayName    string                    `json:"displayName,omitempty"`
	Description    string                    `json:"description"`
	Insert         string                    `json:"insert"`
	Arguments      []skillArgumentDefinition `json:"arguments,omitempty"`
	Tags           []string                  `json:"tags,omitempty"`
	MarkdownBody   string                    `json:"markdownBody"`
	RawFrontmatter map[string]any            `json:"rawFrontmatter,omitempty"`
}

var artifactSupport = engine.ArtifactSupport{
	Kind:      skillKind,
	SchemaID:  skillSchemaID,
	DecoderID: skillDecoderID,
	Validator: ValidateSkillDefinition,
}
