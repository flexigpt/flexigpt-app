package workspace

import (
	"fmt"

	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	workspacemcp "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/adapter/mcp"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/contextengine"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
)

type DefaultPolicySource struct {
	ProviderKey string
	Root        storeSpec.Locator
	Locator     storeSpec.Locator
	DecoderID   storeSpec.DecoderID
	Policy      workspaceDomain.DefaultPolicy
}

func (s DefaultPolicySource) Validate() error {
	if err := storeSpec.ValidateIdentifier(
		"Workspace default policy provider key",
		s.ProviderKey,
		storeSpec.MaxKindBytes,
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
	DirectorySource      support.SourceProfile
	PolicySource         support.SourceProfile
	RootStorageKeyPrefix storeSpec.StorageKey
	DirectoryDiscovery   sourceModel.DiscoverySpec
	ManifestPatterns     []string
	SelectorPatterns     map[declaration.Type][]string
	SkillDocuments       support.Documents
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
	if err := storeSpec.ValidatePathPatterns(
		"Workspace manifest patterns",
		s.ManifestPatterns,
	); err != nil {
		return err
	}
	if len(s.SelectorPatterns) == 0 {
		return fmt.Errorf(
			"%w: Workspace selector discovery patterns are required",
			storeSpec.ErrInvalid,
		)
	}
	for declarationType, patterns := range s.SelectorPatterns {
		if err := storeSpec.ValidateRequiredText(
			"Workspace selector declaration type",
			string(declarationType),
			storeSpec.MaxKindBytes,
		); err != nil {
			return err
		}
		if len(patterns) == 0 {
			return fmt.Errorf(
				"%w: Workspace selector type %q has no discovery patterns",
				storeSpec.ErrInvalid,
				declarationType,
			)
		}
		if err := storeSpec.ValidatePathPatterns(
			"Workspace selector discovery patterns",
			patterns,
		); err != nil {
			return err
		}
	}
	if err := s.SkillDocuments.Validate(); err != nil {
		return err
	}
	return nil
}

func (s Support) Clone() Support {
	output := s
	output.DirectorySource = s.DirectorySource.Clone()
	output.PolicySource = s.PolicySource.Clone()
	output.DirectoryDiscovery = s.DirectoryDiscovery.Clone()
	output.ManifestPatterns = append([]string(nil), s.ManifestPatterns...)
	output.SelectorPatterns = make(
		map[declaration.Type][]string,
		len(s.SelectorPatterns),
	)
	for declarationType, patterns := range s.SelectorPatterns {
		output.SelectorPatterns[declarationType] = append(
			[]string(nil),
			patterns...,
		)
	}
	output.SkillDocuments = s.SkillDocuments.Clone()
	return output
}

func (s Support) selectorPatternsFor(
	declarationType declaration.Type,
) ([]string, bool) {
	patterns, found := s.SelectorPatterns[declarationType]
	if !found {
		return nil, false
	}
	return append([]string(nil), patterns...), true
}

func (s Support) directoryExcludePatterns() []string {
	seen := make(map[string]struct{})
	output := make([]string, 0)
	for _, directory := range s.DirectoryDiscovery.DirectoryRoots {
		for _, pattern := range directory.ExcludePatterns {
			if _, duplicate := seen[pattern]; duplicate {
				continue
			}
			seen[pattern] = struct{}{}
			output = append(output, pattern)
		}
	}
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
