package spec

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"

type MCPReadResourceRequestBody struct {
	URI string `json:"uri" required:"true"`
}

type MCPReadResourceRequest struct {
	Server artifact.ArtifactRef `json:"server" required:"true"`
	Body   *MCPReadResourceRequestBody
}

type MCPReadResourceResponseBody struct {
	Server   artifact.ArtifactRef `json:"server"`
	URI      string               `json:"uri"`
	Contents []MCPContent         `json:"contents,omitempty"`
}

type MCPReadResourceResponse struct {
	Body *MCPReadResourceResponseBody
}
