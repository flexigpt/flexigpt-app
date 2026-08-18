package server

import (
	"regexp"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
)

var placeholderPattern = regexp.MustCompile(
	`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`,
)

const (
	DefaultConnectionTimeoutMS = 30_000
	MaxConnectionTimeoutMS     = 10 * 60 * 1_000
)

type ServerType string

const (
	ServerTypeStdio ServerType = "stdio"
	ServerTypeHTTP  ServerType = "http"
)

type CoreServer struct {
	Type ServerType `json:"type,omitempty"`

	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`

	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type MCPHTTPAuthMode string

const (
	MCPHTTPAuthNone              MCPHTTPAuthMode = "none"
	MCPHTTPAuthAPIKey            MCPHTTPAuthMode = "apiKey"
	MCPHTTPAuthOAuth             MCPHTTPAuthMode = "oauth"
	MCPHTTPAuthClientCredentials MCPHTTPAuthMode = "clientCredentials"
)

type MCPTransportType string

const (
	MCPTransportStreamableHTTP MCPTransportType = "streamableHttp"
	MCPTransportStdio          MCPTransportType = "stdio"
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

type RuntimeConfig struct {
	Server     artifact.ArtifactRef
	Collection collection.CollectionRef

	LogicalName string
	DisplayName string

	Transport                 MCPTransportType
	Stdio                     *MCPRuntimeStdioConfig
	StreamableHTTP            *MCPRuntimeStreamableHTTPConfig
	OAuthClientSecretRequired bool

	TrustLevel    policy.MCPTrustLevel
	DefaultPolicy policy.MCPServerPolicy
	ToolPolicies  map[string]policy.MCPToolPolicyOverride
	AppsPolicy    policy.MCPAppsPolicy

	SensitiveValues []string
}

type InputKind string

const (
	InputText   InputKind = "text"
	InputSecret InputKind = "secret"
	InputPath   InputKind = "path"
	//nolint:gosec // Cred enum.
	InputOAuthClientCredentials InputKind = "oauthClientCredentials"
)

type InputDeclaration struct {
	Kind                 InputKind `json:"kind"`
	Label                string    `json:"label,omitempty"`
	Description          string    `json:"description,omitempty"`
	Note                 string    `json:"note,omitempty"`
	Placeholder          string    `json:"placeholder,omitempty"`
	Required             bool      `json:"required,omitempty"`
	Default              *string   `json:"default,omitempty"`
	ClientSecretRequired bool      `json:"clientSecretRequired,omitempty"`
}

type InstallationDeclaration struct {
	Note             string                      `json:"note,omitempty"`
	Inputs           map[string]InputDeclaration `json:"inputs,omitempty"`
	AllowEnvironment []string                    `json:"allowEnvironment,omitempty"`
}

type AuthenticationDeclaration struct {
	Mode MCPHTTPAuthMode `json:"mode"`

	ClientCredentialsInput      string `json:"clientCredentialsInput,omitempty"`
	ClientIDMetadataDocumentURL string `json:"clientIDMetadataDocumentURL,omitempty"`
}

type StdioProfile struct {
	Command   *string           `json:"command,omitempty"`
	Args      *[]string         `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	RemoveEnv []string          `json:"removeEnv,omitempty"`
}

type HTTPProfile struct {
	URL           *string           `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	RemoveHeaders []string          `json:"removeHeaders,omitempty"`
}

type ConnectionProfile struct {
	Platforms []string      `json:"platforms,omitempty"`
	Stdio     *StdioProfile `json:"stdio,omitempty"`
	HTTP      *HTTPProfile  `json:"http,omitempty"`
}

type PolicyReference struct {
	Ref      basespec.LogicalName `json:"ref"`
	Required bool                 `json:"required"`
}

type ServerExtension struct {
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	TimeoutMS      int                     `json:"timeoutMS,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`

	Auth               AuthenticationDeclaration    `json:"auth"`
	Install            InstallationDeclaration      `json:"install"`
	ConnectionProfiles map[string]ConnectionProfile `json:"connectionProfiles,omitempty"`
	Policy             *PolicyReference             `json:"policy,omitempty"`
}

type ServerDocument struct {
	Kind          basespec.ArtifactKind `json:"kind"`
	SchemaID      basespec.SchemaID     `json:"schemaID"`
	SchemaVersion string                `json:"schemaVersion"`
	Digest        cryptoutil.Digest     `json:"digest,omitempty"`

	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`

	MCPServer CoreServer      `json:"mcpServer"`
	Extension ServerExtension `json:"extension"`
}

// OAuthClientSecretRequired reports whether the declared OAuth client input
// must contain a confidential-client secret. Client-credentials flow always
// requires a secret even if a document omitted the explicit declaration flag.
func (d ServerDocument) OAuthClientSecretRequired() bool {
	if d.Extension.Auth.Mode == MCPHTTPAuthClientCredentials {
		return true
	}
	input := d.Extension.Auth.ClientCredentialsInput
	return input != "" &&
		d.Extension.Install.Inputs[input].ClientSecretRequired
}

type ServerDefinitionBody struct {
	MCPServer CoreServer      `json:"mcpServer"`
	Extension ServerExtension `json:"extension"`
}

type Resolved struct {
	Server               artifact.ArtifactRef     `json:"server"`
	Collection           collection.CollectionRef `json:"collection"`
	ArtifactRevision     uint64                   `json:"artifactRevision"`
	CatalogRevision      uint64                   `json:"catalogRevision"`
	DefinitionDigest     cryptoutil.Digest        `json:"definitionDigest"`
	SourceContentDigest  cryptoutil.Digest        `json:"sourceContentDigest"`
	SourceGeneration     string                   `json:"sourceGeneration"`
	Document             ServerDocument           `json:"document"`
	Installation         ServerData               `json:"installation"`
	Policy               policy.Effective         `json:"policy"`
	InstallationRevision uint64                   `json:"installationRevision"`
	RuntimeEnabled       bool                     `json:"runtimeEnabled"`
	BuiltIn              bool                     `json:"builtIn"`
	Version              cryptoutil.Digest        `json:"version"`
}
