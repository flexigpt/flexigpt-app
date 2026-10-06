package workspace

import (
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	workspacemcp "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/mcp"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/contextengine"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
)

type DefaultPolicySource struct {
	ProviderKey string
	Root        spec.Locator
	Locator     spec.Locator
	DecoderID   spec.DecoderID
	Policy      workspaceDomain.DefaultPolicy
}

func (s DefaultPolicySource) Validate() error {
	if err := spec.ValidateIdentifier(
		"Workspace default policy provider key",
		s.ProviderKey,
		spec.MaxKindBytes,
	); err != nil {
		return err
	}
	if err := s.Root.ValidatePortable(false); err != nil {
		return err
	}
	if err := s.Locator.ValidatePortable(false); err != nil {
		return err
	}
	if err := s.DecoderID.Validate(); err != nil {
		return err
	}
	return s.Policy.Validate()
}

type Config struct {
	ContextComposition  contextengine.CompositionPolicy
	Composition         *composition.Resolver
	MCPServers          workspacemcp.ServerResolver
	DefaultPolicySource DefaultPolicySource

	// AdditionalDecoderHints lets application composition add dedicated
	// decoders without making Workspace import unrelated family services.
	AdditionalDecoderHints []sourceModel.DecoderHint
}

func (c Config) normalized() Config {
	output := c
	output.ContextComposition = output.ContextComposition.Normalized()
	output.AdditionalDecoderHints = make(
		[]sourceModel.DecoderHint,
		len(c.AdditionalDecoderHints),
	)
	for index, hint := range c.AdditionalDecoderHints {
		output.AdditionalDecoderHints[index] = hint.Clone()
	}
	output.DefaultPolicySource = c.DefaultPolicySource
	return output
}

func DefaultConfig() Config {
	return Config{
		ContextComposition: contextengine.DefaultCompositionPolicy(),
	}
}
