package system

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/providerregistry"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

func providerRegistryFromConfig(
	config Config,
) (*providerregistry.Registry, error) {
	registry, err := providerregistry.New(
		config.ArtifactProviders...,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"register Artifact Store providers: %w",
			err,
		)
	}
	return registry, nil
}

// bindProviderSchemas supplies the narrow schema catalog required by modern
// decoders. It validates required schema keys before Artifact Store begins
// discovery, rather than deferring a configuration error until decoding.
func bindProviderSchemas(
	decoders []providerapi.Decoder,
	schemas providerapi.SchemaCatalog,
) error {
	if schemas == nil {
		return fmt.Errorf(
			"%w: provider schema catalog is nil",
			basespec.ErrInvalid,
		)
	}

	available := make(
		map[providerapi.SchemaKey]struct{},
	)
	for _, key := range schemas.Keys() {
		available[key] = struct{}{}
	}

	for _, decoder := range decoders {
		binder, supported := decoder.(providerapi.SchemaCanonicalizerBinder)
		if !supported {
			continue
		}

		required := binder.RequiredSchemaKeys()
		seen := make(
			map[providerapi.SchemaKey]struct{},
			len(required),
		)
		for requiredIndex, key := range required {
			if err := key.Validate(); err != nil {
				return fmt.Errorf(
					"decoder %q required schema %d: %w",
					decoder.ID(),
					requiredIndex,
					err,
				)
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf(
					"%w: decoder %q repeats required schema %q/%q/%q",
					basespec.ErrInvalid,
					decoder.ID(),
					key.Kind,
					key.SchemaID,
					key.SchemaVersion,
				)
			}
			seen[key] = struct{}{}

			if _, found := available[key]; !found {
				return fmt.Errorf(
					"%w: decoder %q requires unregistered schema %q/%q/%q",
					basespec.ErrInvalid,
					decoder.ID(),
					key.Kind,
					key.SchemaID,
					key.SchemaVersion,
				)
			}
		}

		if err := binder.BindExpectedCanonicalizer(schemas); err != nil {
			return fmt.Errorf(
				"bind provider schema catalog to decoder %q: %w",
				decoder.ID(),
				err,
			)
		}
	}

	return nil
}
