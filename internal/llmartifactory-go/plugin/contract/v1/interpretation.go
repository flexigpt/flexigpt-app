package pluginv1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() coreinterpretation.Registration {
	return coreinterpretation.Registration{
		DeclarationType:          PluginType,
		SchemaKey:                PluginSchemaKey,
		SelectorEligible:         true,
		DeclarationAliasEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodePluginEntry(entry)
			return err
		},
		ValidateAdmittedEntry: func(entry declaration.Entry) error {
			_, err := DecodeAdmittedPluginEntry(entry)
			return err
		},
		Relationships: pluginRelationships,
	}
}

func pluginRelationships(
	entry declaration.Entry,
) ([]coreinterpretation.Relationship, error) {
	document, err := DecodeAdmittedPluginEntry(entry)
	if err != nil {
		return nil, err
	}
	return coreinterpretation.MemberRelationships(
		[]string{"members"},
		document.Members,
		coreinterpretation.MemberOptions{AllowSelector: true},
	)
}
