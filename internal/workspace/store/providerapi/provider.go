package providerapi

import "github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"

const providerName = "workspace"

// Provider registers Workspace-owned source adapters and canonical declaration
// schemas. It owns no Collection behavior, Source attachment role, adoption,
// or Artifact lifecycle policy.
type Provider struct {
	descriptor providerapi.Descriptor
}

func NewProvider() (*Provider, error) {
	descriptor := providerapi.Descriptor{
		Name: providerName,
		Decoders: []providerapi.Decoder{
			NewAgentMarkdownDecoder(),
			NewInstructionDecoder(),
			NewContextDecoder(),
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
		string(InstructionMarkdownDecoderID),
		string(ContextMarkdownDecoderID),
	}
}
