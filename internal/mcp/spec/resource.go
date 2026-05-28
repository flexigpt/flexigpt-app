package spec

type MCPReadResourceRequestBody struct {
	ServerID MCPServerID `json:"serverID" required:"true"`
	URI      string      `json:"uri"      required:"true"`
}

type MCPReadResourceRequest struct {
	Body *MCPReadResourceRequestBody
}

type MCPReadResourceResponseBody struct {
	ServerID MCPServerID  `json:"serverID"`
	URI      string       `json:"uri"`
	Contents []MCPContent `json:"contents,omitempty"`
}

type MCPReadResourceResponse struct {
	Body *MCPReadResourceResponseBody
}
