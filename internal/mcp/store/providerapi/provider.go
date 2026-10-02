package providerapi

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"

const artifactProviderName = "mcp"

// Provider registers the standard .mcp.json and mcp.json source-format
// adapter. Canonical MCP Collections are handled by artifactcontract/provider.
type Provider struct {
	descriptor provider.Descriptor
}

func NewProvider() (*Provider, error) {
	decoder := NewDecoder()
	descriptor := provider.Descriptor{
		Name: artifactProviderName,
		Decoders: []provider.Decoder{
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

func (p *Provider) Descriptor() provider.Descriptor {
	if p == nil {
		return provider.Descriptor{}
	}
	return p.descriptor.Clone()
}
