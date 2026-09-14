package modelv1

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
	ModelType          = declaration.TypeModel
	ModelSchemaID      = "artifact.model.v1"
	ModelSchemaVersion = declaration.APIVersionV1
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

	Model      string                     `json:"model,omitempty"`
	Parameters map[string]json.RawMessage `json:"parameters,omitempty"`
}

func ModelJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeModelJSON(raw []byte) (ModelDocument, error) {
	return decodeModel(raw, true)
}

func DecodeModelEntry(
	entry declaration.Entry,
) (ModelDocument, error) {
	var value ModelDocument
	if err := entry.DecodeInto(&value); err != nil {
		return ModelDocument{}, err
	}
	if err := value.ValidateEntry(); err != nil {
		return ModelDocument{}, err
	}
	return value, nil
}

func decodeModel(
	raw []byte,
	requireName bool,
) (ModelDocument, error) {
	var value ModelDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledModelSchema,
		&value,
	); err != nil {
		return ModelDocument{}, err
	}
	if err := value.validate(requireName); err != nil {
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
	return v.validate(true)
}

func (v ModelDocument) ValidateEntry() error {
	return v.validate(false)
}

func (v ModelDocument) validate(requireName bool) error {
	if err := declaration.ValidateDocument(
		compiledModelSchema,
		v,
	); err != nil {
		return fmt.Errorf("model schema: %w", err)
	}
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: ModelType,
		APIVersion:   ModelSchemaVersion,
		RequireName:  requireName,
	}); err != nil {
		return err
	}
	if v.Model != "" {
		if err := basespec.ValidateRequiredText(
			"Model provider-qualified name",
			v.Model,
			basespec.MaxURIBytes,
		); err != nil {
			return err
		}
	}
	return declaration.ValidateRawMessageMap(
		"Model parameters",
		v.Parameters,
	)
}
