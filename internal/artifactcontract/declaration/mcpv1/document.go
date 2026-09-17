package mcpv1

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	MCPType          = declaration.TypeMCP
	MCPSchemaID      = "artifact.mcp.v1"
	MCPSchemaVersion = declaration.SchemaVersionV1

	//nolint:gosec // Str.
	secretOAuthClientCredentials = "oauthClientCredentials"
)

type Transport string

const (
	TransportStdio          Transport = "stdio"
	TransportStreamableHTTP Transport = "streamableHTTP"
)

type HTTPAuthMode string

const (
	HTTPAuthModeNone              HTTPAuthMode = "none"
	HTTPAuthModeAPIKey            HTTPAuthMode = "apiKey"
	HTTPAuthModeOAuth             HTTPAuthMode = "oauth"
	HTTPAuthModeClientCredentials HTTPAuthMode = "clientCredentials"
)

//go:embed mcp-v1.schema.json
var schemaJSON []byte

var compiledMCPSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var MCPSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(MCPType),
	schema.SchemaID(MCPSchemaID),
	MCPSchemaVersion,
)

var mcpURLTemplatePattern = regexp.MustCompile(
	`\$\{[A-Za-z_][A-Za-z0-9_]*\}`,
)

type Include struct {
	Tools     []string `json:"tools,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Prompts   []string `json:"prompts,omitempty"`
}

type Auth struct {
	Mode                        HTTPAuthMode `json:"mode"`
	ClientCredentialsInput      string       `json:"clientCredentialsInput,omitempty"`
	ClientIDMetadataDocumentURL string       `json:"clientIDMetadataDocumentURL,omitempty"`
}

type Install struct {
	Note             string                  `json:"note,omitempty"`
	Inputs           map[string]InstallInput `json:"inputs,omitempty"`
	AllowEnvironment []string                `json:"allowEnvironment,omitempty"`
}

type InstallInput struct {
	Kind                 string           `json:"kind"`
	Label                string           `json:"label,omitempty"`
	Description          string           `json:"description,omitempty"`
	Note                 string           `json:"note,omitempty"`
	Placeholder          string           `json:"placeholder,omitempty"`
	Required             *bool            `json:"required,omitempty"`
	Default              *json.RawMessage `json:"default,omitempty"`
	ClientSecretRequired *bool            `json:"clientSecretRequired,omitempty"`
}

type ConnectionProfile struct {
	Platforms []string      `json:"platforms,omitempty"`
	Stdio     *StdioProfile `json:"stdio,omitempty"`
	HTTP      *HTTPProfile  `json:"http,omitempty"`
}

type StdioProfile struct {
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	RemoveEnv []string          `json:"removeEnv,omitempty"`
}

type HTTPProfile struct {
	URL           string            `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	RemoveHeaders []string          `json:"removeHeaders,omitempty"`
}

type PolicyReference struct {
	Name     basespec.LogicalName `json:"name"`
	Required *bool                `json:"required,omitempty"`
}

type MCPDocument struct {
	declaration.Header

	Server    string    `json:"server,omitempty"`
	Transport Transport `json:"transport,omitempty"`

	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`

	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`

	Include            *Include                     `json:"include,omitempty"`
	TimeoutMS          int                          `json:"timeoutMS,omitempty"`
	Auth               *Auth                        `json:"auth,omitempty"`
	Install            *Install                     `json:"install,omitempty"`
	ConnectionProfiles map[string]ConnectionProfile `json:"connectionProfiles,omitempty"`
	Policy             *PolicyReference             `json:"policy,omitempty"`
}

func MCPJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeMCPJSON(raw []byte) (MCPDocument, error) {
	return decodeMCP(raw)
}

func DecodeMCPEntry(
	entry declaration.Entry,
) (MCPDocument, error) {
	var value MCPDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledMCPSchema,
		&value,
	); err != nil {
		return MCPDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return MCPDocument{}, err
	}
	return value, nil
}

