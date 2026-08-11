package spec

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"

type MCPUIResourceCSP struct {
	ConnectDomains  []string `json:"connectDomains,omitempty"`
	ResourceDomains []string `json:"resourceDomains,omitempty"`
	FrameDomains    []string `json:"frameDomains,omitempty"`
	BaseURIDomains  []string `json:"baseUriDomains,omitempty"`
}

type MCPUIResourcePermissions struct {
	Camera         map[string]any `json:"camera,omitempty"`
	Microphone     map[string]any `json:"microphone,omitempty"`
	Geolocation    map[string]any `json:"geolocation,omitempty"`
	ClipboardWrite map[string]any `json:"clipboardWrite,omitempty"`
}

type MCPUIResourceMeta struct {
	CSP           *MCPUIResourceCSP         `json:"csp,omitempty"`
	Permissions   *MCPUIResourcePermissions `json:"permissions,omitempty"`
	Domain        string                    `json:"domain,omitempty"`
	PrefersBorder *bool                     `json:"prefersBorder,omitempty"`
}

type MCPUIResourceContent struct {
	Server   artifact.ArtifactRef `json:"server"`
	URI      string               `json:"uri"`
	MIMEType string               `json:"mimeType"`
	HTML     string               `json:"html"`
	Meta     *MCPUIResourceMeta   `json:"meta,omitempty"`
	Digest   string               `json:"digest,omitempty"`
}

type MCPAppModelContextUpdate struct {
	InstanceID string               `json:"instanceID,omitempty"`
	Server     artifact.ArtifactRef `json:"server"`

	ResourceURI string `json:"resourceUri,omitempty"`

	Content           []MCPContent `json:"content,omitempty"`
	StructuredContent any          `json:"structuredContent,omitempty"`

	UpdatedAt string `json:"updatedAt,omitempty"`
}
