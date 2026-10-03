package skillv1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	SkillType          = declaration.TypeSkill
	SkillSchemaID      = "artifact.skill.v1"
	SkillSchemaVersion = declaration.SchemaVersionV1
)

//go:embed skill-v1.schema.json
var schemaJSON []byte

var compiledSkillSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var SkillSchemaKey = schemaModel.ArtifactKey(
	artifactModel.ArtifactKind(SkillType),
	schemaModel.SchemaID(SkillSchemaID),
	SkillSchemaVersion,
)

type SkillDocument struct {
	declaration.Header

	License      string                   `json:"license,omitempty"`
	Insert       declaration.InsertTarget `json:"insert,omitempty"`
	AllowedTools []declaration.Entry      `json:"allowedTools,omitempty"`
}

func SkillJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeSkillJSON(raw []byte) (SkillDocument, error) {
	return decodeSkill(raw)
}

func DecodeSkillEntry(
	entry declaration.Entry,
) (SkillDocument, error) {
	var value SkillDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledSkillSchema,
		&value,
	); err != nil {
		return SkillDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return SkillDocument{}, err
	}
	return value, nil
}

func decodeSkill(
	raw []byte,
) (SkillDocument, error) {
	var value SkillDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledSkillSchema,
		&value,
	); err != nil {
		return SkillDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return SkillDocument{}, err
	}
	return value, nil
}

func (v SkillDocument) Clone() (SkillDocument, error) {
	output := v
	output.Header = v.Header.Clone()
	output.AllowedTools = declaration.CloneEntries(v.AllowedTools)
	return output, nil
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
	return v.validate()
}

func (v SkillDocument) ValidateEntry() error {
	return v.validate()
}

func (v SkillDocument) validate() error {
	if err := declaration.ValidateDocument(
		compiledSkillSchema,
		v,
	); err != nil {
		return fmt.Errorf("skill schema: %w", err)
	}
	return v.validateFields()
}

func (v SkillDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: SkillType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if v.Locator == nil {
		return fmt.Errorf(
			"%w: concrete Skill requires a Skill package locator",
			spec.ErrInvalid,
		)
	}
	if v.License != "" {
		if err := spec.ValidateRequiredText(
			"Skill license",
			v.License,
			spec.MaxURIBytes,
		); err != nil {
			return err
		}
	}
	if v.Insert != "" {
		if err := v.Insert.Validate(); err != nil {
			return err
		}
	}
	return declaration.ValidateMembersWithoutRelationshipBehavior(
		"Skill allowedTools",
		v.AllowedTools,
		declaration.TypeTool,
	)
}
