package toolv1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() coreinterpretation.Registration {
	return coreinterpretation.Registration{
		DeclarationType:  ToolType,
		SchemaKey:        ToolSchemaKey,
		SelectorEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeToolEntry(entry)
			return err
		},
		ValidateAdmittedEntry: func(entry declaration.Entry) error {
			_, err := DecodeAdmittedToolEntry(entry)
			return err
		},
	}
}
