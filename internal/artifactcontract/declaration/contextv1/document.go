package contextv1

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
	ContextType          = declaration.TypeContext
	ContextSchemaID      = "artifact.context.v1"
	ContextSchemaVersion = declaration.APIVersionV1
)

//go:embed context-v1.schema.json
var schemaJSON []byte

var compiledContextSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var ContextSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(ContextType),
	schema.SchemaID(ContextSchemaID),
	ContextSchemaVersion,
)

type ContextDocument struct {
	declaration.Header

	Content   *string  `json:"content,omitempty"`
	MediaType string   `json:"mediaType,omitempty"`
	Include   []string `json:"include,omitempty"`
	Exclude   []string `json:"exclude,omitempty"`
}

func ContextJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeContextJSON(raw []byte) (ContextDocument, error) {
	return decodeContext(raw, true)
}

func DecodeContextEntry(
	entry declaration.Entry,
) (ContextDocument, error) {
	var value ContextDocument
	if err := entry.DecodeInto(&value); err != nil {
		return ContextDocument{}, err
	}
	if err := value.ValidateEntry(); err != nil {
		return ContextDocument{}, err
	}
	return value, nil
}

func decodeContext(
	raw []byte,
	requireName bool,
) (ContextDocument, error) {
	var value ContextDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledContextSchema,
		&value,
	); err != nil {
		return ContextDocument{}, err
	}
	if err := value.validate(requireName); err != nil {
		return ContextDocument{}, err
	}
	return value, nil
}

func (v ContextDocument) Clone() (ContextDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v ContextDocument) Canonicalize() (
	ContextDocument,
	error,
) {
	return v.Clone()
}

func (v ContextDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v ContextDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v ContextDocument) Validate() error {
	return v.validate(true)
}

func (v ContextDocument) ValidateEntry() error {
	return v.validate(false)
}

func (v ContextDocument) validate(requireName bool) error {
	if err := declaration.ValidateDocument(
		compiledContextSchema,
		v,
	); err != nil {
		return fmt.Errorf("context schema: %w", err)
	}
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: ContextType,
		APIVersion:   ContextSchemaVersion,
		RequireName:  requireName,
	}); err != nil {
		return err
	}
	if err := declaration.ValidateOptionalContent(v.Content); err != nil {
		return err
	}
	if err := declaration.ValidateOptionalMediaType(v.MediaType); err != nil {
		return err
	}
	if err := basespec.ValidatePathPatterns(
		"Context include",
		v.Include,
	); err != nil {
		return err
	}
	return basespec.ValidatePathPatterns("Context exclude", v.Exclude)
}
