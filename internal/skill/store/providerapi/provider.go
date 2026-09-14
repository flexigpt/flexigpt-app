package providerapi

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"

const artifactProviderName = "agent-skill"

// Provider registers source format adapters and the standalone canonical Skill
// schema. It does not register Collection behavior.
type Provider struct {
	descriptor providerapi.Descriptor
}

func NewProvider() (*Provider, error) {
	markdownDecoder := NewDecoder()
	descriptor := providerapi.Descriptor{
		Name: artifactProviderName,
		Decoders: []providerapi.Decoder{
			markdownDecoder,
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
