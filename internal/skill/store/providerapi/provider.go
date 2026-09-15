package providerapi

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"

const artifactProviderName = "agent-skill"

// Provider registers the physical SKILL.md source adapter. Canonical Skill
// collections are ordinary collection declarations handled by
// artifactcontract/provider. There is no proprietary Skill collection
// manifest adapter.
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
