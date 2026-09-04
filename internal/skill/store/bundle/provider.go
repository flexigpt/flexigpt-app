package bundle

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	skillArtifact "github.com/flexigpt/flexigpt-app/internal/skill/store/artifact"
)

const artifactProviderName = "agent-skill"

// Provider registers the Agent Skill artifact family with Artifact Store.
// Bundle lifecycle and runtime projection remain owned by this package.
type Provider struct {
	descriptor providerapi.Descriptor
}

var _ providerapi.Provider = (*Provider)(nil)

func NewProvider() (*Provider, error) {
	decoder, err := skillArtifact.NewDecoder()
	if err != nil {
		return nil, err
	}

	descriptor := providerapi.Descriptor{
		Name: artifactProviderName,
		CollectionBehaviors: []providerapi.CollectionBehavior{
			NewCollectionBehavior(),
		},
		Schemas: []providerapi.SchemaCodec{
			NewShareableCodec(),
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
