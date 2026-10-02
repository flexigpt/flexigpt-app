package providermarkdown

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"

const providerName = "artifact-markdown"

// Provider registers physical Markdown source-format adapters. It emits Text
// and Agent Artifacts for any Store Root.
type Provider struct {
	descriptor provider.Descriptor
}

func NewProvider() (*Provider, error) {
	descriptor := provider.Descriptor{
		Name: providerName,
		Decoders: []provider.Decoder{
			NewAgentMarkdownDecoder(),
			NewTextDecoder(),
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

func DefaultDecoderIDs() []string {
	return []string{
		string(AgentMarkdownDecoderID),
		string(TextMarkdownDecoderID),
	}
}
