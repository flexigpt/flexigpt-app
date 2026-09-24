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

type ImplementationKind string

const (
	ImplementationKindGo  ImplementationKind = "go"
	ImplementationKindSDK ImplementationKind = "sdk"
)

type SDKToolType string

const (
	SDKToolTypeFunction  SDKToolType = "function"
	SDKToolTypeCustom    SDKToolType = "custom"
	SDKToolTypeWebSearch SDKToolType = "webSearch"
)

//go:embed tool-v1.schema.json
var schemaJSON []byte

var compiledToolSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var ToolSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(ToolType),
	schema.SchemaID(ToolSchemaID),
	ToolSchemaVersion,
)

type ToolImplementation struct {
	Kind ImplementationKind `json:"kind"`

	Function string `json:"function,omitempty"`

	SDKType     string      `json:"sdkType,omitempty"`
	SDKToolType SDKToolType `json:"sdkToolType,omitempty"`
}

type ToolDocument struct {
	declaration.Header

	Version     basespec.LogicalVersion `json:"version"`
	Tags        []string                `json:"tags,omitempty"`
	AutoExecute bool                    `json:"autoExecute"`

	InputSchema   json.RawMessage  `json:"inputSchema"`
	UserArgSchema *json.RawMessage `json:"userArgSchema,omitempty"`
	OutputSchema  *json.RawMessage `json:"outputSchema,omitempty"`

	Implementation ToolImplementation `json:"implementation"`
}

func ToolJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeToolJSON(raw []byte) (ToolDocument, error) {
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
	if err := declaration.ValidateDocument(
		compiledToolSchema,
		v,
	); err != nil {
		return fmt.Errorf("tool schema: %w", err)
	}
	return v.validateFields()
}

func (v ToolDocument) ValidateEntry() error {
	return v.Validate()
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
			"%w: Tool declarations do not support locator",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidatePortableName(
		"Tool version",
		string(v.Version),
	); err != nil {
		return err
	}
	if err := validateTags(v.Tags); err != nil {
		return err
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
	if err := v.Implementation.Validate(); err != nil {
		return err
	}
	if v.Implementation.Kind != ImplementationKindSDK &&
		v.UserArgSchema != nil {
		return fmt.Errorf(
			"%w: Tool userArgSchema is supported only for sdk Tools",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func (v ToolImplementation) Validate() error {
	switch v.Kind {
	case ImplementationKindGo:
		if err := basespec.ValidateRequiredText(
			"Go Tool function",
			v.Function,
			basespec.MaxURIBytes,
		); err != nil {
			return err
		}
		if v.SDKType != "" || v.SDKToolType != "" {
			return fmt.Errorf(
				"%w: Go Tool implementation cannot contain SDK fields",
				basespec.ErrInvalid,
			)
		}
		return nil

	case ImplementationKindSDK:
		if err := basespec.ValidateIdentifier(
			"SDK Tool type",
			v.SDKType,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := v.SDKToolType.Validate(); err != nil {
			return err
		}
		if v.Function != "" {
			return fmt.Errorf(
				"%w: SDK Tool implementation cannot contain Go function",
				basespec.ErrInvalid,
			)
		}
		return nil

	default:
		return fmt.Errorf(
			"%w: unsupported Tool implementation kind %q",
			basespec.ErrInvalid,
			v.Kind,
		)
	}
}

func (v SDKToolType) Validate() error {
	switch v {
	case SDKToolTypeFunction,
		SDKToolTypeCustom,
		SDKToolTypeWebSearch:
		return nil
	default:
		return fmt.Errorf(
			"%w: unsupported SDK Tool type %q",
			basespec.ErrInvalid,
			v,
		)
	}
}

func validateTags(values []string) error {
	if len(values) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: Tool tags exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDefinitionDependencies,
		)
	}

	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if err := basespec.ValidateRequiredText(
			"Tool tag",
			value,
			basespec.MaxLogicalNameBytes,
		); err != nil {
			return fmt.Errorf("tool tags[%d]: %w", index, err)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf(
				"%w: Tool tag %q is repeated",
				basespec.ErrIdentityConflict,
				value,
			)
		}
		seen[value] = struct{}{}
	}
	return nil
}
