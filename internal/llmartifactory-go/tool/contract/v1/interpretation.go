package toolv1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() interpretation.Registration {
	return interpretation.Registration{
		DeclarationType:  ToolType,
		SchemaKey:        ToolSchemaKey,
		SelectorEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeToolEntry(entry)
			return err
		},
	}
}
