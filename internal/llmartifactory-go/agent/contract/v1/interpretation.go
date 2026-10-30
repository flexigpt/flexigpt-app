package agentv1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() coreinterpretation.Registration {
	return coreinterpretation.Registration{
		DeclarationType:          AgentType,
		SchemaKey:                AgentSchemaKey,
		SelectorEligible:         true,
		DeclarationAliasEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeAgentEntry(entry)
			return err
		},
		ValidateAdmittedEntry: func(entry declaration.Entry) error {
			_, err := DecodeAdmittedAgentEntry(entry)
			return err
		},
		Relationships: agentRelationships,
	}
}

func agentRelationships(
	entry declaration.Entry,
) ([]coreinterpretation.Relationship, error) {
	document, err := DecodeAdmittedAgentEntry(entry)
	if err != nil {
		return nil, err
	}
	return coreinterpretation.MemberRelationships(
		[]string{"members"},
		document.Members,
		coreinterpretation.MemberOptions{AllowSelector: true},
	)
}
