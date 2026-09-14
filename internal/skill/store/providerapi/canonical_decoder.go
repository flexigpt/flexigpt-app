package providerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/skillv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

// CanonicalDecoder handles standalone canonical JSON skill declarations.
// YAML normalization belongs to the future canonical YAML format adapter.
type CanonicalDecoder struct {
	canonicalizer providerapi.ExpectedCanonicalizer
}

func NewCanonicalDecoder() *CanonicalDecoder {
	return &CanonicalDecoder{}
}

func (*CanonicalDecoder) ID() basespec.DecoderID {
	return skillDomain.CanonicalDecoderID
}

func (*CanonicalDecoder) Revision() string {
	return skillDomain.SkillSchemaVersion
}

func (*CanonicalDecoder) RequiredSchemaKeys() []schema.Key {
	return []schema.Key{skillv1.SkillSchemaKey}
}

func (d *CanonicalDecoder) BindExpectedCanonicalizer(
	canonicalizer providerapi.SchemaCatalog,
) error {
	if canonicalizer == nil {
		return fmt.Errorf(
			"%w: Skill canonical decoder schema catalog is nil",
			basespec.ErrInvalid,
		)
	}
	d.canonicalizer = canonicalizer
	return nil
}

func (*CanonicalDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	var header struct {
		Type declaration.Type `json:"type"`
	}
	if err := json.Unmarshal(candidate.Content, &header); err != nil {
		return providerapi.RecognitionNone
	}
	if header.Type != declaration.TypeSkill {
		return providerapi.RecognitionNone
	}
	return providerapi.RecognitionPreferred
}

func (d *CanonicalDecoder) Decode(
	ctx context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	if d == nil || d.canonicalizer == nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "artifact.skill-schema-unavailable",
			Message:  "Skill canonical declaration schema is unavailable",
			Location: &diagnostic.Location{
				Locator: candidate.Locator,
			},
		}}
	}

	parsed, err := d.canonicalizer.CanonicalizeExpected(
		ctx,
		skillv1.SkillSchemaKey,
		candidate.Content,
	)
	if err != nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "artifact.skill-invalid",
			Message:  diagnostic.BoundedMessage(err.Error()),
			Location: &diagnostic.Location{
				Locator: candidate.Locator,
			},
		}}
	}

	document, err := skillv1.DecodeSkillJSON(parsed.Raw)
	if err != nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "artifact.skill-invalid",
			Message:  diagnostic.BoundedMessage(err.Error()),
			Location: &diagnostic.Location{
				Locator: candidate.Locator,
			},
		}}
	}
	definitionValue, err := skillDomain.DefinitionForSkillDeclaration(
		document,
	)
	if err != nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "artifact.skill-definition-invalid",
			Message:  diagnostic.BoundedMessage(err.Error()),
			Location: &diagnostic.Location{
				Locator: candidate.Locator,
			},
		}}
	}
	return []providerapi.Decoded{{
		Definition: definitionValue,
	}}, nil
}
