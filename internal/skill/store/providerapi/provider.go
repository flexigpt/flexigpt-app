package providerapi

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"

const artifactProviderName = "agent-skill"

// Provider registers the physical SKILL.md source adapter. Canonical Skill
// Plugins are ordinary Plugin declarations handled by artifactcontract/provider.
// There is no proprietary Skill Plugin
// manifest adapter.
type Provider struct {
	descriptor provider.Descriptor
}

func NewProvider() (*Provider, error) {
	markdownDecoder := NewDecoder()
	descriptor := provider.Descriptor{
		Name: artifactProviderName,
		Decoders: []provider.Decoder{
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

func (p *Provider) Descriptor() provider.Descriptor {
	if p == nil {
		return provider.Descriptor{}
	}
	return p.descriptor.Clone()
}
