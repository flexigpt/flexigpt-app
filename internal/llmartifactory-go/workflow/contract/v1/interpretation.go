package workflowv1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() coreinterpretation.Registration {
	return coreinterpretation.Registration{
		DeclarationType:  WorkflowType,
		SchemaKey:        WorkflowSchemaKey,
		SelectorEligible: true,

		// Workflow documents are concrete declarations. Target references can
		// still use registered portable locator handlers.
		DeclarationAliasEligible: false,
		RequireAPIVersion:        WorkflowSourceVersionRequired,

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

func workflowRelationships(entry declaration.Entry) ([]coreinterpretation.Relationship, error) {
	document, err := DecodeAdmittedWorkflowEntry(entry)
	if err != nil {
		return nil, err
	}
	output := make([]coreinterpretation.Relationship, 0, len(document.Targets)+len(document.Requirements))
	for _, group := range []struct {
		path   string
		values map[string]Target
	}{
		{path: "targets", values: document.Targets},
		{path: "requirements", values: document.Requirements},
	} {
		for _, name := range sortedKeys(group.values) {
			member, artifactBacked, err := group.values[name].DeclarationMember()
			if err != nil {
				return nil, err
			}
			if !artifactBacked {
				continue
			}
			form, err := member.MemberForm()
			if err != nil {
				return nil, err
			}
			output = append(output, coreinterpretation.Relationship{
				Path:     []string{group.path, name},
				Declared: member,
				Form:     form,
				Required: true,
			})
		}
	}
	return output, nil
}
