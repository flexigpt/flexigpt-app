package loopv1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() interpretation.Registration {
	return interpretation.Registration{
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
) ([]interpretation.Relationship, error) {
	document, err := DecodeAdmittedLoopEntry(entry)
	if err != nil {
		return nil, err
	}
	if document.Body == nil {
		return []interpretation.Relationship{}, nil
	}
	value, err := interpretation.NewMemberRelationship(
		[]string{"body"},
		*document.Body,
		interpretation.MemberOptions{},
	)
	if err != nil {
		return nil, err
	}
	return []interpretation.Relationship{value}, nil
}
