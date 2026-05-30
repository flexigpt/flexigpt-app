package spec

import "github.com/flexigpt/flexigpt-app/internal/bundleitemutils"

type MCPReadResourceRequestBody struct {
	URI string `json:"uri" required:"true"`
}

type MCPReadResourceRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	ServerID MCPServerID              `path:"serverID" required:"true"`

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
