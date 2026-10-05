package locator

import (
	"fmt"
)

type Registry struct {
	factories []Factory
}

func NewRegistry(
	factories ...Factory,
) (*Registry, error) {
	output := &Registry{
		factories: make([]Factory, 0, len(factories)),
	}

	seen := make(map[FactoryKey]struct{})

	for index, factory := range factories {
		if err := ValidateFactory(factory); err != nil {
			return nil, fmt.Errorf(
				"locator resolver factory %d: %w",
				index,
				err,
			)
		}

		for _, artifactKind := range factory.ArtifactKinds() {
			key := FactoryKey{
				LocatorKind:  factory.LocatorKind(),
				ArtifactKind: artifactKind,
			}
			if _, duplicate := seen[key]; duplicate {
				return nil, fmt.Errorf(
					"duplicate locator resolver %q for Artifact kind %q",
					key.LocatorKind,
					key.ArtifactKind,
				)
			}
			seen[key] = struct{}{}
		}

		output.factories = append(output.factories, factory)
	}

	return output, nil
}

func (r *Registry) Factories() []Factory {
	if r == nil {
		return nil
	}
	return append([]Factory(nil), r.factories...)
}
