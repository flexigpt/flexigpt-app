package providerapi

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// Provider is one Artifact Store inbound artifact-family registration.
//
// It registers schemas, decoders, and collection-kind behavior without
// requiring Artifact Store to import the concrete provider package.
type Provider interface {
	Descriptor() Descriptor
}

// Descriptor is immutable provider registration metadata.
//
// CollectionKinds establishes exclusive ownership of persisted collection
// kinds through executable CollectionBehavior implementations.
type Descriptor struct {
	Name string

	CollectionBehaviors []CollectionBehavior
	Schemas             []SchemaCodec
	Decoders            []Decoder
}

func (d Descriptor) Clone() Descriptor {
	output := d
	output.CollectionBehaviors = append(
		[]CollectionBehavior(nil),
		d.CollectionBehaviors...,
	)
	output.Schemas = append([]SchemaCodec(nil), d.Schemas...)
	output.Decoders = append([]Decoder(nil), d.Decoders...)
	return output
}

func (d Descriptor) Validate() error {
	if err := basespec.ValidateIdentifier(
		"artifact provider name",
		d.Name,
		basespec.MaxKindBytes,
	); err != nil {
		return err
	}

	if len(d.CollectionBehaviors) == 0 &&
		len(d.Schemas) == 0 &&
		len(d.Decoders) == 0 {
		return fmt.Errorf(
			"%w: artifact provider %q has no registered capabilities",
			basespec.ErrInvalid,
			d.Name,
		)
	}

	seenCollectionBehaviors := make(
		map[basespec.CollectionKind]struct{},
		len(d.CollectionBehaviors),
	)
	for index, behavior := range d.CollectionBehaviors {
		if err := ValidateCollectionBehavior(behavior); err != nil {
			return fmt.Errorf(
				"artifact provider %q collection behavior %d: %w",
				d.Name,
				index,
				err,
			)
		}
		kind := behavior.CollectionKind()
		if _, duplicate := seenCollectionBehaviors[kind]; duplicate {
			return fmt.Errorf(
				"%w: artifact provider %q repeats collection behavior %q",
				basespec.ErrConflict,
				d.Name,
				kind,
			)
		}
		seenCollectionBehaviors[kind] = struct{}{}
	}

	seenSchemas := make(map[SchemaKey]struct{}, len(d.Schemas))
	for index, codec := range d.Schemas {
		if codec == nil {
			return fmt.Errorf(
				"%w: artifact provider %q schema codec %d is nil",
				basespec.ErrInvalid,
				d.Name,
				index,
			)
		}
		key := codec.Key()
		if err := key.Validate(); err != nil {
			return fmt.Errorf(
				"artifact provider %q schema codec %d: %w",
				d.Name,
				index,
				err,
			)
		}
		if _, duplicate := seenSchemas[key]; duplicate {
			return fmt.Errorf(
				"%w: artifact provider %q repeats schema %q/%q/%q",
				basespec.ErrConflict,
				d.Name,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			)
		}
		seenSchemas[key] = struct{}{}
	}

	seenDecoders := make(
		map[basespec.DecoderID]struct{},
		len(d.Decoders),
	)
	for index, decoder := range d.Decoders {
		if decoder == nil {
			return fmt.Errorf(
				"%w: artifact provider %q decoder %d is nil",
				basespec.ErrInvalid,
				d.Name,
				index,
			)
		}

		id := decoder.ID()
		if err := basespec.ValidateDecoderID(id); err != nil {
			return fmt.Errorf(
				"artifact provider %q decoder %d: %w",
				d.Name,
				index,
				err,
			)
		}
		if err := basespec.ValidateRequiredText(
			"artifact provider decoder revision",
			decoder.Revision(),
			basespec.MaxVersionBytes,
		); err != nil {
			return fmt.Errorf(
				"artifact provider %q decoder %q: %w",
				d.Name,
				id,
				err,
			)
		}
		if _, duplicate := seenDecoders[id]; duplicate {
			return fmt.Errorf(
				"%w: artifact provider %q repeats decoder %q",
				basespec.ErrConflict,
				d.Name,
				id,
			)
		}
		seenDecoders[id] = struct{}{}
	}

	return nil
}

// Registry is an immutable Artifact Store provider registry.
//
// It validates global uniqueness for provider names, collection kinds, schema
// keys, and decoder IDs before Artifact Store opens metadata or starts source
// discovery.
type Registry struct {
	providers []Descriptor
	schemas   []SchemaCodec
	decoders  []Decoder

	byCollectionKind    map[basespec.CollectionKind]Descriptor
	collectionBehaviors map[basespec.CollectionKind]CollectionBehavior
}

func NewRegistry(
	providers ...Provider,
) (*Registry, error) {
	output := &Registry{
		providers: make([]Descriptor, 0, len(providers)),
		schemas:   make([]SchemaCodec, 0),
		decoders:  make([]Decoder, 0),

		byCollectionKind: make(
			map[basespec.CollectionKind]Descriptor,
		),
		collectionBehaviors: make(map[basespec.CollectionKind]CollectionBehavior),
	}

	seenProviderNames := make(map[string]struct{}, len(providers))
	schemaOwners := make(map[SchemaKey]string)
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

func (r *Registry) Providers() []Descriptor {
	if r == nil {
		return nil
	}

	output := make([]Descriptor, len(r.providers))
	for index, descriptor := range r.providers {
		output[index] = descriptor.Clone()
	}
	return output
}

func (r *Registry) Schemas() []SchemaCodec {
	if r == nil {
		return nil
	}
	return append([]SchemaCodec(nil), r.schemas...)
}

func (r *Registry) Decoders() []Decoder {
	if r == nil {
		return nil
	}
	return append([]Decoder(nil), r.decoders...)
}

func (r *Registry) ProviderForCollectionKind(
	kind basespec.CollectionKind,
) (Descriptor, bool) {
	if r == nil {
		return Descriptor{}, false
	}

	value, found := r.byCollectionKind[kind]
	if !found {
		return Descriptor{}, false
	}
	return value.Clone(), true
}

func (r *Registry) CollectionBehavior(
	kind basespec.CollectionKind,
) (CollectionBehavior, bool) {
	if r == nil {
		return nil, false
	}

	behavior, found := r.collectionBehaviors[kind]
	if !found {
		return nil, false
	}
	return behavior, true
}
