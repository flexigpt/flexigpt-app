package spec

type MCPGetPromptRequestBody struct {
	ServerID   MCPServerID       `json:"serverID"            required:"true"`
	PromptName string            `json:"promptName"          required:"true"`
	Arguments  map[string]string `json:"arguments,omitempty"`
}

type MCPGetPromptRequest struct {
	Body *MCPGetPromptRequestBody
}

type MCPGetPromptResponseBody struct {
	ServerID    MCPServerID        `json:"serverID"`
	PromptName  string             `json:"promptName"`
	Description string             `json:"description,omitempty"`
	Messages    []MCPPromptMessage `json:"messages,omitempty"`
}

type MCPGetPromptResponse struct {
	Body *MCPGetPromptResponseBody
}

type MCPCompleteArgumentRequestBody struct {
	ServerID      MCPServerID       `json:"serverID"                required:"true"`
	RefType       string            `json:"refType"                 required:"true"` // resource | prompt
	Name          string            `json:"name"                    required:"true"`
	ArgumentName  string            `json:"argumentName"            required:"true"`
	ArgumentValue string            `json:"argumentValue,omitempty"`
	Context       map[string]string `json:"context,omitempty"`
}

type MCPCompleteArgumentRequest struct {
	Body *MCPCompleteArgumentRequestBody
}

type MCPCompletionResult struct {
	Values  []string `json:"values,omitempty"`
	Total   int      `json:"total,omitempty"`
	HasMore bool     `json:"hasMore,omitempty"`
}
