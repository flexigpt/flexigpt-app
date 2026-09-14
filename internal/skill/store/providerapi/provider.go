package providerapi

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"

const artifactProviderName = "agent-skill"

// Provider registers Skill and Skill Collection source-format adapters. It
// does not register Collection behavior or Store lifecycle policy.
type Provider struct {
	descriptor providerapi.Descriptor
}

func NewProvider() (*Provider, error) {
	markdownDecoder := NewDecoder()
	collectionDecoder := NewCollectionDecoder()

	descriptor := providerapi.Descriptor{
		Name: artifactProviderName,
		Decoders: []providerapi.Decoder{
			markdownDecoder,
			collectionDecoder,
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
