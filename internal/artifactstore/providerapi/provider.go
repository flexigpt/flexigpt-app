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
