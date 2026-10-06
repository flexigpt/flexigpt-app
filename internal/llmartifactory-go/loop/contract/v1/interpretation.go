package loopv1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() coreinterpretation.Registration {
	return coreinterpretation.Registration{
		DeclarationType:          LoopType,
		SchemaKey:                LoopSchemaKey,
		SelectorEligible:         true,
		DeclarationAliasEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeLoopEntry(entry)
			return err
		},
		ValidateAdmittedEntry: func(entry declaration.Entry) error {
			_, err := DecodeAdmittedLoopEntry(entry)
			return err
		},
		Relationships: loopRelationships,
	}
}

func loopRelationships(
	entry declaration.Entry,
) ([]coreinterpretation.Relationship, error) {
	document, err := DecodeAdmittedLoopEntry(entry)
	if err != nil {
		return nil, err
	}
	if document.Body == nil {
		return []coreinterpretation.Relationship{}, nil
	}
	value, err := coreinterpretation.NewMemberRelationship(
		[]string{"body"},
		*document.Body,
		coreinterpretation.MemberOptions{},
	)
	if err != nil {
		return nil, err
	}
	return []coreinterpretation.Relationship{value}, nil
}
