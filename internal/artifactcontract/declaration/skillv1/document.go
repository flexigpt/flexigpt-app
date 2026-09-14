package skillv1

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
	SkillType          = declaration.TypeSkill
	SkillSchemaID      = "artifact.skill.v1"
	SkillSchemaVersion = declaration.APIVersionV1
)

//go:embed skill-v1.schema.json
var schemaJSON []byte

var compiledSkillSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var SkillSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(SkillType),
	schema.SchemaID(SkillSchemaID),
	SkillSchemaVersion,
)

type SkillDocument struct {
	declaration.Header

	License      string              `json:"license,omitempty"`
	AllowedTools []declaration.Entry `json:"allowedTools,omitempty"`
}

func SkillJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeSkillJSON(raw []byte) (SkillDocument, error) {
	return decodeSkill(raw, true)
}

func DecodeSkillEntry(
	entry declaration.Entry,
) (SkillDocument, error) {
	var value SkillDocument
	if err := entry.DecodeInto(&value); err != nil {
		return SkillDocument{}, err
	}
	if err := value.ValidateEntry(); err != nil {
		return SkillDocument{}, err
	}
	return value, nil
}

func decodeSkill(
	raw []byte,
	requireName bool,
) (SkillDocument, error) {
	var value SkillDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledSkillSchema,
		&value,
	); err != nil {
		return SkillDocument{}, err
	}
	if err := value.validate(requireName); err != nil {
		return SkillDocument{}, err
	}
	return value, nil
}

func (v SkillDocument) Clone() (SkillDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v SkillDocument) Canonicalize() (SkillDocument, error) {
	return v.Clone()
}

func (v SkillDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v SkillDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v SkillDocument) Validate() error {
	return v.validate(true)
}

func (v SkillDocument) ValidateEntry() error {
	return v.validate(false)
}

func (v SkillDocument) validate(requireName bool) error {
	if err := declaration.ValidateDocument(
		compiledSkillSchema,
		v,
	); err != nil {
		return fmt.Errorf("skill schema: %w", err)
	}
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: SkillType,
		APIVersion:   SkillSchemaVersion,
		RequireName:  requireName,
	}); err != nil {
		return err
	}
	if v.License != "" {
		if err := basespec.ValidateRequiredText(
			"Skill license",
			v.License,
			basespec.MaxURIBytes,
		); err != nil {
			return err
		}
	}
	if len(v.AllowedTools) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: Skill allowedTools exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDefinitionDependencies,
		)
	}
	return declaration.ValidateEntryTypes(
		"Skill allowedTools",
		v.AllowedTools,
		declaration.TypeTool,
	)
}
