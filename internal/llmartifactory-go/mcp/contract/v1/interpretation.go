package mcpv1

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() interpretation.Registration {
	return interpretation.Registration{
		DeclarationType:          MCPType,
		SchemaKey:                MCPSchemaKey,
		SelectorEligible:         true,
		DeclarationAliasEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeMCPEntry(entry)
			return err
		},
		ValidateAdmittedEntry: func(entry declaration.Entry) error {
			_, err := DecodeAdmittedMCPEntry(entry)
			return err
		},
		Relationships: mcpRelationships,
	}
}

func mcpRelationships(
	entry declaration.Entry,
) ([]interpretation.Relationship, error) {
	document, err := DecodeAdmittedMCPEntry(entry)
	if err != nil {
		return nil, err
	}
	if document.Policy == nil {
		return []interpretation.Relationship{}, nil
	}

	member, err := declaration.NewSymbolicEntry(
		declaration.Type("mcp.policy"),
		document.Policy.Name,
	)
	if err != nil {
		return nil, err
	}

	optional := document.Policy.Required != nil &&
		!*document.Policy.Required
	value, err := interpretation.NewMemberRelationship(
		[]string{"policy"},
		member,
		interpretation.MemberOptions{Optional: optional},
	)
	if err != nil {
		return nil, err
	}
	if value.Declared.Header().Name == "" {
		return nil, spec.ErrInvalid
	}
	return []interpretation.Relationship{value}, nil
}
