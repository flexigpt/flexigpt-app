package workspace

import (
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

const (
	providerName                   = "workspace"
	defaultDiscoveryPolicyRevision = "workspace.discovery.v1"
)

// ProviderConfig contains Workspace's inbound Artifact Store behavior
// configuration. It is supplied while Artifact Store providers are registered,
// before metadata is opened or a Workspace is refreshed.
type ProviderConfig struct {
	WorkspaceRootID         basespec.RootID
	Supports                []spec.ArtifactSupport
	DiscoveryProfiles       spec.DiscoveryProfiles
	DiscoveryPolicyRevision string
	SkillRoots              []basespec.Locator
}

// DefaultProviderConfig returns the application Workspace provider behavior.
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		WorkspaceRootID:         artifactbuiltin.WorkspaceRootID,
		Supports:                DefaultArtifactSupports(),
		DiscoveryPolicyRevision: defaultDiscoveryPolicyRevision,
		SkillRoots:              artifactbuiltin.WorkspaceSkillRoots(),
	}
}

// ProviderConfig returns the provider configuration that shares this
// Workspace API configuration's root and supported Artifact matrix. Provider
// planning settings remain explicit provider-composition settings.
func (c Config) ProviderConfig() ProviderConfig {
	c = c.normalized()
	return ProviderConfig{
		WorkspaceRootID: c.WorkspaceRootID,
		Supports:        append([]spec.ArtifactSupport(nil), c.Supports...),
	}
}

type workspaceProviderConfiguration struct {
	workspaceRootID basespec.RootID
	supports        []spec.ArtifactSupport
	decoderIDs      []basespec.DecoderID
	profiles        spec.DiscoveryProfiles
	revision        string
}

// Provider registers Workspace's schema codec, context decoder, and
// collection-kind behavior as one Artifact Store plugin.
type Provider struct {
	descriptor providerapi.Descriptor
}

