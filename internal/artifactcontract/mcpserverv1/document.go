package mcpserverv1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	MCPServerKind          = "mcp.server"
	MCPServerSchemaID      = "mcp.server.v1"
	MCPServerSchemaVersion = "v1"
)

//go:embed mcp-server-v1.schema.json
var schemaJSON []byte

var compiledMCPServerSchema = jsonutil.MustCompileJSONSchema(
	schemaJSON,
)

var MCPServerSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(MCPServerKind),
	schema.SchemaID(MCPServerSchemaID),
	MCPServerSchemaVersion,
)

func MCPServerJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

type MCPServerDocument struct {
	Digest         *string             `json:"digest,omitempty"`
	Kind           string              `json:"kind"`
	SchemaID       string              `json:"schemaID"`
	SchemaVersion  string              `json:"schemaVersion"`
	LogicalName    string              `json:"logicalName"`
	LogicalVersion string              `json:"logicalVersion,omitempty"`
	DisplayName    string              `json:"displayName,omitempty"`
	Description    string              `json:"description,omitempty"`
	Labels         map[string]string   `json:"labels,omitempty"`
	MCPServer      MCPServerCoreServer `json:"mcpServer"`
	Extension      MCPServerExtension  `json:"extension"`
}

type MCPServerCoreServer struct {
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type MCPServerAuthentication struct {
	Mode                        string `json:"mode,omitempty"`
	ClientCredentialsInput      string `json:"clientCredentialsInput,omitempty"`
	ClientIDMetadataDocumentURL string `json:"clientIDMetadataDocumentURL,omitempty"`
}

type MCPServerInputDeclaration struct {
	Kind                 string `json:"kind"`
	Label                string `json:"label,omitempty"`
	Description          string `json:"description,omitempty"`
	Note                 string `json:"note,omitempty"`
	Placeholder          string `json:"placeholder,omitempty"`
	Required             *bool  `json:"required,omitempty"`
	Default              string `json:"default,omitempty"`
	ClientSecretRequired *bool  `json:"clientSecretRequired,omitempty"`
}

type MCPServerInstallation struct {
	Note             string                               `json:"note,omitempty"`
	Inputs           map[string]MCPServerInputDeclaration `json:"inputs,omitempty"`
	AllowEnvironment []string                             `json:"allowEnvironment,omitempty"`
}

type MCPServerStdioProfile struct {
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	RemoveEnv []string          `json:"removeEnv,omitempty"`
}

type MCPServerHTTPProfile struct {
	URL           string            `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	RemoveHeaders []string          `json:"removeHeaders,omitempty"`
}

type MCPServerConnectionProfile struct {
	Platforms []string               `json:"platforms,omitempty"`
	Stdio     *MCPServerStdioProfile `json:"stdio,omitempty"`
	HTTP      *MCPServerHTTPProfile  `json:"http,omitempty"`
}

type MCPServerPolicyReference struct {
	Ref      string `json:"ref"`
	Required *bool  `json:"required,omitempty"`
}

type MCPServerExtension struct {
	LogicalVersion     string                                `json:"logicalVersion,omitempty"`
	DisplayName        string                                `json:"displayName,omitempty"`
	Description        string                                `json:"description,omitempty"`
	TimeoutMS          *int                                  `json:"timeoutMS,omitempty"`
	Labels             map[string]string                     `json:"labels,omitempty"`
	Auth               *MCPServerAuthentication              `json:"auth,omitempty"`
	Install            *MCPServerInstallation                `json:"install,omitempty"`
	ConnectionProfiles map[string]MCPServerConnectionProfile `json:"connectionProfiles,omitempty"`
	Policy             *MCPServerPolicyReference             `json:"policy,omitempty"`
}

func DecodeMCPServerJSON(raw []byte) (MCPServerDocument, error) {
	return jsonutil.DecodeCanonicalObject[MCPServerDocument](
		raw,
		basespec.MaxDefinitionBytes,
	)
}

func (v MCPServerDocument) Clone() (MCPServerDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v MCPServerDocument) Canonicalize() (MCPServerDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v MCPServerDocument) CanonicalJSON() ([]byte, error) {
	return jsonutil.MarshalCanonicalObject(v, basespec.MaxDefinitionBytes)
}

func (v MCPServerDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	payload := v
	payload.Digest = nil
	return cryptoutil.CanonicalDigest(payload)
}

func (v MCPServerDocument) Validate() error {
	if err := jsonutil.ValidateJSONSchema(
		compiledMCPServerSchema,
		v,
		basespec.MaxDefinitionBytes,
	); err != nil {
		return fmt.Errorf("MCP Server schema: %w", err)
	}
	return basespec.ValidatePortableMetadata(
		basespec.LogicalName(v.LogicalName),
		basespec.LogicalVersion(v.LogicalVersion),
		v.DisplayName,
		v.Description,
		v.Labels,
		v.Digest,
	)
}
