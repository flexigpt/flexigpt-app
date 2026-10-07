package workspace

import (
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
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

type Support struct {
	DirectorySource         support.SourceProfile
	PolicySource            support.SourceProfile
	RootStorageKeyPrefix    spec.StorageKey
	DirectoryDiscovery      sourceModel.DiscoverySpec
	ManifestPatterns        []string
	SelectorIncludePatterns []string
	SkillDocuments          support.Documents
}

func (s Support) Validate() error {
	if err := s.DirectorySource.Validate(); err != nil {
		return err
	}
	if err := s.PolicySource.Validate(); err != nil {
		return err
	}
	if err := s.DirectoryDiscovery.Validate(); err != nil {
		return err
	}
	if err := spec.ValidatePathPatterns(
		"Workspace manifest patterns",
		s.ManifestPatterns,
	); err != nil {
		return err
	}
	if err := spec.ValidatePathPatterns(
		"Workspace selector patterns",
		s.SelectorIncludePatterns,
	); err != nil {
		return err
	}
	return s.SkillDocuments.Validate()
}

func (s Support) Clone() Support {
	output := s
	output.DirectorySource = s.DirectorySource.Clone()
	output.PolicySource = s.PolicySource.Clone()
	output.DirectoryDiscovery = s.DirectoryDiscovery.Clone()
	output.ManifestPatterns = append([]string(nil), s.ManifestPatterns...)
	output.SelectorIncludePatterns = append(
		[]string(nil),
		s.SelectorIncludePatterns...,
	)
	output.SkillDocuments = s.SkillDocuments.Clone()
	return output
}

type Config struct {
	ContextComposition  contextengine.CompositionPolicy
	Composition         *composition.Resolver
	MCPServers          workspacemcp.ServerResolver
	DefaultPolicySource DefaultPolicySource
	Support             Support

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
	output.Support = c.Support.Clone()
	return output
}

func DefaultConfig() Config {
	return Config{
		ContextComposition: contextengine.DefaultCompositionPolicy(),
	}
}
