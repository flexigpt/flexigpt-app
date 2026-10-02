package source

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"

// ManagedPackagePublication atomically publishes one package at one semantic
// address. ExpectedGeneration is optional for creation and required by callers
// that intentionally replace existing package content.
type ManagedPackagePublication struct {
	Address            ManagedPackageAddress `json:"address"`
	ExpectedGeneration string                `json:"expectedGeneration,omitempty"`
	Files              []ManagedPackageFile  `json:"files"`
}

// NormalizeManagedPackagePublication validates both the semantic address and
// package files and returns fully owned deterministic data.
func NormalizeManagedPackagePublication(
	input ManagedPackagePublication,
) (ManagedPackagePublication, error) {
	if err := input.Address.Validate(); err != nil {
		return ManagedPackagePublication{}, err
	}
	if input.ExpectedGeneration != "" {
		if err := basespec.ValidateSourceGeneration(input.ExpectedGeneration); err != nil {
			return ManagedPackagePublication{}, err
		}
	}

	files, err := NormalizeManagedPackageFiles(input.Files)
	if err != nil {
		return ManagedPackagePublication{}, err
	}

	output := input
	output.Files = files
	return output, nil
}
