package toolv1

import (
	_ "embed"
	"encoding/json"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

const (
	ToolType          declaration.Type = "tool"
	ToolSchemaID                       = "artifact.tool.v1"
	ToolSchemaVersion                  = declaration.SchemaVersionV1
)

// ImplementationKind identifies the runtime owner of a Tool declaration.
//
// Go Tools are invoked by FlexiGPT's local Tool runtime. SDK Tools are
// provider-native inference ToolChoices and are hydrated for an inference
// request; they are never routed through the local Go Tool runtime.
type ImplementationKind string

const (
	ImplementationKindGo  ImplementationKind = "go"
	ImplementationKindSDK ImplementationKind = "sdk"
)

// SDKToolType identifies the provider-native ToolChoice representation.
type SDKToolType string

const (
	SDKToolTypeFunction  SDKToolType = "function"
	SDKToolTypeCustom    SDKToolType = "custom"
	SDKToolTypeWebSearch SDKToolType = "webSearch"
)

//go:embed tool-v1.schema.json
var schemaJSON []byte

var compiledToolSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var ToolSchemaKey = schemaModel.ArtifactKey(
	artifactModel.ArtifactKind(ToolType),
	schemaModel.SchemaID(ToolSchemaID),
	ToolSchemaVersion,
)

// ToolImplementation declares one of the two supported Tool implementations.
//
// Go uses Function. SDK uses SDKType and SDKToolType. The mutually exclusive
// shape is enforced by both the declaration schema and Validate.
type ToolImplementation struct {
	Kind ImplementationKind `json:"kind"`

	Function string `json:"function,omitempty"`

	SDKType     string      `json:"sdkType,omitempty"`
	SDKToolType SDKToolType `json:"sdkToolType,omitempty"`
}

type ToolDocument struct {
	declaration.Header

	Version     spec.LogicalVersion `json:"version"`
	Tags        []string            `json:"tags,omitempty"`
	AutoExecute bool                `json:"autoExecute"`

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

func DecodeAdmittedToolJSON(
	raw []byte,
) (ToolDocument, error) {
	entry, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return ToolDocument{}, err
	}
	return DecodeAdmittedToolEntry(entry)
}

func DecodeAdmittedToolEntry(
	entry declaration.Entry,
) (ToolDocument, error) {
	var value ToolDocument
	if err := declaration.DecodeAdmittedEntryInto(
		entry,
		&value,
	); err != nil {
		return ToolDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return ToolDocument{}, err
	}
	return value, nil
}

func DefinitionForDocument(
	document ToolDocument,
) (definitionModel.Definition, error) {
	if err := document.Validate(); err != nil {
		return definitionModel.Definition{}, err
	}
	body, err := document.CanonicalJSON()
	if err != nil {
		return definitionModel.Definition{}, err
	}
	displayName := document.DisplayName
	if displayName == "" {
		displayName = document.Name
	}
	return definitionModel.Canonicalize(definitionModel.Definition{
		Kind:           artifactModel.ArtifactKind(ToolType),
		SchemaID:       ToolSchemaKey.SchemaID,
		SchemaVersion:  ToolSchemaKey.SchemaVersion,
		LogicalName:    spec.LogicalName(document.Name),
		LogicalVersion: document.Version,
		DisplayName:    displayName,
		Description:    document.Description,
		Labels:         declaration.CloneStringMap(document.Labels),
		Body:           body,
	})
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
			spec.ErrInvalid,
		)
	}
	if err := spec.ValidatePortableName(
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
			spec.ErrInvalid,
		)
	}
	return nil
}

func (v ToolImplementation) Validate() error {
	switch v.Kind {
	case ImplementationKindGo:
		if err := spec.ValidateRequiredText(
			"Go Tool function",
			v.Function,
			spec.MaxURIBytes,
		); err != nil {
			return err
		}
		if v.SDKType != "" || v.SDKToolType != "" {
			return fmt.Errorf(
				"%w: Go Tool implementation cannot contain SDK fields",
				spec.ErrInvalid,
			)
		}
		return nil

	case ImplementationKindSDK:
		if err := spec.ValidateIdentifier(
			"SDK Tool type",
			v.SDKType,
			spec.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := v.SDKToolType.Validate(); err != nil {
			return err
		}
		if v.Function != "" {
			return fmt.Errorf(
				"%w: SDK Tool implementation cannot contain Go function",
				spec.ErrInvalid,
			)
		}
		return nil

	default:
		return fmt.Errorf(
			"%w: unsupported Tool implementation kind %q",
			spec.ErrInvalid,
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
			spec.ErrInvalid,
			v,
		)
	}
}

func validateTags(values []string) error {
	if len(values) > spec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: Tool tags exceed %d entries",
			spec.ErrInvalid,
			spec.MaxDefinitionDependencies,
		)
	}

	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if err := spec.ValidateRequiredText(
			"Tool tag",
			value,
			spec.MaxLogicalNameBytes,
		); err != nil {
			return fmt.Errorf("tool tags[%d]: %w", index, err)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf(
				"%w: Tool tag %q is repeated",
				spec.ErrIdentityConflict,
				value,
			)
		}
		seen[value] = struct{}{}
	}
	return nil
}
