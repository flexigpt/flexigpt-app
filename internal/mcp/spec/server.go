package spec

import "time"

type MCPServerConfig struct {
	SchemaVersion string           `json:"schemaVersion"`
	ID            MCPServerID      `json:"id"`
	DisplayName   string           `json:"displayName"`
	Enabled       bool             `json:"enabled"`
	Transport     MCPTransportType `json:"transport"`

	Stdio          *MCPStdioConfig          `json:"stdio,omitempty"`
	StreamableHTTP *MCPStreamableHTTPConfig `json:"streamableHttp,omitempty"`

	Availability MCPServerAvailability `json:"availability"`
	TrustLevel   MCPTrustLevel         `json:"trustLevel"`

	DefaultPolicy MCPServerPolicy                  `json:"defaultPolicy"`
	ToolPolicies  map[string]MCPToolPolicyOverride `json:"toolPolicies,omitempty"`
	AppsPolicy    *MCPAppsPolicy                   `json:"appsPolicy,omitempty"`
	AuthRef       *MCPAuthRef                      `json:"authRef,omitempty"`

	CreatedAt     time.Time  `json:"createdAt"`
	ModifiedAt    time.Time  `json:"modifiedAt"`
	SoftDeletedAt *time.Time `json:"softDeletedAt,omitempty"`
}

type MCPStdioConfig struct {
	Command          string            `json:"command"`
	Args             []string          `json:"args,omitempty"`
	WorkingDir       string            `json:"workingDir,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	SecretEnvRefs    map[string]string `json:"secretEnvRefs,omitempty"`
	StartupTimeoutMS int               `json:"startupTimeoutMS,omitempty"`
}

type MCPStreamableHTTPConfig struct {
	URL              string            `json:"url"`
	TimeoutMS        int               `json:"timeoutMS,omitempty"`
	CustomHeaders    map[string]string `json:"customHeaders,omitempty"`
	SecretHeaderRefs map[string]string `json:"secretHeaderRefs,omitempty"`
	AuthMode         MCPHTTPAuthMode   `json:"authMode"`
}

type MCPAuthRef struct {
	AuthMode            MCPHTTPAuthMode `json:"authMode"`
	TokenRef            string          `json:"tokenRef,omitempty"`
	ClientCredentialRef string          `json:"clientCredentialRef,omitempty"`
	MetadataRef         string          `json:"metadataRef,omitempty"`
}

type MCPAuthStatus struct {
	ServerID MCPServerID     `json:"serverID"`
	AuthMode MCPHTTPAuthMode `json:"authMode"`
	State    MCPAuthState    `json:"state"`

	Scopes              []string   `json:"scopes,omitempty"`
	ExpiresAt           *time.Time `json:"expiresAt,omitempty"`
	LastError           string     `json:"lastError,omitempty"`
	AuthorizationServer string     `json:"authorizationServer,omitempty"`
	Resource            string     `json:"resource,omitempty"`
}
