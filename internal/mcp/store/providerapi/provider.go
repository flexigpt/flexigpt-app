package providerapi

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"

const artifactProviderName = "mcp"

// Provider registers standalone MCP and MCP Policy schemas plus source format
// adapters. It has no Collection behavior and emits only flat Artifacts.
type Provider struct {
	descriptor providerapi.Descriptor
}

func NewProvider() (*Provider, error) {
	decoder := NewDecoder()
	descriptor := providerapi.Descriptor{
		Name: artifactProviderName,
		Schemas: []providerapi.SchemaCodec{
			NewMCPCodec(),
			NewPolicyCodec(),
		},
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
