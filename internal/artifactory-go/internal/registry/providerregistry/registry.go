package providerregistry

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
)

type Registry struct {
	providers        []provider.Descriptor
	schemas          []provider.SchemaCodec
	decoders         []provider.Decoder
	locatorResolvers []provider.LocatorResolverFactory
}

func New(
	providers ...provider.Provider,
) (*Registry, error) {
	output := &Registry{
		providers:        make([]provider.Descriptor, 0, len(providers)),
		schemas:          make([]provider.SchemaCodec, 0),
		decoders:         make([]provider.Decoder, 0),
		locatorResolvers: make([]provider.LocatorResolverFactory, 0),
	}

	seenProviderNames := make(map[string]struct{}, len(providers))
	schemaOwners := make(map[schema.Key]string)
	decoderOwners := make(map[model.DecoderID]string)
	locatorResolverOwners := make(map[provider.LocatorResolverKey]string)

	for index, pro := range providers {
		if pro == nil {
			return nil, fmt.Errorf(
				"%w: artifact provider %d is nil",
				model.ErrInvalid,
				index,
			)
		}

		descriptor := pro.Descriptor().Clone()
		if err := descriptor.Validate(); err != nil {
			return nil, err
		}

		if _, duplicate := seenProviderNames[descriptor.Name]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate artifact provider %q",
				model.ErrConflict,
				descriptor.Name,
			)
		}
		seenProviderNames[descriptor.Name] = struct{}{}

		for _, codec := range descriptor.Schemas {
			key := codec.Key()
			if owner, exists := schemaOwners[key]; exists {
				return nil, fmt.Errorf(
					"%w: schema %q/%q/%q is owned by both providers %q and %q",
					model.ErrConflict,
					key.Kind,
					key.SchemaID,
					key.SchemaVersion,
					owner,
					descriptor.Name,
				)
			}
			schemaOwners[key] = descriptor.Name
		}

		for _, decoder := range descriptor.Decoders {
			id := decoder.ID()
			if owner, exists := decoderOwners[id]; exists {
				return nil, fmt.Errorf(
					"%w: decoder %q is owned by both providers %q and %q",
					model.ErrConflict,
					id,
					owner,
					descriptor.Name,
				)
			}
			decoderOwners[id] = descriptor.Name
		}
		for _, resolver := range descriptor.LocatorResolvers {
			for _, artifactKind := range resolver.ArtifactKinds() {
				key := provider.LocatorResolverKey{
					LocatorKind:  resolver.LocatorKind(),
					ArtifactKind: artifactKind,
				}
				if owner, exists := locatorResolverOwners[key]; exists {
					return nil, fmt.Errorf(
						"%w: locator resolver %q for Artifact kind %q is owned by both providers %q and %q",
						model.ErrConflict,
						key.LocatorKind,
						key.ArtifactKind,
						owner,
						descriptor.Name,
					)
				}
				locatorResolverOwners[key] = descriptor.Name
			}
		}

		output.providers = append(
			output.providers,
			descriptor.Clone(),
		)
		output.schemas = append(
			output.schemas,
			descriptor.Schemas...,
		)
		output.decoders = append(
			output.decoders,
			descriptor.Decoders...,
		)
		output.locatorResolvers = append(
			output.locatorResolvers,
			descriptor.LocatorResolvers...,
		)
	}

	return output, nil
}

func (r *Registry) Providers() []provider.Descriptor {
	if r == nil {
		return nil
	}

	output := make([]provider.Descriptor, len(r.providers))
	for index, descriptor := range r.providers {
		output[index] = descriptor.Clone()
	}
	return output
}

func (r *Registry) Schemas() []provider.SchemaCodec {
	if r == nil {
		return nil
	}
	return append([]provider.SchemaCodec(nil), r.schemas...)
}

func (r *Registry) Decoders() []provider.Decoder {
	if r == nil {
		return nil
	}
	return append([]provider.Decoder(nil), r.decoders...)
}

func (r *Registry) LocatorResolvers() []provider.LocatorResolverFactory {
	if r == nil {
		return nil
	}
	return append(
		[]provider.LocatorResolverFactory(nil),
		r.locatorResolvers...,
	)
}
