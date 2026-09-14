package providerapi

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
)

// Provider is one Artifact Store inbound format registration.
//
// Providers register schema codecs, source decoders, and optional portable
// locator resolver metadata. They do not own Store lifecycle behavior.
type Provider interface {
	Descriptor() Descriptor
}

type Descriptor struct {
	Name string

	Schemas          []SchemaCodec
	Decoders         []Decoder
	LocatorResolvers []LocatorResolver
}

func (d Descriptor) Clone() Descriptor {
	output := d
	output.Schemas = append([]SchemaCodec(nil), d.Schemas...)
	output.Decoders = append([]Decoder(nil), d.Decoders...)
	output.LocatorResolvers = append(
		[]LocatorResolver(nil),
		d.LocatorResolvers...,
	)
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
	if len(d.Schemas) == 0 &&
		len(d.Decoders) == 0 &&
		len(d.LocatorResolvers) == 0 {
		return fmt.Errorf(
			"%w: artifact provider %q has no registered capabilities",
			basespec.ErrInvalid,
			d.Name,
		)
	}

	seenSchemas := make(map[schema.Key]struct{}, len(d.Schemas))
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
				"%w: Artifact provider %q decoder %d is nil",
				basespec.ErrInvalid,
				d.Name,
				index,
			)
		}
		if err := decoder.ID().Validate(); err != nil {
			return err
		}
		if err := basespec.ValidateRequiredText(
			"Artifact provider decoder revision",
			decoder.Revision(),
			basespec.MaxVersionBytes,
		); err != nil {
			return err
		}
		if _, duplicate := seenDecoders[decoder.ID()]; duplicate {
			return fmt.Errorf(
				"%w: Artifact provider %q repeats decoder %q",
				basespec.ErrConflict,
				d.Name,
				decoder.ID(),
			)
		}
		seenDecoders[decoder.ID()] = struct{}{}
	}

	seenResolvers := make(map[string]struct{}, len(d.LocatorResolvers))
	for _, resolver := range d.LocatorResolvers {
		if err := ValidateLocatorResolver(resolver); err != nil {
			return err
		}
		if _, duplicate := seenResolvers[resolver.LocatorKind()]; duplicate {
			return fmt.Errorf(
				"%w: Artifact provider %q repeats locator resolver %q",
				basespec.ErrConflict,
				d.Name,
				resolver.LocatorKind(),
			)
		}
		seenResolvers[resolver.LocatorKind()] = struct{}{}
	}
	return nil
}
