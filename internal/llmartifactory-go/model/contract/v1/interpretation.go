package modelv1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() coreinterpretation.Registration {
	return coreinterpretation.Registration{
		DeclarationType:  ModelType,
		SchemaKey:        ModelSchemaKey,
		SelectorEligible: true,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeModelEntry(entry)
			return err
		},
		ValidateAdmittedEntry: func(entry declaration.Entry) error {
			_, err := DecodeAdmittedModelEntry(entry)
			return err
		},
	}
}
