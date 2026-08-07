package artifact

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"

const (
	Kind      basespec.ArtifactKind = "agent.skill"
	SchemaID  basespec.SchemaID     = "agent.skill.v1"
	DecoderID basespec.DecoderID    = "agent.skill-markdown"

	SchemaVersion      = "v1"
	DefinitionFileName = "SKILL.md"
	InsertLabelKey     = "skill.insert"
)

type Argument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type Body struct {
	Name           string         `json:"name"`
	DisplayName    string         `json:"displayName,omitempty"`
	Description    string         `json:"description"`
	Insert         string         `json:"insert"`
	Arguments      []Argument     `json:"arguments,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	MarkdownBody   string         `json:"markdownBody"`
	RawFrontmatter map[string]any `json:"rawFrontmatter,omitempty"`
}

type LocatorPolicy interface {
	ExpectedName(
		locator basespec.Locator,
	) (string, bool)
}