func NewProvider(config ProviderConfig) (*Provider, error) {
	behavior, err := newWorkspaceCollectionBehavior(config)
	if err != nil {
		return nil, err
	}

	descriptor := providerapi.Descriptor{
		Name: providerName,
		CollectionBehaviors: []providerapi.CollectionBehavior{
			behavior,
		},
		Schemas: []providerapi.SchemaCodec{
			NewCollectionCodec(),
		},
		Decoders: DefaultDecoders(),
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

// DefaultDecoders returns only the decoders owned by Workspace. Shared
// artifact-family decoders are registered by their owning providers.
func DefaultDecoders() []providerapi.Decoder {
	return []providerapi.Decoder{
		contextadapter.NewContextDecoder(),
	}
}

func normalizeProviderConfig(
	input ProviderConfig,
) (workspaceProviderConfiguration, error) {
	if err := basespec.ValidateRootID(input.WorkspaceRootID); err != nil {
		return workspaceProviderConfiguration{}, err
	}

	supports := append([]spec.ArtifactSupport(nil), input.Supports...)
	if len(supports) == 0 {
		supports = DefaultArtifactSupports()
	}
	if len(supports) == 0 {
		return workspaceProviderConfiguration{}, fmt.Errorf(
			"%w: Workspace provider requires at least one Artifact support",
			spec.ErrInvalidWorkspace,
		)
	}

	seenKinds := make(
		map[basespec.ArtifactKind]struct{},
		len(supports),
	)
	seenDecoders := make(
		map[basespec.DecoderID]struct{},
		len(supports),
	)
	decoderIDs := make([]basespec.DecoderID, 0, len(supports))

	for _, support := range supports {
		if err := support.Validate(); err != nil {
			return workspaceProviderConfiguration{}, err
		}
		if _, duplicate := seenKinds[support.Kind]; duplicate {
			return workspaceProviderConfiguration{}, fmt.Errorf(
				"%w: duplicate Workspace Artifact support %q",
				spec.ErrInvalidWorkspace,
				support.Kind,
			)
		}
		seenKinds[support.Kind] = struct{}{}

		if _, duplicate := seenDecoders[support.DecoderID]; duplicate {
			continue
		}
		seenDecoders[support.DecoderID] = struct{}{}
		decoderIDs = append(decoderIDs, support.DecoderID)
	}
	slices.Sort(decoderIDs)

	revision := input.DiscoveryPolicyRevision
	if revision == "" {
		revision = defaultDiscoveryPolicyRevision
	}
	if err := basespec.ValidateRequiredText(
		"Workspace provider discovery policy revision",
		revision,
		basespec.MaxVersionBytes,
	); err != nil {
		return workspaceProviderConfiguration{}, err
	}

	skillRoots, err := normalizeWorkspaceSkillRoots(input.SkillRoots)
	if err != nil {
		return workspaceProviderConfiguration{}, err
	}

	profiles := cloneWorkspaceDiscoveryProfiles(input.DiscoveryProfiles)
	if workspaceDiscoveryProfilesEmpty(profiles) {
		profiles = defaultDiscoveryProfiles()
	}
	profiles.Primary = mergeWorkspaceDiscoveryProfile(
		profiles.Primary,
		contextadapter.DiscoveryProfile(),
	)
	profiles.Primary = mergeWorkspaceDiscoveryProfile(
		profiles.Primary,
		workspaceSkillDiscoveryProfile(skillRoots),
	)
	profiles.Attached = mergeWorkspaceDiscoveryProfile(
		profiles.Attached,
		workspaceSkillDiscoveryProfile(skillRoots),
	)

	if err := validateWorkspaceDiscoveryProfiles(profiles); err != nil {
		return workspaceProviderConfiguration{}, err
	}
	if len(profiles.Primary.ExplicitLocators) == 0 &&
		len(profiles.Primary.DirectoryRoots) == 0 {
		return workspaceProviderConfiguration{}, fmt.Errorf(
			"%w: Workspace primary discovery profile is required",
			spec.ErrInvalidWorkspace,
		)
	}

	return workspaceProviderConfiguration{
		workspaceRootID: input.WorkspaceRootID,
		supports:        supports,
		decoderIDs:      decoderIDs,
		profiles:        profiles,
		revision:        revision,
	}, nil
}

func normalizeWorkspaceSkillRoots(
	input []basespec.Locator,
) ([]basespec.Locator, error) {
	roots := append([]basespec.Locator(nil), input...)
	if len(roots) == 0 {
		roots = artifactbuiltin.WorkspaceSkillRoots()
	}

	seen := make(map[basespec.Locator]struct{}, len(roots))
	for _, root := range roots {
		if err := basespec.ValidateLocator(root, true); err != nil {
			return nil, err
		}
		if _, duplicate := seen[root]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate Workspace Skill root %q",
				spec.ErrInvalidWorkspace,
				root,
			)
		}
		seen[root] = struct{}{}
	}
	slices.Sort(roots)
	return roots, nil
}

func workspaceSkillDiscoveryProfile(
	roots []basespec.Locator,
) spec.DiscoveryProfile {
	output := spec.DiscoveryProfile{
		DirectoryRoots: make([]spec.DirectoryRoot, 0, len(roots)),
	}
	for _, root := range roots {
		output.DirectoryRoots = append(
			output.DirectoryRoots,
			spec.DirectoryRoot{
				Root:      root,
				Recursive: true,
				IncludePatterns: []string{
					string(artifactbuiltin.AgentSkillDefinitionFileName),
				},
			},
		)
	}
	return output
}

func defaultDiscoveryProfiles() spec.DiscoveryProfiles {
	return spec.DiscoveryProfiles{
		Primary: spec.DiscoveryProfile{},
		Attached: spec.DiscoveryProfile{
			DirectoryRoots: []spec.DirectoryRoot{{
				Root:      artifactbuiltin.RepositoryRootLocator,
				Recursive: true,
				IncludePatterns: []string{
					artifactbuiltin.WorkspaceMarkdownPattern,
				},
			}},
		},
	}
}
