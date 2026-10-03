package providercanonical

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/codec"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/locator"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
)

// Registration is the explicit canonical declaration vocabulary registration.
//
// It is intentionally not an artifactory-go generic Provider. The canonical
// declaration language belongs to artifactcontract.
type Registration struct {
	schemaCodecs     []schema.Codec
	decoders         []ingest.Decoder
	locatorFactories []locator.Factory
}

func NewRegistration() (*Registration, error) {
	schemaCodecs, err := schema.NormalizeCodecs(
		codec.AllSchemaCodecs(),
	)
	if err != nil {
		return nil, err
	}

	decoders, err := ingest.NormalizeDecoders([]ingest.Decoder{
		decoder.NewJSONDecoder(),
		decoder.NewYAMLDecoder(),
	})
	if err != nil {
		return nil, err
	}

	locatorRegistry, err := locator.NewRegistry(
		newLocatorpathFactory(),
	)
	if err != nil {
		return nil, err
	}

	return &Registration{
		schemaCodecs:     schemaCodecs,
		decoders:         decoders,
		locatorFactories: locatorRegistry.Factories(),
	}, nil
}

func (r *Registration) SchemaCodecs() []schema.Codec {
	if r == nil {
		return nil
	}
	return append([]schema.Codec(nil), r.schemaCodecs...)
}

func (r *Registration) Decoders() []ingest.Decoder {
	if r == nil {
		return nil
	}
	return append([]ingest.Decoder(nil), r.decoders...)
}

func (r *Registration) LocatorFactories() []locator.Factory {
	if r == nil {
		return nil
	}
	return append(
		[]locator.Factory(nil),
		r.locatorFactories...,
	)
}
