package instructionv1

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
	InstructionType          = declaration.TypeInstruction
	InstructionSchemaID      = "artifact.instruction.v1"
	InstructionSchemaVersion = declaration.APIVersionV1
)

//go:embed instruction-v1.schema.json
var schemaJSON []byte

var compiledInstructionSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var InstructionSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(InstructionType),
	schema.SchemaID(InstructionSchemaID),
	InstructionSchemaVersion,
)

type InstructionDocument struct {
	declaration.Header

	Content   *string `json:"content,omitempty"`
	MediaType string  `json:"mediaType,omitempty"`
}

func InstructionJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeInstructionJSON(
	raw []byte,
) (InstructionDocument, error) {
	return decodeInstruction(raw)
}

func DecodeInstructionEntry(
	entry declaration.Entry,
) (InstructionDocument, error) {
	var value InstructionDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledInstructionSchema,
		&value,
	); err != nil {
		return InstructionDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return InstructionDocument{}, err
	}
	return value, nil
}

func decodeInstruction(
	raw []byte,
) (InstructionDocument, error) {
	var value InstructionDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledInstructionSchema,
		&value,
	); err != nil {
		return InstructionDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return InstructionDocument{}, err
	}
	return value, nil
}

func (v InstructionDocument) Clone() (
	InstructionDocument,
	error,
) {
	return jsonutil.CloneJSON(v)
}

func (v InstructionDocument) Canonicalize() (
	InstructionDocument,
	error,
) {
	return v.Clone()
}

func (v InstructionDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v InstructionDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v InstructionDocument) Validate() error {
	return v.validate()
}

func (v InstructionDocument) ValidateEntry() error {
	return v.validate()
}

func (v InstructionDocument) validate() error {
	if err := declaration.ValidateDocument(
		compiledInstructionSchema,
		v,
	); err != nil {
		return fmt.Errorf("instruction schema: %w", err)
	}
	return v.validateFields()
}

func (v InstructionDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: InstructionType,
		APIVersion:   InstructionSchemaVersion,
	}); err != nil {
		return err
	}
	switch {
	case v.Content != nil && v.Locator != nil:
		return fmt.Errorf(
			"%w: Instruction cannot contain both content and locator",
			basespec.ErrInvalid,
		)
	case v.Content == nil && v.Locator == nil:
		return fmt.Errorf(
			"%w: concrete Instruction requires content or locator",
			basespec.ErrInvalid,
		)
	}
	if err := declaration.ValidateOptionalContent(v.Content); err != nil {
		return err
	}
	return declaration.ValidateOptionalMediaType(v.MediaType)
}
