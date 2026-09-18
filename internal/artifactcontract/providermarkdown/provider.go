package providermarkdown

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"

const providerName = "artifact-markdown"

// Provider registers physical Markdown source-format adapters. It emits Text
// and Agent Artifacts for any Store Root.
type Provider struct {
	descriptor providerapi.Descriptor
}

func NewProvider() (*Provider, error) {
	descriptor := providerapi.Descriptor{
		Name: providerName,
		Decoders: []providerapi.Decoder{
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

func (p *Provider) Descriptor() providerapi.Descriptor {
	if p == nil {
		return providerapi.Descriptor{}
	}
	return p.descriptor.Clone()
}

func DefaultDecoderIDs() []string {
	return []string{
		string(AgentMarkdownDecoderID),
		string(TextMarkdownDecoderID),
	}
}
