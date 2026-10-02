package modelv1

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schema "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	ModelType          = declaration.TypeModel
	ModelSchemaID      = "artifact.model.v1"
	ModelSchemaVersion = declaration.SchemaVersionV1
)

//go:embed model-v1.schema.json
var schemaJSON []byte

var compiledModelSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var ModelSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(ModelType),
	schema.SchemaID(ModelSchemaID),
	ModelSchemaVersion,
)

// ModelDocument is a source-backed provider-specific model declaration.
//
// Defaults, capabilities, and adapterParameters deliberately remain portable
// canonical JSON objects at the declaration boundary. The Model Store domain
// owns their typed projection and layered merge semantics. This prevents the
// declaration package from importing inference-go types while retaining the
// full schema surface in model-v1.schema.json.
type ModelDocument struct {
	declaration.Header

	Provider          declaration.ArtifactNameReference `json:"provider"`
	ProviderModelID   string                            `json:"providerModelID"`
	Defaults          json.RawMessage                   `json:"defaults,omitempty"`
	Capabilities      json.RawMessage                   `json:"capabilities,omitempty"`
	AdapterParameters json.RawMessage                   `json:"adapterParameters,omitempty"`
}

func ModelJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeModelJSON(raw []byte) (ModelDocument, error) {
	var value ModelDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledModelSchema,
		&value,
	); err != nil {
		return ModelDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return ModelDocument{}, err
	}
	return value, nil
}

func DecodeModelEntry(
	entry declaration.Entry,
) (ModelDocument, error) {
	var value ModelDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledModelSchema,
		&value,
	); err != nil {
		return ModelDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return ModelDocument{}, err
	}
	return value, nil
}

func (v ModelDocument) Clone() (ModelDocument, error) {
	output := v
	output.Header = v.Header.Clone()
	output.Provider = v.Provider.Clone()
	output.Defaults = cloneRawMessage(v.Defaults)
	output.Capabilities = cloneRawMessage(v.Capabilities)
	output.AdapterParameters = cloneRawMessage(v.AdapterParameters)
	return output, nil
}

func (v ModelDocument) Canonicalize() (ModelDocument, error) {
	return v.Clone()
}

func (v ModelDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v ModelDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v ModelDocument) Validate() error {
	if err := declaration.ValidateDocument(
		compiledModelSchema,
		v,
	); err != nil {
		return fmt.Errorf("model schema: %w", err)
	}
	return v.validateFields()
}

func (v ModelDocument) ValidateEntry() error {
	return v.Validate()
}

func (v ModelDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: ModelType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if v.Locator != nil {
		return fmt.Errorf(
			"%w: model declarations do not support locator",
			spec.ErrInvalid,
		)
	}
	if err := v.Provider.Validate(); err != nil {
		return fmt.Errorf("model provider reference: %w", err)
	}
	if err := spec.ValidateRequiredText(
		"Model providerModelID",
		v.ProviderModelID,
		spec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := validatePatchObject("Model defaults", v.Defaults); err != nil {
		return err
	}
	if err := validatePatchObject(
		"Model capabilities",
		v.Capabilities,
	); err != nil {
		return err
	}
	return validateNoSecretObject(
		"Model adapterParameters",
		v.AdapterParameters,
	)
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}

// ValidateDefaultsPatch validates one portable request/defaults patch using
// the same schema and secret checks as a source-backed Model declaration.
//
// The runtime request layer intentionally accepts only the `defaults` shape.
// It cannot change Model identity, Provider linkage, or providerModelID.
func ValidateDefaultsPatch(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return fmt.Errorf("model defaults patch: %w", err)
	}

	envelope := map[string]json.RawMessage{
		"type":            json.RawMessage(`"model"`),
		"name":            json.RawMessage(`"request-patch-model"`),
		"provider":        json.RawMessage(`{"name":"request-patch-provider"}`),
		"providerModelID": json.RawMessage(`"request-patch-model"`),
		"defaults":        canonical,
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	if _, err := DecodeModelJSON(encoded); err != nil {
		return fmt.Errorf("model defaults patch: %w", err)
	}
	return nil
}

func validatePatchObject(
	label string,
	raw json.RawMessage,
) error {
	if len(raw) == 0 {
		return nil
	}
	_, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return validateNoSecretObject(label, raw)
}

// validateNoSecretObject prevents common accidental credential persistence in
// fields intentionally reserved for adapter-specific non-secret data.
//
// It is deliberately conservative. Credentials belong to a settings or secret
// system and are represented here only through external opaque references.
func validateNoSecretObject(
	label string,
	raw json.RawMessage,
) error {
	if len(raw) == 0 {
		return nil
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}

	var value map[string]any
	if err := json.Unmarshal(canonical, &value); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return rejectSecretValues(label, value)
}

func rejectSecretValues(
	label string,
	value any,
) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if modelSecretKey(key) && child != nil {
				return fmt.Errorf(
					"%w: %s contains forbidden credential field %q",
					spec.ErrInvalid,
					label,
					key,
				)
			}
			if err := rejectSecretValues(label, child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := rejectSecretValues(label, child); err != nil {
				return err
			}
		}
	}
	return nil
}

func modelSecretKey(value string) bool {
	normalized := strings.NewReplacer(
		"-",
		"",
		"_",
		"",
		".",
		"",
	).Replace(strings.ToLower(strings.TrimSpace(value)))

	switch normalized {
	case
		"apikey",
		"authorization",
		"accesstoken",
		"refreshtoken",
		"clientsecret",
		"password",
		"credential",
		"credentials",
		"secret":
		return true
	default:
		return strings.HasSuffix(normalized, "apikey") ||
			strings.HasSuffix(normalized, "clientsecret")
	}
}
