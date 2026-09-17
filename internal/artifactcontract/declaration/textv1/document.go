package textv1

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
	TextType          = declaration.TypeText
	TextSchemaID      = "artifact.text.v1"
	TextSchemaVersion = declaration.SchemaVersionV1
)

//go:embed text-v1.schema.json
var schemaJSON []byte

var compiledTextSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var TextSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(TextType),
	schema.SchemaID(TextSchemaID),
	TextSchemaVersion,
)

type TextDocument struct {
	declaration.Header

	Insert    declaration.InsertTarget `json:"insert"`
	Content   *string                  `json:"content,omitempty"`
	MediaType string                   `json:"mediaType,omitempty"`
	Include   []string                 `json:"include,omitempty"`
	Exclude   []string                 `json:"exclude,omitempty"`
}

func TextJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeTextJSON(raw []byte) (TextDocument, error) {
	var value TextDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledTextSchema,
		&value,
	); err != nil {
		return TextDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return TextDocument{}, err
	}
	return value, nil
}

func DecodeTextEntry(entry declaration.Entry) (TextDocument, error) {
	var value TextDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledTextSchema,
		&value,
	); err != nil {
		return TextDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return TextDocument{}, err
	}
	return value, nil
}

func (v TextDocument) Clone() (TextDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v TextDocument) Canonicalize() (TextDocument, error) {
	return v.Clone()
}

func (v TextDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v TextDocument) CalculatedDigest() (cryptoutil.Digest, error) {
	return declaration.DocumentDigest(v)
}

func (v TextDocument) Validate() error {
	if err := declaration.ValidateDocument(
		compiledTextSchema,
		v,
	); err != nil {
		return fmt.Errorf("text schema: %w", err)
	}
	return v.validateFields()
}

func (v TextDocument) ValidateEntry() error {
	return v.Validate()
}

func (v TextDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: TextType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if err := v.Insert.Validate(); err != nil {
		return err
	}
	switch {
	case v.Content != nil && v.Locator != nil:
		return fmt.Errorf(
			"%w: Text cannot contain both content and locator",
			basespec.ErrInvalid,
		)
	case v.Content == nil && v.Locator == nil:
		return fmt.Errorf(
			"%w: Text requires content or locator",
			basespec.ErrInvalid,
		)
	case v.Content != nil &&
		(len(v.Include) != 0 || len(v.Exclude) != 0):
		return fmt.Errorf(
			"%w: inline Text cannot contain include or exclude patterns",
			basespec.ErrInvalid,
		)
	}
	if err := declaration.ValidateOptionalContent(v.Content); err != nil {
		return err
	}
	if err := declaration.ValidateOptionalMediaType(v.MediaType); err != nil {
		return err
	}
	if err := basespec.ValidatePathPatterns("Text include", v.Include); err != nil {
		return err
	}
	return basespec.ValidatePathPatterns("Text exclude", v.Exclude)
}
