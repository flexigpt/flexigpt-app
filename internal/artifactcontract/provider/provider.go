package provider

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/codec"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

const providerName = "artifact-declaration"

// Provider registers the complete canonical declaration vocabulary exactly
// once. Physical source-format providers remain separate.
type Provider struct {
	descriptor providerapi.Descriptor
}

func New() (*Provider, error) {
	descriptor := providerapi.Descriptor{
		Name:    providerName,
		Schemas: codec.AllSchemaCodecs(),
		Decoders: []providerapi.Decoder{
			decoder.NewJSONDecoder(),
			decoder.NewYAMLDecoder(),
		},
		LocatorResolvers: []providerapi.LocatorResolverFactory{
			newLocatorpathFactory(),
		},
	}
	if err := descriptor.Validate(); err != nil {
		return nil, err
	}
	return &Provider{
		descriptor: descriptor.Clone(),
	}, nil
}

func (p *Provider) Descriptor() providerapi.Descriptor {
	if p == nil {
		return providerapi.Descriptor{}
	}
	return p.descriptor.Clone()
}
