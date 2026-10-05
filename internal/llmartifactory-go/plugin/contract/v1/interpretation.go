package pluginv1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() interpretation.Registration {
	return interpretation.Registration{
		DeclarationType:          PluginType,
		SchemaKey:                PluginSchemaKey,
		SelectorEligible:         true,
		DeclarationAliasEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodePluginEntry(entry)
			return err
		},
		Relationships: pluginRelationships,
	}
}

func pluginRelationships(
	entry declaration.Entry,
) ([]interpretation.Relationship, error) {
	document, err := DecodePluginEntry(entry)
	if err != nil {
		return nil, err
	}
	return interpretation.MemberRelationships(
		[]string{"members"},
		document.Members,
		interpretation.MemberOptions{AllowSelector: true},
	)
}
