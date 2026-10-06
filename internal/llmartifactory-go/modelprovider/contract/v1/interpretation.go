package modelproviderv1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() coreinterpretation.Registration {
	return coreinterpretation.Registration{
		DeclarationType: ModelProviderType,
		SchemaKey:       ModelProviderSchemaKey,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeModelProviderEntry(entry)
			return err
		},
		ValidateAdmittedEntry: func(entry declaration.Entry) error {
			_, err := DecodeAdmittedModelProviderEntry(entry)
			return err
		},
	}
}
