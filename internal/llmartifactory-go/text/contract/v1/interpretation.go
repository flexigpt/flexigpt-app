package textv1

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Interpretation() coreinterpretation.Registration {
	return coreinterpretation.Registration{
		DeclarationType: TextType,
		SchemaKey:       TextSchemaKey,
		LogicalVersion: func(
			entry declaration.Entry,
		) (spec.LogicalVersion, error) {
			document, err := DecodeTextEntry(entry)
			if err != nil {
				return "", err
			}
			return spec.LogicalVersion(document.Insert), nil
		},
		ValidateEntry: func(entry declaration.Entry) error {
			_, err := DecodeTextEntry(entry)
			return err
		},
		ValidateAdmittedEntry: func(entry declaration.Entry) error {
			_, err := DecodeAdmittedTextEntry(entry)
			return err
		},
	}
}
