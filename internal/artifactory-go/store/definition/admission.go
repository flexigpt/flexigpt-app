package definition

import definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"

// Admit is Definition's single canonicalization and digest-admission boundary
// for decoded or trusted generated Definition values. Callers receive an owned
// canonical value suitable for source observations and immutable publication.
func Admit(value definitionModel.Definition) (definitionModel.Definition, error) {
	admitted, err := definitionModel.Canonicalize(value)
	if err != nil {
		return definitionModel.Definition{}, err
	}
	return admitted.Clone(), nil
}
