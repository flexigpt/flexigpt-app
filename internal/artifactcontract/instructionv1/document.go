package instructionv1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	InstructionType          = artifactcontract.TypeInstruction
	InstructionSchemaID      = "artifact.instruction.v1"
	InstructionSchemaVersion = artifactcontract.APIVersionV1
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
	artifactcontract.Header

	Content   *string `json:"content,omitempty"`
	MediaType string  `json:"mediaType,omitempty"`
}

func InstructionJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeInstructionJSON(
	raw []byte,
) (InstructionDocument, error) {
	return decodeInstruction(raw, true)
}

func DecodeInstructionEntry(
	entry artifactcontract.Entry,
) (InstructionDocument, error) {
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return InstructionDocument{}, err
	}
	return decodeInstruction(raw, false)
}

func decodeInstruction(
	raw []byte,
	requireName bool,
) (InstructionDocument, error) {
	var value InstructionDocument
	if err := artifactcontract.DecodeDocumentInto(
		raw,
		compiledInstructionSchema,
		&value,
	); err != nil {
		return InstructionDocument{}, err
	}
	if err := value.validate(requireName); err != nil {
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
	return artifactcontract.CanonicalDocumentJSON(v)
}

func (v InstructionDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return artifactcontract.DocumentDigest(v)
}

func (v InstructionDocument) Validate() error {
	return v.validate(true)
}

func (v InstructionDocument) ValidateEntry() error {
	return v.validate(false)
}

func (v InstructionDocument) validate(
	requireName bool,
) error {
	if err := artifactcontract.ValidateDocument(
		compiledInstructionSchema,
		v,
	); err != nil {
		return fmt.Errorf("instruction schema: %w", err)
	}
	if err := v.Header.Validate(artifactcontract.HeaderValidation{
		ExpectedType: InstructionType,
		APIVersion:   InstructionSchemaVersion,
		RequireName:  requireName,
	}); err != nil {
		return err
	}
	if err := artifactcontract.ValidateOptionalContent(v.Content); err != nil {
		return err
	}
	return artifactcontract.ValidateOptionalMediaType(v.MediaType)
}
