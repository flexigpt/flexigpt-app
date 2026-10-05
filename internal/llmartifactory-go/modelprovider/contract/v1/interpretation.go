package modelproviderv1

import (
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() interpretation.Registration {
	return interpretation.Registration{
		DeclarationType: ModelProviderType,
		SchemaKey:       ModelProviderSchemaKey,
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeModelProviderEntry(entry)
			return err
		},
	}
}
