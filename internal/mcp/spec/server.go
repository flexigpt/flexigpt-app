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

	// Headers are non-secret HTTP headers sent to the MCP endpoint.
	Headers map[string]string `json:"headers,omitempty"`

	// SecretHeaderRefs maps HTTP header names to MCP secret refs. Use this for
	// API keys, PATs, or Authorization Bearer tokens.
	SecretHeaderRefs map[string]string `json:"secretHeaderRefs,omitempty"`

	// ClientCredentialRef references an MCP OAuth client credential secret.
	// For authMode "oauth", the secret may contain only clientID for a public
	// PKCE client, or clientID plus clientSecret for a confidential client.
	// For authMode "clientCredentials", clientSecret is required.
	ClientCredentialRef string `json:"clientCredentialRef,omitempty"`

	// ClientIDMetadataDocumentURL enables the standard OAuth Client ID Metadata
	// Document registration path supported by the official MCP Go SDK.
	ClientIDMetadataDocumentURL string `json:"clientIDMetadataDocumentURL,omitempty"`
}

type MCPServerSetupInputKind string

const (
	//nolint:gosec // Enum val.
	MCPSetupKindOAuthClientCredentials MCPServerSetupInputKind = "oauthClientCredentials"
	MCPSetupKindHTTPHeader             MCPServerSetupInputKind = "httpHeader"
	MCPSetupKindStdioEnv               MCPServerSetupInputKind = "stdioEnv"
	MCPSetupKindStreamableHTTPURL      MCPServerSetupInputKind = "streamableHttpUrl"
	MCPSetupKindClientIDMetadataDocURL MCPServerSetupInputKind = "clientIDMetadataDocumentURL"
)

type MCPSetupOAuthClientCredentialsInput struct {
	ClientSecretRequired bool `json:"clientSecretRequired,omitempty"`
}

type MCPSetupHTTPHeaderInput struct {
	HeaderName string `json:"headerName"`
	// Secret stores the value in the secret store and references it via
	// secretHeaderRefs. Non-secret headers are stored inline.
	Secret      bool   `json:"secret,omitempty"`
	ValuePrefix string `json:"valuePrefix,omitempty"`
	ValueSuffix string `json:"valueSuffix,omitempty"`
}

type MCPSetupStdioEnvInput struct {
	EnvName     string `json:"envName"`
	Secret      bool   `json:"secret,omitempty"`
	ValuePrefix string `json:"valuePrefix,omitempty"`
	ValueSuffix string `json:"valueSuffix,omitempty"`
}

type MCPSetupStreamableHTTPURLInput struct{}

type MCPSetupClientIDMetadataDocumentURLInput struct{}

// MCPServerSetupInput is a discriminated union. Kind is mandatory and exactly
// one kind-specific pointer must be set and match Kind.
type MCPServerSetupInput struct {
	ID          string                  `json:"id"`
	Kind        MCPServerSetupInputKind `json:"kind"`
	Label       string                  `json:"label,omitempty"`
	Description string                  `json:"description,omitempty"`
	Note        string                  `json:"note,omitempty"`
	Placeholder string                  `json:"placeholder,omitempty"`
	Required    bool                    `json:"required,omitempty"`

	OAuthClientCredentials      *MCPSetupOAuthClientCredentialsInput      `json:"oauthClientCredentials,omitempty"`
	HTTPHeader                  *MCPSetupHTTPHeaderInput                  `json:"httpHeader,omitempty"`
	StdioEnv                    *MCPSetupStdioEnvInput                    `json:"stdioEnv,omitempty"`
	StreamableHTTPURL           *MCPSetupStreamableHTTPURLInput           `json:"streamableHttpUrl,omitempty"`
	ClientIDMetadataDocumentURL *MCPSetupClientIDMetadataDocumentURLInput `json:"clientIDMetadataDocumentURL,omitempty"`
}

// MCPServerSetup declares what user input a server needs before it can connect.
type MCPServerSetup struct {
	Note   string                `json:"note,omitempty"`
	Inputs []MCPServerSetupInput `json:"inputs,omitempty"`
}

// MCPBuiltInServerOverlay is user-owned runtime config layered on top of a
// read-only built-in server. Secret values are referenced, not inlined.
type MCPBuiltInServerOverlay struct {
	Stdio          *MCPStdioConfigOverlay          `json:"stdio,omitempty"`
	StreamableHTTP *MCPStreamableHTTPConfigOverlay `json:"streamableHttp,omitempty"`
}

type MCPStdioConfigOverlay struct {
	Env           map[string]string `json:"env,omitempty"`
	SecretEnvRefs map[string]string `json:"secretEnvRefs,omitempty"`
}

type MCPStreamableHTTPConfigOverlay struct {
	URL       *string `json:"url,omitempty"`
	TimeoutMS *int    `json:"timeoutMS,omitempty"`

	Headers          map[string]string `json:"headers,omitempty"`
	SecretHeaderRefs map[string]string `json:"secretHeaderRefs,omitempty"`

	ClientCredentialRef         *string `json:"clientCredentialRef,omitempty"`
	ClientIDMetadataDocumentURL *string `json:"clientIDMetadataDocumentURL,omitempty"`
}

type MCPSettings struct {
	// Empty means a random loopback port is used for this process.
	// Example fixed value: "127.0.0.1:37645".
	OAuthLoopbackListenAddr string `json:"oauthLoopbackListenAddr,omitempty"`
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
	Setup         *MCPServerSetup                  `json:"setup,omitempty"`

	IsBuiltIn  bool      `json:"isBuiltIn"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
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

	OAuthRedirectURL        string `json:"oauthRedirectURL,omitempty"`
	OAuthLoopbackListenAddr string `json:"oauthLoopbackListenAddr,omitempty"`

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
