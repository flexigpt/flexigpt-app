package providercanonical

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/codec"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
)

const providerName = "artifact-declaration"

// Provider registers the complete canonical declaration vocabulary exactly
// once. Physical source-format providers remain separate.
type Provider struct {
	descriptor provider.Descriptor
}

func New() (*Provider, error) {
	descriptor := provider.Descriptor{
		Name:    providerName,
		Schemas: codec.AllSchemaCodecs(),
		Decoders: []provider.Decoder{
			decoder.NewJSONDecoder(),
			decoder.NewYAMLDecoder(),
		},
		LocatorResolvers: []provider.LocatorResolverFactory{
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

func (p *Provider) Descriptor() provider.Descriptor {
	if p == nil {
		return provider.Descriptor{}
	}
	return p.descriptor.Clone()
}
