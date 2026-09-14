package skillv1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	SkillType          = artifactcontract.TypeSkill
	SkillSchemaID      = "artifact.skill.v1"
	SkillSchemaVersion = artifactcontract.APIVersionV1
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
	artifactcontract.Header

	License      string                   `json:"license,omitempty"`
	AllowedTools []artifactcontract.Entry `json:"allowedTools,omitempty"`
}

func SkillJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeSkillJSON(raw []byte) (SkillDocument, error) {
	return decodeSkill(raw, true)
}

func DecodeSkillEntry(
	entry artifactcontract.Entry,
) (SkillDocument, error) {
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return SkillDocument{}, err
	}
	return decodeSkill(raw, false)
}

func decodeSkill(
	raw []byte,
	requireName bool,
) (SkillDocument, error) {
	var value SkillDocument
	if err := artifactcontract.DecodeDocumentInto(
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
	return artifactcontract.CanonicalDocumentJSON(v)
}

func (v SkillDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return artifactcontract.DocumentDigest(v)
}

func (v SkillDocument) Validate() error {
	return v.validate(true)
}

func (v SkillDocument) ValidateEntry() error {
	return v.validate(false)
}

func (v SkillDocument) validate(requireName bool) error {
	if err := artifactcontract.ValidateDocument(
		compiledSkillSchema,
		v,
	); err != nil {
		return fmt.Errorf("skill schema: %w", err)
	}
	if err := v.Header.Validate(artifactcontract.HeaderValidation{
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
	return artifactcontract.ValidateEntryTypes(
		"Skill allowedTools",
		v.AllowedTools,
		artifactcontract.TypeTool,
	)
}
