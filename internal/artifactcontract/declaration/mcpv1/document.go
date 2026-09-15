package mcpv1

import (
	_ "embed"
	"encoding/json"
	"fmt"

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
	MCPSchemaVersion = declaration.APIVersionV1
)

type Transport string

const (
	TransportStdio          Transport = "stdio"
	TransportStreamableHTTP Transport = "streamable-http"
	TransportSSE            Transport = "sse"
)

//go:embed mcp-v1.schema.json
var schemaJSON []byte

var compiledMCPSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var MCPSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(MCPType),
	schema.SchemaID(MCPSchemaID),
	MCPSchemaVersion,
)

type Include struct {
	Tools     []string `json:"tools,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Prompts   []string `json:"prompts,omitempty"`
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

	Include *Include `json:"include,omitempty"`
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

	return definition.Canonicalize(definition.Definition{
		Kind:          artifact.ArtifactKind(MCPType),
		SchemaID:      MCPSchemaKey.SchemaID,
		SchemaVersion: MCPSchemaKey.SchemaVersion,
		LogicalName:   basespec.LogicalName(input.Name),
		DisplayName:   input.Name,
		Description:   input.Description,
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
		APIVersion:   MCPSchemaVersion,
	}); err != nil {
		return err
	}
	if v.Server != "" {
		if err := basespec.ValidateRequiredText(
			"MCP server selector",
			v.Server,
			basespec.MaxLogicalNameBytes,
		); err != nil {
			return err
		}
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
		if err := declaration.ValidateAbsoluteURL(
			"MCP URL",
			v.URL,
		); err != nil {
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

func validateTransport(v MCPDocument) error {
	switch v.Transport {
	case "":
		if v.Locator != nil &&
			v.Locator.Kind == declaration.LocatorKindCommand {
			if v.URL != "" || len(v.Headers) != 0 {
				return fmt.Errorf(
					"%w: command-located MCP cannot contain HTTP fields",
					basespec.ErrInvalid,
				)
			}
			return nil
		}
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

	case TransportStreamableHTTP, TransportSSE:
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
