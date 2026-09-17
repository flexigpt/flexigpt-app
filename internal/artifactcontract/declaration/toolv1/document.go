package toolv1

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	ToolType          = declaration.TypeTool
	ToolSchemaID      = "artifact.tool.v1"
	ToolSchemaVersion = declaration.SchemaVersionV1
)

//go:embed tool-v1.schema.json
var schemaJSON []byte

var compiledToolSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var ToolSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(ToolType),
	schema.SchemaID(ToolSchemaID),
	ToolSchemaVersion,
)

type ToolDocument struct {
	declaration.Header

	UserCallable   bool             `json:"userCallable"`
	LLMCallable    bool             `json:"llmCallable"`
	AutoExecute    bool             `json:"autoExecute"`
	LLMToolType    string           `json:"llmToolType"`
	InputSchema    json.RawMessage  `json:"inputSchema"`
	UserArgSchema  *json.RawMessage `json:"userArgSchema,omitempty"`
	OutputSchema   *json.RawMessage `json:"outputSchema,omitempty"`
	Implementation Implementation   `json:"implementation"`
}

type Implementation struct {
	Kind string `json:"kind"`

	Function string `json:"function,omitempty"`

	Request  *HTTPRequest  `json:"request,omitempty"`
	Response *HTTPResponse `json:"response,omitempty"`

	Provider  string `json:"provider,omitempty"`
	Operation string `json:"operation,omitempty"`

	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type HTTPRequest struct {
	Method       string            `json:"method"`
	URLTemplate  string            `json:"urlTemplate"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodyTemplate string            `json:"bodyTemplate,omitempty"`
}

type HTTPResponse struct {
	SuccessCodes   []int  `json:"successCodes"`
	BodyOutputMode string `json:"bodyOutputMode"`
}

func ToolJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeToolJSON(raw []byte) (ToolDocument, error) {
	return decodeTool(raw)
}

func DecodeToolEntry(
	entry declaration.Entry,
) (ToolDocument, error) {
	var value ToolDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledToolSchema,
		&value,
	); err != nil {
		return ToolDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return ToolDocument{}, err
	}
	return value, nil
}

func decodeTool(
	raw []byte,
) (ToolDocument, error) {
	var value ToolDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledToolSchema,
		&value,
	); err != nil {
		return ToolDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return ToolDocument{}, err
	}
	return value, nil
}

func (v ToolDocument) Clone() (ToolDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v ToolDocument) Canonicalize() (ToolDocument, error) {
	return v.Clone()
}

func (v ToolDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v ToolDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v ToolDocument) Validate() error {
	return v.validate()
}

func (v ToolDocument) ValidateEntry() error {
	return v.validate()
}

func (v ToolDocument) validate() error {
	if err := declaration.ValidateDocument(
		compiledToolSchema,
		v,
	); err != nil {
		return fmt.Errorf("tool schema: %w", err)
	}
	return v.validateFields()
}

func (v ToolDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: ToolType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if v.Locator != nil {
		return fmt.Errorf(
			"%w: Tool implementation must use implementation, not a command locator",
			basespec.ErrInvalid,
		)
	}
	if v.LLMToolType != "function" {
		return fmt.Errorf(
			"%w: unsupported Tool llmToolType %q",
			basespec.ErrInvalid,
			v.LLMToolType,
		)
	}
	if err := declaration.ValidateJSONSchemaValue(
		"Tool inputSchema",
		v.InputSchema,
	); err != nil {
		return err
	}
	if v.UserArgSchema != nil {
		if err := declaration.ValidateJSONSchemaValue(
			"Tool userArgSchema",
			*v.UserArgSchema,
		); err != nil {
			return err
		}
	}
	if v.OutputSchema != nil {
		if err := declaration.ValidateJSONSchemaValue(
			"Tool outputSchema",
			*v.OutputSchema,
		); err != nil {
			return err
		}
	}
	return v.Implementation.Validate()
}

func (v Implementation) Validate() error {
	switch v.Kind {
	case "go":
		if v.Function == "" {
			return fmt.Errorf(
				"%w: Go Tool implementation requires function",
				basespec.ErrInvalid,
			)
		}
	case "http":
		if v.Request == nil || v.Response == nil {
			return fmt.Errorf(
				"%w: HTTP Tool implementation requires request and response",
				basespec.ErrInvalid,
			)
		}
		if err := v.Request.Validate(); err != nil {
			return err
		}
		if err := v.Response.Validate(); err != nil {
			return err
		}
	case "sdk":
		if v.Provider == "" || v.Operation == "" {
			return fmt.Errorf(
				"%w: SDK Tool implementation requires provider and operation",
				basespec.ErrInvalid,
			)
		}
	case "command":
		if err := basespec.ValidateRequiredText(
			"Tool implementation command",
			v.Command,
			basespec.MaxURIBytes,
		); err != nil {
			return err
		}
		if err := declaration.ValidateTextSlice(
			"Tool implementation args",
			v.Args,
			basespec.MaxURIBytes,
		); err != nil {
			return err
		}
		if err := declaration.ValidateStringMap(
			"Tool implementation env",
			v.Env,
		); err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"%w: unsupported Tool implementation kind %q",
			basespec.ErrInvalid,
			v.Kind,
		)
	}
	return nil
}

func (v HTTPRequest) Validate() error {
	if err := basespec.ValidateRequiredText(
		"Tool HTTP request method",
		v.Method,
		32,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Tool HTTP request URL template",
		v.URLTemplate,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := declaration.ValidateStringMap(
		"Tool HTTP request headers",
		v.Headers,
	); err != nil {
		return err
	}
	return declaration.ValidateOptionalContent(&v.BodyTemplate)
}

func (v HTTPResponse) Validate() error {
	if len(v.SuccessCodes) == 0 {
		return fmt.Errorf(
			"%w: Tool HTTP response requires successCodes",
			basespec.ErrInvalid,
		)
	}
	for _, code := range v.SuccessCodes {
		if code < 100 || code > 599 {
			return fmt.Errorf(
				"%w: invalid Tool HTTP success code %d",
				basespec.ErrInvalid,
				code,
			)
		}
	}
	switch v.BodyOutputMode {
	case "json", "text", "bytes":
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid Tool HTTP bodyOutputMode %q",
			basespec.ErrInvalid,
			v.BodyOutputMode,
		)
	}
}
