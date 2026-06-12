package spec

import (
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
)

type MCPStdioConfig struct {
	Command          string            `json:"command"`
	Args             []string          `json:"args,omitempty"`
	WorkingDir       string            `json:"workingDir,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	SecretEnvRefs    map[string]string `json:"secretEnvRefs,omitempty"`
	StartupTimeoutMS int               `json:"startupTimeoutMS,omitempty"`
}

type MCPStreamableHTTPConfig struct {
	URL       string          `json:"url"`
	TimeoutMS int             `json:"timeoutMS,omitempty"`
	AuthMode  MCPHTTPAuthMode `json:"authMode"`

	// ClientCredentialRef references an MCP OAuth client credential secret.
	// For authMode "oauth", the secret may contain only clientID for a public
	// PKCE client, or clientID plus clientSecret for a confidential client.
	// For authMode "clientCredentials", clientSecret is required.
	ClientCredentialRef string `json:"clientCredentialRef,omitempty"`

	// ClientIDMetadataDocumentURL enables the standard OAuth Client ID Metadata
	// Document registration path supported by the official MCP Go SDK.
	ClientIDMetadataDocumentURL string `json:"clientIDMetadataDocumentURL,omitempty"`
}

type MCPAuthStatus struct {
	BundleID bundleitemutils.BundleID `json:"bundleID"`
	ServerID MCPServerID              `json:"serverID"`
	AuthMode MCPHTTPAuthMode          `json:"authMode"`
	State    MCPAuthState             `json:"state"`

	Scopes              []string   `json:"scopes,omitempty"`
	ExpiresAt           *time.Time `json:"expiresAt,omitempty"`
	LastError           string     `json:"lastError,omitempty"`
	AuthorizationServer string     `json:"authorizationServer,omitempty"`
	Resource            string     `json:"resource,omitempty"`
}

type MCPServerConfig struct {
	SchemaVersion string `json:"schemaVersion"`

	BundleID    bundleitemutils.BundleID `json:"bundleID"`
	ID          MCPServerID              `json:"id"`
	DisplayName string                   `json:"displayName"`
	Enabled     bool                     `json:"enabled"`
	Transport   MCPTransportType         `json:"transport"`

	TrustLevel MCPTrustLevel `json:"trustLevel"`

	Stdio          *MCPStdioConfig          `json:"stdio,omitempty"`
	StreamableHTTP *MCPStreamableHTTPConfig `json:"streamableHttp,omitempty"`

	DefaultPolicy MCPServerPolicy                  `json:"defaultPolicy"`
	ToolPolicies  map[string]MCPToolPolicyOverride `json:"toolPolicies,omitempty"`
	AppsPolicy    *MCPAppsPolicy                   `json:"appsPolicy,omitempty"`

	IsBuiltIn     bool       `json:"isBuiltIn"`
	CreatedAt     time.Time  `json:"createdAt"`
	ModifiedAt    time.Time  `json:"modifiedAt"`
	SoftDeletedAt *time.Time `json:"softDeletedAt,omitempty"`
}

type MCPAuthHealth struct {
	BundleID bundleitemutils.BundleID `json:"bundleID,omitempty"`

	ServerID MCPServerID        `json:"serverID"`
	AuthMode MCPHTTPAuthMode    `json:"authMode"`
	State    MCPAuthHealthState `json:"state"`

	Configured bool `json:"configured"`

	Resource  string     `json:"resource,omitempty"`
	Scopes    []string   `json:"scopes,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	AuthorizationPending   bool   `json:"authorizationPending,omitempty"`
	AuthorizationURL       string `json:"authorizationURL,omitempty"`
	AuthorizationExpiresAt string `json:"authorizationExpiresAt,omitempty"`

	LastError string `json:"lastError,omitempty"`
}

type MCPBundle struct {
	SchemaVersion string                     `json:"schemaVersion"`
	ID            bundleitemutils.BundleID   `json:"id"`
	Slug          bundleitemutils.BundleSlug `json:"slug"`

	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`

	IsEnabled     bool       `json:"isEnabled"`
	CreatedAt     time.Time  `json:"createdAt"`
	ModifiedAt    time.Time  `json:"modifiedAt"`
	IsBuiltIn     bool       `json:"isBuiltIn"`
	SoftDeletedAt *time.Time `json:"softDeletedAt,omitempty"`
}

type AllMCPBundles struct {
	Bundles map[bundleitemutils.BundleID]MCPBundle `json:"bundles"`
}