// DefinitionForDeclaration projects a named canonical MCP declaration without
// translating it through a consumer runtime representation. In particular it
// preserves locator, server, include, metadata, and transport data in Body.
func DefinitionForDeclaration(
	input MCPDocument,
) (definition.Definition, error) {
	if err := input.Validate(); err != nil {
		return definition.Definition{}, err
	}
	body, err := input.CanonicalJSON()
	if err != nil {
		return definition.Definition{}, err
	}

	name := input.DisplayName
	if name == "" {
		name = input.Name
	}

	return definition.Canonicalize(definition.Definition{
		Kind:          artifact.ArtifactKind(MCPType),
		SchemaID:      MCPSchemaKey.SchemaID,
		SchemaVersion: MCPSchemaKey.SchemaVersion,
		LogicalName:   basespec.LogicalName(input.Name),
		DisplayName:   name,
		Description:   input.Description,
		Labels:        declaration.CloneStringMap(input.Labels),
		Body:          json.RawMessage(body),
	})
}

func decodeMCP(
	raw []byte,
) (MCPDocument, error) {
	var value MCPDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledMCPSchema,
		&value,
	); err != nil {
		return MCPDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return MCPDocument{}, err
	}
	return value, nil
}

func (v MCPDocument) Clone() (MCPDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v MCPDocument) Canonicalize() (MCPDocument, error) {
	return v.Clone()
}

func (v MCPDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v MCPDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v MCPDocument) Validate() error {
	return v.validate()
}

func (v MCPDocument) ValidateEntry() error {
	return v.validate()
}

func (v MCPDocument) validate() error {
	if err := declaration.ValidateDocument(
		compiledMCPSchema,
		v,
	); err != nil {
		return fmt.Errorf("MCP schema: %w", err)
	}
	return v.validateFields()
}

