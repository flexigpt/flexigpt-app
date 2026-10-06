package workspacev1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() coreinterpretation.Registration {
	return coreinterpretation.Registration{
		DeclarationType:          WorkspaceType,
		SchemaKey:                WorkspaceSchemaKey,
		DeclarationAliasEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeWorkspaceEntry(entry)
			return err
		},
		ValidateAdmittedEntry: func(entry declaration.Entry) error {
			_, err := DecodeAdmittedWorkspaceEntry(entry)
			return err
		},
		Relationships: workspaceRelationships,
	}
}

func workspaceRelationships(
	entry declaration.Entry,
) ([]coreinterpretation.Relationship, error) {
	document, err := DecodeAdmittedWorkspaceEntry(entry)
	if err != nil {
		return nil, err
	}
	return coreinterpretation.MemberRelationships(
		[]string{"members"},
		document.Members,
		coreinterpretation.MemberOptions{AllowSelector: true},
	)
}
