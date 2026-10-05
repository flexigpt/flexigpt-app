package workflowv1

import (
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() interpretation.Registration {
	return interpretation.Registration{
		DeclarationType:          WorkflowType,
		SchemaKey:                WorkflowSchemaKey,
		SelectorEligible:         true,
		DeclarationAliasEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeWorkflowEntry(entry)
			return err
		},
		Relationships: workflowRelationships,
	}
}

func workflowRelationships(
	entry declaration.Entry,
) ([]interpretation.Relationship, error) {
	document, err := DecodeWorkflowEntry(entry)
	if err != nil {
		return nil, err
	}

	nodes := append([]Node(nil), document.Nodes...)
	sort.SliceStable(nodes, func(left, right int) bool {
		return nodes[left].ID < nodes[right].ID
	})

	output := make([]interpretation.Relationship, 0, len(nodes))
	for _, node := range nodes {
		segment, err := stableWorkflowNodeSegment(node.ID)
		if err != nil {
			return nil, err
		}
		value, err := interpretation.NewMemberRelationship(
			[]string{
				"nodes",
				segment,
			},
			node.Member,
			interpretation.MemberOptions{},
		)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	return output, nil
}
