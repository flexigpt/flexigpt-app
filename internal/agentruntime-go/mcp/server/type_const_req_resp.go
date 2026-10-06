package server

type MCPGetPromptRequestBody struct {
	PromptName string            `json:"promptName"          required:"true"`
	Arguments  map[string]string `json:"arguments,omitempty"`
}

type MCPGetPromptRequest struct {
	Server ServerID `json:"server" required:"true"`
	Body   *MCPGetPromptRequestBody
}

type MCPGetPromptResponseBody struct {
	Server      ServerID           `json:"server"`
	PromptName  string             `json:"promptName"`
	Description string             `json:"description,omitempty"`
	Messages    []MCPPromptMessage `json:"messages,omitempty"`
}

type MCPGetPromptResponse struct {
	Body *MCPGetPromptResponseBody
}

type MCPCompleteArgumentRequestBody struct {
	RefType       string            `json:"refType"                 required:"true"` // resource | prompt
	Name          string            `json:"name"                    required:"true"`
	ArgumentName  string            `json:"argumentName"            required:"true"`
	ArgumentValue string            `json:"argumentValue,omitempty"`
	Context       map[string]string `json:"context,omitempty"`
}

type MCPCompleteArgumentRequest struct {
	Server ServerID `json:"server" required:"true"`
	Body   *MCPCompleteArgumentRequestBody
}

type MCPCompletionResult struct {
	Values  []string `json:"values,omitempty"`
	Total   int      `json:"total,omitempty"`
	HasMore bool     `json:"hasMore,omitempty"`
}

type MCPReadResourceRequestBody struct {
	URI string `json:"uri" required:"true"`
}

type MCPReadResourceRequest struct {
	Server ServerID `json:"server" required:"true"`
	Body   *MCPReadResourceRequestBody
}

type MCPReadResourceResponseBody struct {
	Server   ServerID     `json:"server"`
	URI      string       `json:"uri"`
	Contents []MCPContent `json:"contents,omitempty"`
}

type MCPReadResourceResponse struct {
	Body *MCPReadResourceResponseBody
}
