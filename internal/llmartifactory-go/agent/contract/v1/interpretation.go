package agentv1

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() interpretation.Registration {
	return interpretation.Registration{
		DeclarationType:          AgentType,
		SchemaKey:                AgentSchemaKey,
		SelectorEligible:         true,
		DeclarationAliasEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeAgentEntry(entry)
			return err
		},
		Relationships: agentRelationships,
	}
}

func agentRelationships(
	entry declaration.Entry,
) ([]interpretation.Relationship, error) {
	document, err := DecodeAgentEntry(entry)
	if err != nil {
		return nil, err
	}

	output, err := interpretation.MemberRelationships(
		[]string{"members"},
		document.Members,
		interpretation.MemberOptions{AllowSelector: true},
	)
	if err != nil {
		return nil, err
	}
	if document.Loop != nil {
		value, err := interpretation.NewMemberRelationship(
			[]string{"loop"},
			*document.Loop,
			interpretation.MemberOptions{
				DirectPosition: true,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("loop: %w", err)
		}
		output = append(output, value)
	}
	if document.Workflow != nil {
		value, err := interpretation.NewMemberRelationship(
			[]string{"workflow"},
			*document.Workflow,
			interpretation.MemberOptions{
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
