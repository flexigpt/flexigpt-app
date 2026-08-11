package spec

import (
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
)

// MCPRuntimeStdioConfig is the materialized process-local stdio transport
// configuration. Secret values have already been resolved into Env only at
// connection preparation time and are never persisted in an MCP document.
type MCPRuntimeStdioConfig struct {
	Command          string            `json:"command"`
	Args             []string          `json:"args,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	StartupTimeoutMS int               `json:"startupTimeoutMS,omitempty"`
}

// MCPRuntimeStreamableHTTPConfig is the materialized process-local HTTP
// transport configuration. ClientCredentialRef remains an opaque
// Artifact-scoped Setting Store reference. Secret HTTP values are materialized
// only immediately before opening a runtime connection.
type MCPRuntimeStreamableHTTPConfig struct {
	URL       string          `json:"url"`
	TimeoutMS int             `json:"timeoutMS,omitempty"`
	AuthMode  MCPHTTPAuthMode `json:"authMode"`

	Headers map[string]string `json:"headers,omitempty"`

	ClientCredentialRef         string `json:"clientCredentialRef,omitempty"`
	ClientIDMetadataDocumentURL string `json:"clientIDMetadataDocumentURL,omitempty"`
}

type MCPSettings struct {
	// Empty means a random loopback port is used for the current process.
	OAuthLoopbackListenAddr string `json:"oauthLoopbackListenAddr,omitempty"`
}

type MCPAuthStatus struct {
	Server   artifact.ArtifactRef `json:"server"`
	AuthMode MCPHTTPAuthMode      `json:"authMode"`
	State    MCPAuthState         `json:"state"`

	Scopes              []string   `json:"scopes,omitempty"`
	ExpiresAt           *time.Time `json:"expiresAt,omitempty"`
	LastError           string     `json:"lastError,omitempty"`
	AuthorizationServer string     `json:"authorizationServer,omitempty"`
	Resource            string     `json:"resource,omitempty"`
}

type MCPAuthHealth struct {
	Server   artifact.ArtifactRef `json:"server"`
	AuthMode MCPHTTPAuthMode      `json:"authMode"`
	State    MCPAuthHealthState   `json:"state"`

	Configured bool `json:"configured"`

	Resource  string     `json:"resource,omitempty"`
	Scopes    []string   `json:"scopes,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	AuthorizationPending   bool   `json:"authorizationPending,omitempty"`
	AuthorizationURL       string `json:"authorizationURL,omitempty"`
	AuthorizationExpiresAt string `json:"authorizationExpiresAt,omitempty"`

	OAuthRedirectURL        string `json:"oauthRedirectURL,omitempty"`
	OAuthLoopbackListenAddr string `json:"oauthLoopbackListenAddr,omitempty"`

	LastError string `json:"lastError,omitempty"`
}
