package providerregistry

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

// Registry is Artifact Store's immutable provider capability registry.
//
// It owns global uniqueness enforcement for provider names, collection kinds,
// schema keys, and decoder IDs. Providers receive no access to this registry.
type Registry struct {
	providers []providerapi.Descriptor
	schemas   []providerapi.SchemaCodec
	decoders  []providerapi.Decoder

	byCollectionKind    map[basespec.CollectionKind]providerapi.Descriptor
	collectionBehaviors map[basespec.CollectionKind]providerapi.CollectionBehavior
}

func New(
	providers ...providerapi.Provider,
) (*Registry, error) {
	output := &Registry{
		providers: make([]providerapi.Descriptor, 0, len(providers)),
		schemas:   make([]providerapi.SchemaCodec, 0),
		decoders:  make([]providerapi.Decoder, 0),
		byCollectionKind: make(
			map[basespec.CollectionKind]providerapi.Descriptor,
		),
		collectionBehaviors: make(
			map[basespec.CollectionKind]providerapi.CollectionBehavior,
		),
	}

	seenProviderNames := make(map[string]struct{}, len(providers))
	schemaOwners := make(map[providerapi.SchemaKey]string)
	decoderOwners := make(map[basespec.DecoderID]string)

	for index, provider := range providers {
		if provider == nil {
			return nil, fmt.Errorf(
				"%w: artifact provider %d is nil",
				basespec.ErrInvalid,
				index,
			)
		}

		descriptor := provider.Descriptor().Clone()
		if err := descriptor.Validate(); err != nil {
			return nil, err
		}

		if _, duplicate := seenProviderNames[descriptor.Name]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate artifact provider %q",
				basespec.ErrConflict,
				descriptor.Name,
			)
		}
		seenProviderNames[descriptor.Name] = struct{}{}

		for _, behavior := range descriptor.CollectionBehaviors {
			kind := behavior.CollectionKind()
			if owner, exists := output.byCollectionKind[kind]; exists {
				return nil, fmt.Errorf(
					"%w: collection kind %q is owned by both providers %q and %q",
					basespec.ErrConflict,
					kind,
					owner.Name,
					descriptor.Name,
				)
			}
			output.byCollectionKind[kind] = descriptor.Clone()
			output.collectionBehaviors[kind] = behavior
		}

		for _, codec := range descriptor.Schemas {
			key := codec.Key()
			if owner, exists := schemaOwners[key]; exists {
				return nil, fmt.Errorf(
					"%w: schema %q/%q/%q is owned by both providers %q and %q",
					basespec.ErrConflict,
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
					basespec.ErrConflict,
					id,
					owner,
					descriptor.Name,
				)
			}
			decoderOwners[id] = descriptor.Name
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
	}

	return output, nil
}

func (r *Registry) Providers() []providerapi.Descriptor {
	if r == nil {
		return nil
	}

	output := make([]providerapi.Descriptor, len(r.providers))
	for index, descriptor := range r.providers {
		output[index] = descriptor.Clone()
	}
	return output
}

func (r *Registry) Schemas() []providerapi.SchemaCodec {
	if r == nil {
		return nil
	}
	return append([]providerapi.SchemaCodec(nil), r.schemas...)
}

func (r *Registry) Decoders() []providerapi.Decoder {
	if r == nil {
		return nil
	}
	return append([]providerapi.Decoder(nil), r.decoders...)
}

func (r *Registry) ProviderForCollectionKind(
	kind basespec.CollectionKind,
) (providerapi.Descriptor, bool) {
	if r == nil {
		return providerapi.Descriptor{}, false
	}

	value, found := r.byCollectionKind[kind]
	if !found {
		return providerapi.Descriptor{}, false
	}
	return value.Clone(), true
}

func (r *Registry) CollectionBehavior(
	kind basespec.CollectionKind,
) (providerapi.CollectionBehavior, bool) {
	if r == nil {
		return nil, false
	}

	behavior, found := r.collectionBehaviors[kind]
	if !found {
		return nil, false
	}
	return behavior, true
}
