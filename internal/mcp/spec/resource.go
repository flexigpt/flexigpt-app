package spec

import "github.com/flexigpt/flexigpt-app/internal/bundleitemutils"

type MCPReadResourceRequestBody struct {
	BundleID bundleitemutils.BundleID `json:"bundleID" required:"true"`
	ServerID MCPServerID              `json:"serverID" required:"true"`
	URI      string                   `json:"uri"      required:"true"`
}

type MCPReadResourceRequest struct {
	Body *MCPReadResourceRequestBody
}

type MCPReadResourceResponseBody struct {
	BundleID bundleitemutils.BundleID `json:"bundleID"`
	ServerID MCPServerID              `json:"serverID"`
	URI      string                   `json:"uri"`
	Contents []MCPContent             `json:"contents,omitempty"`
}

type MCPReadResourceResponse struct {
	Body *MCPReadResourceResponseBody
}
