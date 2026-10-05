package workspacev1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() interpretation.Registration {
	return interpretation.Registration{
		DeclarationType:          WorkspaceType,
		SchemaKey:                WorkspaceSchemaKey,
		DeclarationAliasEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeWorkspaceEntry(entry)
			return err
		},
		Relationships: workspaceRelationships,
	}
}

func workspaceRelationships(
	entry declaration.Entry,
) ([]interpretation.Relationship, error) {
	document, err := DecodeWorkspaceEntry(entry)
	if err != nil {
		return nil, err
	}
	return interpretation.MemberRelationships(
		[]string{"members"},
		document.Members,
		interpretation.MemberOptions{AllowSelector: true},
	)
}
