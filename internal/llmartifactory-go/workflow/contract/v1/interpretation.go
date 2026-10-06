package workflowv1

import (
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() coreinterpretation.Registration {
	return coreinterpretation.Registration{
		DeclarationType:          WorkflowType,
		SchemaKey:                WorkflowSchemaKey,
		SelectorEligible:         true,
		DeclarationAliasEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeWorkflowEntry(entry)
			return err
		},
		ValidateAdmittedEntry: func(entry declaration.Entry) error {
			_, err := DecodeAdmittedWorkflowEntry(entry)
			return err
		},
		Relationships: workflowRelationships,
	}
}

func workflowRelationships(
	entry declaration.Entry,
) ([]coreinterpretation.Relationship, error) {
	document, err := DecodeAdmittedWorkflowEntry(entry)
	if err != nil {
		return nil, err
	}

	nodes := append([]Node(nil), document.Nodes...)
	sort.SliceStable(nodes, func(left, right int) bool {
		return nodes[left].ID < nodes[right].ID
	})

	output := make([]coreinterpretation.Relationship, 0, len(nodes))
	for _, node := range nodes {
		segment, err := NodeRelationshipSegment(node.ID)
		if err != nil {
			return nil, err
		}
		value, err := coreinterpretation.NewMemberRelationship(
			[]string{
				"nodes",
				segment,
			},
			node.Member,
			coreinterpretation.MemberOptions{},
		)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	return output, nil
}
