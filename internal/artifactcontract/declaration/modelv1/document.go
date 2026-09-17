package modelv1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
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

type ModelDocument struct {
	declaration.Header

	Model           string   `json:"model"`
	SystemPrompt    *string  `json:"systemPrompt,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
}

func ModelJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeModelJSON(raw []byte) (ModelDocument, error) {
	return decodeModel(raw)
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

func decodeModel(
	raw []byte,
) (ModelDocument, error) {
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

func (v ModelDocument) Clone() (ModelDocument, error) {
	return jsonutil.CloneJSON(v)
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
	return v.validate()
}

func (v ModelDocument) ValidateEntry() error {
	return v.validate()
}

func (v ModelDocument) validate() error {
	if err := declaration.ValidateDocument(
		compiledModelSchema,
		v,
	); err != nil {
		return fmt.Errorf("model schema: %w", err)
	}
	return v.validateFields()
}

func (v ModelDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: ModelType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Model provider-qualified name",
		v.Model,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := declaration.ValidateOptionalContent(v.SystemPrompt); err != nil {
		return err
	}
	if v.Temperature != nil &&
		(*v.Temperature < 0 || *v.Temperature > 2) {
		return fmt.Errorf(
			"%w: Model temperature must be between 0 and 2",
			basespec.ErrInvalid,
		)
	}
	if v.MaxOutputTokens != nil && *v.MaxOutputTokens <= 0 {
		return fmt.Errorf(
			"%w: Model maxOutputTokens must be positive",
			basespec.ErrInvalid,
		)
	}
	return nil
}
