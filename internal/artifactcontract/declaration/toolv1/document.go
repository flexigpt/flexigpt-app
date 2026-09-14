package toolv1

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	ToolType          = declaration.TypeTool
	ToolSchemaID      = "artifact.tool.v1"
	ToolSchemaVersion = declaration.APIVersionV1
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

	InputSchema  *json.RawMessage `json:"inputSchema,omitempty"`
	OutputSchema *json.RawMessage `json:"outputSchema,omitempty"`
}

func ToolJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeToolJSON(raw []byte) (ToolDocument, error) {
	return decodeTool(raw, true)
}

func DecodeToolEntry(
	entry declaration.Entry,
) (ToolDocument, error) {
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return ToolDocument{}, err
	}
	return decodeTool(raw, false)
}

func decodeTool(
	raw []byte,
	requireName bool,
) (ToolDocument, error) {
	var value ToolDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledToolSchema,
		&value,
	); err != nil {
		return ToolDocument{}, err
	}
	if err := value.validate(requireName); err != nil {
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
	return v.validate(true)
}

func (v ToolDocument) ValidateEntry() error {
	return v.validate(false)
}

func (v ToolDocument) validate(requireName bool) error {
	if err := declaration.ValidateDocument(
		compiledToolSchema,
		v,
	); err != nil {
		return fmt.Errorf("tool schema: %w", err)
	}
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: ToolType,
		APIVersion:   ToolSchemaVersion,
		RequireName:  requireName,
	}); err != nil {
		return err
	}
	if v.InputSchema != nil {
		if err := declaration.ValidateJSONSchemaValue(
			"Tool inputSchema",
			*v.InputSchema,
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
	return nil
}
