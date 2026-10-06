package server

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/policy"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	mcpv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/contract/v1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/secret"
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
	Mode mcpv1.HTTPAuthMode `json:"mode"`

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
	Name     spec.LogicalName `json:"name"`
	Required bool             `json:"required"`
}

// ServerConfiguration is the runtime projection of direct MCP declaration
// fields. It is not encoded in metadata and is not Artifact-local state.
type ServerConfiguration struct {
	TimeoutMS          int                          `json:"timeoutMS,omitempty"`
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

// ServerDocument is the MCP consumer projection of one direct canonical mcpv1
// declaration. It is not itself a portable document format.
type ServerDocument struct {
	LogicalName    spec.LogicalName    `json:"logicalName"`
	LogicalVersion spec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string              `json:"displayName,omitempty"`
	Description    string              `json:"description,omitempty"`
	Labels         map[string]string   `json:"labels,omitempty"`

	MCPServer     CoreServer          `json:"mcpServer"`
	Include       *Include            `json:"include,omitempty"`
	Configuration ServerConfiguration `json:"configuration"`
}

func (d ServerDocument) OAuthClientSecretRequired() bool {
	if d.Configuration.Auth.Mode == mcpv1.HTTPAuthModeClientCredentials {
		return true
	}
	input := d.Configuration.Auth.ClientCredentialsInput
	return input != "" &&
		d.Configuration.Install.Inputs[input].ClientSecretRequired
}

func (d ServerDocument) Clone() ServerDocument {
	output := d
	output.Labels = maps.Clone(d.Labels)
	output.MCPServer = cloneCore(d.MCPServer)
	output.Configuration = cloneConfiguration(d.Configuration)
	if d.Include != nil {
		value := *d.Include
		value.Tools = slices.Clone(d.Include.Tools)
		value.Resources = slices.Clone(d.Include.Resources)
		value.Prompts = slices.Clone(d.Include.Prompts)
		output.Include = &value
	}
	return output
}

func (d ServerDocument) Validate() error {
	if err := d.LogicalName.Validate(); err != nil {
		return err
	}
	if err := d.LogicalVersion.Validate(true); err != nil {
		return err
	}
	if err := spec.ValidateOptionalText(
		"MCP server display name",
		d.DisplayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateOptionalText(
		"MCP server description",
		d.Description,
		spec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateLabels(
		"MCP server",
		d.Labels,
	); err != nil {
		return err
	}
	if err := validateInclude(d.Include); err != nil {
		return err
	}
	return validateParts(
		string(d.LogicalName),
		d.MCPServer,
		d.Configuration,
	)
}

func (d ServerDocument) SecretInputTargets() (
	map[string]SecretInputTarget,
	error,
) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return secretInputTargets(d.MCPServer, d.Configuration)
}

func (d ServerDocument) AcceptsSecretTarget(
	kind secret.MCPSecretKind,
	slot string,
) error {
	if err := d.Validate(); err != nil {
		return err
	}
	normalizedSlot, err := kind.NormalizeSlot(slot)
	if err != nil {
		return err
	}

	switch kind {
	case secret.MCPSecretKindOAuthClientCredentials:
		input := d.Configuration.Auth.ClientCredentialsInput
		declaration, found := d.Configuration.Install.Inputs[input]
		if input == "" ||
			!found ||
			declaration.Kind != InputOAuthClientCredentials {
			return fmt.Errorf(
				"%w: OAuth client credential target is not declared",
				spec.ErrReferenceUnresolved,
			)
		}
		return nil

	case secret.MCPSecretKindStdioEnv, secret.MCPSecretKindHTTPHeader:
		targets, err := d.SecretInputTargets()
		if err != nil {
			return err
		}
		for _, target := range targets {
			expectedKind := secret.MCPSecretKindStdioEnv
			if target.Kind == SecretInputTargetHTTPHeader {
				expectedKind = secret.MCPSecretKindHTTPHeader
			}
			if expectedKind == kind &&
				strings.EqualFold(target.Slot, normalizedSlot) {
				return nil
			}
		}
		return fmt.Errorf(
			"%w: MCP secret target is not declared by the server",
			spec.ErrReferenceUnresolved,
		)

	default:
		return fmt.Errorf(
			"%w: unsupported MCP secret kind %q",
			spec.ErrInvalid,
			kind,
		)
	}
}

type MaterializedServer struct {
	Core                           CoreServer                `json:"-"`
	Auth                           AuthenticationDeclaration `json:"-"`
	ClientCredentialRef            string                    `json:"-"`
	ClientCredentialSecretRequired bool                      `json:"-"`
	TimeoutMS                      int                       `json:"-"`
	SensitiveValues                []string                  `json:"-"`
}

// Resolved is backend runtime material. Consumer APIs project it into
// domain-specific views and must not serialize connection or installation
// internals directly.
type Resolved struct {
	Server               artifactModel.ArtifactRef `json:"-"`
	ArtifactRevision     uint64                    `json:"-"`
	DefinitionDigest     cryptoutil.Digest         `json:"-"`
	SourceContentDigest  cryptoutil.Digest         `json:"-"`
	SourceGeneration     string                    `json:"-"`
	Document             ServerDocument            `json:"-"`
	Installation         ServerData                `json:"-"`
	Policy               mcpPolicy.Effective       `json:"-"`
	InstallationRevision uint64                    `json:"-"`
	BuiltIn              bool                      `json:"-"`
	Version              cryptoutil.Digest         `json:"-"`
}