func (v MCPDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: MCPType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if v.Server != "" {
		if err := basespec.LogicalName(v.Server).Validate(); err != nil {
			return fmt.Errorf("MCP server selector: %w", err)
		}
	}
	if err := validateMCPSource(v); err != nil {
		return err
	}
	if err := validateTransport(v); err != nil {
		return err
	}
	if v.Command != "" {
		if err := basespec.ValidateRequiredText(
			"MCP command",
			v.Command,
			basespec.MaxURIBytes,
		); err != nil {
			return err
		}
	}
	if v.URL != "" {
		if err := validateMCPURL(v.URL); err != nil {
			return err
		}
	}
	if err := declaration.ValidateStringMap(
		"MCP env",
		v.Env,
	); err != nil {
		return err
	}
	if err := declaration.ValidateStringMap(
		"MCP headers",
		v.Headers,
	); err != nil {
		return err
	}
	if v.TimeoutMS < 0 {
		return fmt.Errorf(
			"%w: MCP timeoutMS cannot be negative",
			basespec.ErrInvalid,
		)
	}
	if v.Auth != nil {
		if err := v.Auth.Validate(); err != nil {
			return err
		}
	}
	if v.Install != nil {
		if err := v.Install.Validate(); err != nil {
			return err
		}
	}
	for name, profile := range v.ConnectionProfiles {
		if err := basespec.LogicalName(name).Validate(); err != nil {
			return fmt.Errorf("MCP connection profile name: %w", err)
		}
		if err := profile.Validate(); err != nil {
			return fmt.Errorf("MCP connection profile %q: %w", name, err)
		}
	}
	if v.Policy != nil {
		if err := v.Policy.Name.Validate(); err != nil {
			return fmt.Errorf("MCP policy reference: %w", err)
		}
	}
	if v.Include == nil {
		return nil
	}
	if err := declaration.ValidateTextSlice(
		"MCP include tools",
		v.Include.Tools,
		basespec.MaxLogicalNameBytes,
	); err != nil {
		return err
	}
	if err := declaration.ValidateTextSlice(
		"MCP include resources",
		v.Include.Resources,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return declaration.ValidateTextSlice(
		"MCP include prompts",
		v.Include.Prompts,
		basespec.MaxLogicalNameBytes,
	)
}

// validateMCPURL accepts a normal absolute MCP URL and the declared runtime
// placeholder form used by MCP installation inputs. The MCP consumer validates
// that every placeholder is declared and resolves it before runtime use.
func validateMCPURL(
	value string,
) error {
	return declaration.ValidateAbsoluteURL(
		"MCP URL",
		mcpURLTemplatePattern.ReplaceAllString(
			value,
			"placeholder",
		),
	)
}

func validateMCPSource(v MCPDocument) error {
	if v.Locator != nil && v.hasSourceSelectedOverrides() {
		return fmt.Errorf(
			"%w: source-selected MCP cannot contain local MCP configuration",
			basespec.ErrInvalid,
		)
	}
	if v.Server != "" && v.Locator == nil {
		return fmt.Errorf(
			"%w: MCP server selector requires a locator",
			basespec.ErrInvalid,
		)
	}
	if v.Locator != nil && hasMCPConnectionFields(v) {
		return fmt.Errorf(
			"%w: source-selected MCP cannot overlay transport or connection fields",
			basespec.ErrInvalid,
		)
	}
	if v.Locator == nil && !hasInlineMCPConnection(v) {
		return fmt.Errorf(
			"%w: concrete MCP requires a locator or executable connection",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func (v MCPDocument) hasSourceSelectedOverrides() bool {
	return v.Include != nil ||
		v.TimeoutMS != 0 ||
		v.Auth != nil ||
		v.Install != nil ||
		v.ConnectionProfiles != nil ||
		v.Policy != nil ||
		hasMCPConnectionFields(v)
}

func hasMCPConnectionFields(v MCPDocument) bool {
	return v.Transport != "" ||
		v.Command != "" ||
		v.Args != nil ||
		v.Env != nil ||
		v.URL != "" ||
		v.Headers != nil
}

func hasInlineMCPConnection(v MCPDocument) bool {
	switch v.Transport {
	case TransportStdio:
		return v.Command != ""
	case TransportStreamableHTTP:
		return v.URL != ""
	default:
		return false
	}
}

func validateTransport(v MCPDocument) error {
	switch v.Transport {
	case "":
		if v.Command != "" ||
			len(v.Args) != 0 ||
			len(v.Env) != 0 ||
			v.URL != "" ||
			len(v.Headers) != 0 {
			return fmt.Errorf(
				"%w: MCP transport is required when connection fields are present",
				basespec.ErrInvalid,
			)
		}
		return nil

	case TransportStdio:
		if v.Locator == nil && v.Command == "" {
			return fmt.Errorf(
				"%w: inline stdio MCP requires command",
				basespec.ErrInvalid,
			)
		}
		if v.URL != "" || len(v.Headers) != 0 {
			return fmt.Errorf(
				"%w: stdio MCP cannot contain HTTP fields",
				basespec.ErrInvalid,
			)
		}
		return nil

	case TransportStreamableHTTP:
		if v.Locator == nil && v.URL == "" {
			return fmt.Errorf(
				"%w: inline HTTP MCP requires url",
				basespec.ErrInvalid,
			)
		}
		if v.Command != "" || len(v.Args) != 0 || len(v.Env) != 0 {
			return fmt.Errorf(
				"%w: HTTP MCP cannot contain stdio fields",
				basespec.ErrInvalid,
			)
		}
		return nil

	default:
		return fmt.Errorf(
			"%w: unsupported MCP transport %q",
			basespec.ErrInvalid,
			v.Transport,
		)
	}
}

func (v Auth) Validate() error {
	m := v.Mode
	switch m {
	case HTTPAuthModeNone, HTTPAuthModeAPIKey, HTTPAuthModeOAuth, HTTPAuthModeClientCredentials:
	default:
		return fmt.Errorf(
			"%w: unsupported MCP auth mode %q",
			basespec.ErrInvalid,
			v.Mode,
		)
	}
	if v.ClientCredentialsInput != "" {
		if err := basespec.LogicalName(v.ClientCredentialsInput).Validate(); err != nil {
			return fmt.Errorf("MCP auth clientCredentialsInput: %w", err)
		}
		if v.Mode != HTTPAuthModeOAuth && v.Mode != HTTPAuthModeClientCredentials {
			return fmt.Errorf(
				"%w: MCP clientCredentialsInput requires oauth or clientCredentials mode",
				basespec.ErrInvalid,
			)
		}
	}
	if v.ClientIDMetadataDocumentURL != "" {
		if err := declaration.ValidateAbsoluteURL(
			"MCP client ID metadata document URL",
			v.ClientIDMetadataDocumentURL,
		); err != nil {
			return err
		}
		if v.Mode != HTTPAuthModeOAuth {
			return fmt.Errorf(
				"%w: MCP client ID metadata document requires oauth mode",
				basespec.ErrInvalid,
			)
		}
	}
	return nil
}

func (v Install) Validate() error {
	if err := basespec.ValidateOptionalText(
		"MCP install note",
		v.Note,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	for name, input := range v.Inputs {
		if err := basespec.LogicalName(name).Validate(); err != nil {
			return fmt.Errorf("MCP installation input name: %w", err)
		}
		if err := input.Validate(); err != nil {
			return fmt.Errorf("MCP installation input %q: %w", name, err)
		}
	}
	return declaration.ValidateTextSlice(
		"MCP install allowEnvironment",
		v.AllowEnvironment,
		basespec.MaxLogicalNameBytes,
	)
}

func (v InstallInput) Validate() error {
	switch v.Kind {
	case "text", "secret", "path", secretOAuthClientCredentials:
	default:
		return fmt.Errorf(
			"%w: unsupported MCP installation input kind %q",
			basespec.ErrInvalid,
			v.Kind,
		)
	}
	for label, value := range map[string]string{
		"label":       v.Label,
		"description": v.Description,
		"note":        v.Note,
		"placeholder": v.Placeholder,
	} {
		if err := basespec.ValidateOptionalText(
			"MCP installation input "+label,
			value,
			basespec.MaxDescriptionBytes,
		); err != nil {
			return err
		}
	}
	if v.Default != nil {
		if v.Kind == "secret" || v.Kind == secretOAuthClientCredentials {
			return fmt.Errorf(
				"%w: secret MCP installation inputs cannot declare defaults",
				basespec.ErrInvalid,
			)
		}
		if err := declaration.ValidateJSONValue(
			"MCP installation input default",
			*v.Default,
			basespec.MaxLocalDataBytes,
		); err != nil {
			return err
		}
	}
	if v.ClientSecretRequired != nil &&
		v.Kind != secretOAuthClientCredentials {
		return fmt.Errorf(
			"%w: clientSecretRequired is valid only for oauthClientCredentials",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func (v ConnectionProfile) Validate() error {
	switch {
	case v.Stdio == nil && v.HTTP == nil:
		return fmt.Errorf(
			"%w: MCP connection profile requires stdio or http",
			basespec.ErrInvalid,
		)
	case v.Stdio != nil && v.HTTP != nil:
		return fmt.Errorf(
			"%w: MCP connection profile cannot contain both stdio and http",
			basespec.ErrInvalid,
		)
	}
	for _, platform := range v.Platforms {
		switch platform {
		case "linux", "darwin", "windows":
		default:
			return fmt.Errorf(
				"%w: unsupported MCP connection profile platform %q",
				basespec.ErrInvalid,
				platform,
			)
		}
	}
	if v.Stdio != nil {
		if err := declaration.ValidateTextSlice(
			"MCP profile stdio args",
			v.Stdio.Args,
			basespec.MaxURIBytes,
		); err != nil {
			return err
		}
		if err := declaration.ValidateStringMap(
			"MCP profile stdio env",
			v.Stdio.Env,
		); err != nil {
			return err
		}
		if err := declaration.ValidateTextSlice(
			"MCP profile removeEnv",
			v.Stdio.RemoveEnv,
			basespec.MaxLogicalNameBytes,
		); err != nil {
			return err
		}
	}
	if v.HTTP != nil {
		if v.HTTP.URL != "" {
			if err := validateMCPURL(v.HTTP.URL); err != nil {
				return err
			}
		}
		if err := declaration.ValidateStringMap(
			"MCP profile HTTP headers",
			v.HTTP.Headers,
		); err != nil {
			return err
		}
		if err := declaration.ValidateTextSlice(
			"MCP profile removeHeaders",
			v.HTTP.RemoveHeaders,
			basespec.MaxURIBytes,
		); err != nil {
			return err
		}
	}
	return nil
}
