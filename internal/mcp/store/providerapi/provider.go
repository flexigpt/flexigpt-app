package providerapi

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"

const artifactProviderName = "mcp"

// Provider registers MCP source-format adapters. MCP Collection source
// documents normalize into a canonical collection plus flat MCP Artifacts.
type Provider struct {
	descriptor providerapi.Descriptor
}

func NewProvider() (*Provider, error) {
	decoder := NewDecoder()
	descriptor := providerapi.Descriptor{
		Name: artifactProviderName,
		Decoders: []providerapi.Decoder{
			decoder,
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
