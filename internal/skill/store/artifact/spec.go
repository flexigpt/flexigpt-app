package artifact

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
