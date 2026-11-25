package modelcatalog

import (
	"context"
	"maps"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	modelAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model"
)

// AdapterDefinition declares one catalog-owned inference adapter.
type AdapterDefinition struct {
	ID      string
	Version string

	SDKType         inferenceSpec.ProviderSDKType
	Origin          string
	Path            string
	APIKeyHeaderKey string
	DefaultHeaders  map[string]string
}

// AdapterRegistry owns the builtin adapter declarations used by Model Store.
// Callers receive independent copies when they need execution definitions.
type AdapterRegistry struct {
	definitions []AdapterDefinition
}

func DefaultAdapterRegistry() AdapterRegistry {
	return AdapterRegistry{
		definitions: defaultAdapterDefinitions(),
	}
}

func (r AdapterRegistry) LookupModelAdapter(
	_ context.Context,
	adapter string,
) (modelAPI.AdapterDescriptor, bool, error) {
	for _, definition := range r.definitions {
		if definition.ID == adapter {
			return modelAPI.AdapterDescriptor{
				ID:      definition.ID,
				Version: definition.Version,
			}, true, nil
		}
	}
	return modelAPI.AdapterDescriptor{}, false, nil
}

// Definitions returns independent copies of the registry declarations.
func (r AdapterRegistry) Definitions() []AdapterDefinition {
	output := make([]AdapterDefinition, len(r.definitions))
	for index, definition := range r.definitions {
		definition.DefaultHeaders = maps.Clone(definition.DefaultHeaders)
		output[index] = definition
	}
	return output
}
