package server

import (
	"regexp"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

var placeholderPattern = regexp.MustCompile(
	`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`,
)

const (
	DefaultConnectionTimeoutMS = 30_000
	MaxConnectionTimeoutMS     = 10 * 60 * 1_000

	TransportLabelKey = "mcp.transport"
	AuthModeLabelKey  = "mcp.auth-mode"
)

type ServerType string

const (
	ServerTypeStdio ServerType = "stdio"
	ServerTypeHTTP  ServerType = "http"
	ServerTypeSSE   ServerType = "sse"
)

type MCPHTTPAuthMode = mcpServer.MCPHTTPAuthMode

const (
	MCPHTTPAuthNone              = mcpServer.MCPHTTPAuthNone
	MCPHTTPAuthAPIKey            = mcpServer.MCPHTTPAuthAPIKey
	MCPHTTPAuthOAuth             = mcpServer.MCPHTTPAuthOAuth
	MCPHTTPAuthClientCredentials = mcpServer.MCPHTTPAuthClientCredentials
)

type CoreServer struct {
	Type ServerType `json:"type,omitempty"`

	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`

	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type InputKind string

const (
	InputText   InputKind = "text"
	InputSecret InputKind = "secret"
	InputPath   InputKind = "path"
	//nolint:gosec // Enum.
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

// ServerExtension is consumer-specific MCP runtime metadata. It is encoded
// under the namespaced canonical mcpv1 metadata extension key rather than
// becoming Artifact Store state.
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

type Include struct {
	Tools     []string `json:"tools,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Prompts   []string `json:"prompts,omitempty"`
}

// ServerDocument is the MCP consumer projection of one canonical mcpv1
// declaration. It is not itself a portable document format.
type ServerDocument struct {
	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`

	MCPServer CoreServer      `json:"mcpServer"`
	Include   *Include        `json:"include,omitempty"`
	Extension ServerExtension `json:"extension"`
}

func (d ServerDocument) OAuthClientSecretRequired() bool {
	if d.Extension.Auth.Mode == MCPHTTPAuthClientCredentials {
		return true
	}
	input := d.Extension.Auth.ClientCredentialsInput
	return input != "" &&
		d.Extension.Install.Inputs[input].ClientSecretRequired
}

type MaterializedServer struct {
	Core                           CoreServer
	Auth                           AuthenticationDeclaration
	ClientCredentialRef            string
	ClientCredentialSecretRequired bool
	TimeoutMS                      int
	SensitiveValues                []string
}

type Resolved struct {
	Server               artifact.ArtifactRef `json:"server"`
	ArtifactRevision     uint64               `json:"artifactRevision"`
	DefinitionDigest     cryptoutil.Digest    `json:"definitionDigest"`
	SourceContentDigest  cryptoutil.Digest    `json:"sourceContentDigest"`
	SourceGeneration     string               `json:"sourceGeneration"`
	Document             ServerDocument       `json:"document"`
	Installation         ServerData           `json:"installation"`
	Policy               mcpPolicy.Effective  `json:"policy"`
	InstallationRevision uint64               `json:"installationRevision"`
	BuiltIn              bool                 `json:"builtIn"`
	Version              cryptoutil.Digest    `json:"version"`
}
