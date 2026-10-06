package agentv1

import (
	"fmt"

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

	output, err := coreinterpretation.MemberRelationships(
		[]string{"members"},
		document.Members,
		coreinterpretation.MemberOptions{AllowSelector: true},
	)
	if err != nil {
		return nil, err
	}
	if document.Loop != nil {
		value, err := coreinterpretation.NewMemberRelationship(
			[]string{"loop"},
			*document.Loop,
			coreinterpretation.MemberOptions{
				DirectPosition: true,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("loop: %w", err)
		}
		output = append(output, value)
	}
	if document.Workflow != nil {
		value, err := coreinterpretation.NewMemberRelationship(
			[]string{"workflow"},
			*document.Workflow,
			coreinterpretation.MemberOptions{
				DirectPosition: true,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("workflow: %w", err)
		}
		output = append(output, value)
	}
	return output, nil
}
