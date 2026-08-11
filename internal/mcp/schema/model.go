package schema

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	BundleKind basespec.CollectionKind = "mcp.bundle"
	ServerKind basespec.ArtifactKind   = "mcp.server"
	PolicyKind basespec.ArtifactKind   = "mcp.policy"

	BundleSchemaID basespec.SchemaID = "mcp.bundle.v1"
	ServerSchemaID basespec.SchemaID = "mcp.server.v1"
	PolicySchemaID basespec.SchemaID = "mcp.policy.v1"

	SchemaVersion = "v1"

	BundleSchemaURL = "https://schemas.flexigpt.dev/mcp/bundle/v1.json"
	ServerSchemaURL = "https://schemas.flexigpt.dev/mcp/server/v1.json"
	PolicySchemaURL = "https://schemas.flexigpt.dev/mcp/policy/v1.json"

	BundleFileName = ".mcp.json"
)

type ServerType string

const (
	ServerTypeStdio ServerType = "stdio"
	ServerTypeHTTP  ServerType = "http"
)

type InputKind string

const (
	InputText   InputKind = "text"
	InputSecret InputKind = "secret"
	InputPath   InputKind = "path"
	//nolint:gosec // Cred enum.
	InputOAuthClientCredentials InputKind = "oauthClientCredentials"
)

type CoreServer struct {
	Type ServerType `json:"type,omitempty"`

	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`

	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

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
	Mode mcpSpec.MCPHTTPAuthMode `json:"mode"`

	ClientCredentialsInput      string `json:"clientCredentialsInput,omitempty"`
	ClientIDMetadataDocumentURL string `json:"clientIDMetadataDocumentURL,omitempty"`
}

type PolicyReference struct {
	Ref      basespec.LogicalName `json:"ref"`
	Required bool                 `json:"required"`
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

type ServerExtension struct {
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`

	Auth               AuthenticationDeclaration    `json:"auth"`
	Install            InstallationDeclaration      `json:"install"`
	ConnectionProfiles map[string]ConnectionProfile `json:"connectionProfiles,omitempty"`
	Policy             *PolicyReference             `json:"policy,omitempty"`
}

type PolicyBody struct {
	TrustLevel    mcpSpec.MCPTrustLevel                    `json:"trustLevel"`
	DefaultPolicy mcpSpec.MCPServerPolicy                  `json:"defaultPolicy"`
	ToolPolicies  map[string]mcpSpec.MCPToolPolicyOverride `json:"toolPolicies,omitempty"`
	AppsPolicy    mcpSpec.MCPAppsPolicy                    `json:"appsPolicy"`
}

type PolicyDocument struct {
	SchemaURL     string                `json:"$schema"`
	Kind          basespec.ArtifactKind `json:"kind"`
	SchemaID      basespec.SchemaID     `json:"schemaID"`
	SchemaVersion string                `json:"schemaVersion"`
	Digest        cryptoutil.Digest     `json:"digest,omitempty"`

	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`

	Body PolicyBody `json:"body"`
}

type BundleExtension struct {
	Servers  map[string]ServerExtension `json:"servers,omitempty"`
	Policies map[string]PolicyDocument  `json:"policies,omitempty"`
}

type BundleDocument struct {
	SchemaURL     string                  `json:"$schema"`
	Kind          basespec.CollectionKind `json:"kind"`
	SchemaID      basespec.SchemaID       `json:"schemaID"`
	SchemaVersion string                  `json:"schemaVersion"`
	Digest        cryptoutil.Digest       `json:"digest,omitempty"`

	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`

	MCPServers      map[string]CoreServer `json:"mcpServers"`
	BundleExtension BundleExtension       `json:"bundleExtension"`
}

type ServerDocument struct {
	SchemaURL     string                `json:"$schema"`
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
	if d.Extension.Auth.Mode == mcpSpec.MCPHTTPAuthClientCredentials {
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
