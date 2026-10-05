package skillv1

import (
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() interpretation.Registration {
	return interpretation.Registration{
		DeclarationType:  SkillType,
		SchemaKey:        SkillSchemaKey,
		SelectorEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeSkillEntry(entry)
			return err
		},
		Relationships: skillRelationships,
		LocatorCandidates: func(
			target spec.Locator,
		) ([]spec.Locator, error) {
			if err := target.Validate(false); err != nil {
				return nil, err
			}
			if path.Base(string(target)) == "SKILL.md" {
				return []spec.Locator{target}, nil
			}
			document := spec.Locator(
				path.Join(string(target), "SKILL.md"),
			)
			if err := document.Validate(false); err != nil {
				return nil, err
			}
			return []spec.Locator{target, document}, nil
		},
	}
}

func skillRelationships(
	entry declaration.Entry,
) ([]interpretation.Relationship, error) {
	document, err := DecodeSkillEntry(entry)
	if err != nil {
		return nil, err
	}
	return interpretation.MemberRelationships(
		[]string{"allowedTools"},
		document.AllowedTools,
		interpretation.MemberOptions{AllowSelector: true},
	)
}
