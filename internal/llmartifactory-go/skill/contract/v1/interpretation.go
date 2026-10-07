package skillv1

import (
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
)

func Interpretation(
	documents support.Documents,
) coreinterpretation.Registration {
	return coreinterpretation.Registration{
		DeclarationType:  SkillType,
		SchemaKey:        SkillSchemaKey,
		SelectorEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeSkillEntry(entry)
			return err
		},
		ValidateAdmittedEntry: func(entry declaration.Entry) error {
			_, err := DecodeAdmittedSkillEntry(entry)
			return err
		},
		Relationships: skillRelationships,
		LocatorCandidates: func(
			target spec.Locator,
		) ([]spec.Locator, error) {
			if err := target.Validate(false); err != nil {
				return nil, err
			}
			if documents.Matches(target) {
				return []spec.Locator{target}, nil
			}

			output := make([]spec.Locator, 0, len(documents.Files)+1)
			output = append(output, target)
			for _, documentName := range documents.Files {
				document := spec.Locator(path.Join(
					string(target),
					string(documentName),
				))
				if err := document.Validate(false); err != nil {
					return nil, err
				}
				output = append(output, document)
			}
			return output, nil
		},
	}
}

func skillRelationships(
	entry declaration.Entry,
) ([]coreinterpretation.Relationship, error) {
	document, err := DecodeAdmittedSkillEntry(entry)
	if err != nil {
		return nil, err
	}
	return coreinterpretation.MemberRelationships(
		[]string{"allowedTools"},
		document.AllowedTools,
		coreinterpretation.MemberOptions{AllowSelector: true},
	)
}
