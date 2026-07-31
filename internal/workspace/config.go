package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	artifactstoreDiscovery "github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
	"github.com/flexigpt/flexigpt-app/internal/workspace/artifactadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/discovery"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

const (
	defaultDiscoveryPolicyRevision = "workspace.discovery.v1"
	markdownFilePattern            = "*.md"
)

func defaultDiscoveryProfiles() spec.DiscoveryProfiles {
	return spec.DiscoveryProfiles{
		Primary: spec.DiscoveryProfile{},
		Attached: spec.DiscoveryProfile{
			DirectoryRoots: []spec.DirectoryRoot{
				{
					Root:      spec.RepositoryRootLocator,
					Recursive: true,
					IncludePatterns: []string{
						markdownFilePattern,
					},
				},
			},
		},
	}
}

type Config struct {
	Supports                []spec.ArtifactSupport
	DiscoveryProfiles       spec.DiscoveryProfiles
	DiscoveryPolicyRevision string
	SkillRoots              []basespec.Locator
	ContextComposition      contextadapter.CompositionPolicy
	SourceUsePolicy         artifactadapter.SourceUsePolicy
	Clock                   clockutil.Clock
	AutoAdoptionIDProvider  artifactadapter.ArtifactIDProvider
}

type defaultArtifactSupport struct {
	support spec.ArtifactSupport
}

// defaultArtifactSupportMatrix is the Workspace-local support matrix.
//
// DefaultConfig and decoder construction both derive from this matrix.
var defaultArtifactSupportMatrix = []defaultArtifactSupport{
	{
		support: contextadapter.ArtifactSupport(),
	},
	{
		support: spec.ArtifactSupport{
			Kind:      skillartifact.Kind,
			SchemaID:  skillartifact.SchemaID,
			DecoderID: skillartifact.DecoderID,
			Validator: skillartifact.ValidateDefinition,
		},
	},
}

func DefaultConfig() Config {
	return Config{
		Supports:                DefaultArtifactSupports(),
		DiscoveryPolicyRevision: defaultDiscoveryPolicyRevision,
		SkillRoots:              skilladapter.DefaultSkillRoots(),
		ContextComposition:      contextadapter.DefaultCompositionPolicy(),
		SourceUsePolicy:         artifactadapter.NewArtifactRuntimePolicy(),
	}
}

func (c Config) normalized() Config {
	output := c
	if len(output.Supports) == 0 {
		output.Supports = DefaultArtifactSupports()
	}
	if output.DiscoveryPolicyRevision == "" {
		output.DiscoveryPolicyRevision = defaultDiscoveryPolicyRevision
	}
	if output.Clock == nil {
		output.Clock = clockutil.System{}
	}
	return output
}

func (c Config) normalizedDiscoveryProfiles(
	skillConventions *skilladapter.ConventionRegistry,
) spec.DiscoveryProfiles {
	var profiles spec.DiscoveryProfiles
	if len(c.DiscoveryProfiles.Primary.ExplicitLocators) == 0 &&
		len(c.DiscoveryProfiles.Primary.DirectoryRoots) == 0 &&
		len(c.DiscoveryProfiles.Attached.ExplicitLocators) == 0 &&
		len(c.DiscoveryProfiles.Attached.DirectoryRoots) == 0 {
		profiles = defaultDiscoveryProfiles()
	} else {
		profiles = c.DiscoveryProfiles
	}
	contextProfile := contextadapter.DiscoveryProfile()
	profiles.Primary = discovery.MergeDiscoveryProfile(
		profiles.Primary,
		contextProfile,
	)
	skillProfile := skillConventions.DiscoveryProfile()
	profiles.Primary = discovery.MergeDiscoveryProfile(
		profiles.Primary,
		skillProfile,
	)
	profiles.Attached = discovery.MergeDiscoveryProfile(
		profiles.Attached,
		skillProfile,
	)
	return profiles
}

func (c Config) normalizedSupports() ([]spec.ArtifactSupport, error) {
	if len(c.Supports) == 0 {
		return nil, fmt.Errorf(
			"%w: workspace artifact support is required",
			spec.ErrInvalidWorkspace,
		)
	}

	output := make([]spec.ArtifactSupport, 0, len(c.Supports))
	seenKinds := make(map[basespec.ArtifactKind]struct{}, len(c.Supports))

	for _, support := range c.Supports {
		if err := support.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seenKinds[support.Kind]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate workspace artifact kind %q",
				spec.ErrInvalidWorkspace,
				support.Kind,
			)
		}
		seenKinds[support.Kind] = struct{}{}
		output = append(output, support)
	}
	return output, nil
}

func DefaultArtifactSupports() []spec.ArtifactSupport {
	output := make(
		[]spec.ArtifactSupport,
		0,
		len(defaultArtifactSupportMatrix),
	)
	for _, value := range defaultArtifactSupportMatrix {
		output = append(output, value.support)
	}
	return output
}

// DefaultDecoders contains Workspace-owned decoders only. The shared
// agent.skill decoder is registered exactly once by Artifact Store composition.
func DefaultDecoders() []artifactstoreDiscovery.Decoder {
	return []artifactstoreDiscovery.Decoder{
		contextadapter.NewContextDecoder(),
	}
}

func DefaultDiscoveryProfiles() spec.DiscoveryProfiles {
	config := DefaultConfig()
	registry, err := config.skillConventions()
	if err != nil {
		panic(err)
	}
	return config.normalizedDiscoveryProfiles(registry)
}

func (c Config) skillConventions() (*skilladapter.ConventionRegistry, error) {
	return skilladapter.NewConventionRegistry(c.SkillRoots...)
}

func (c Config) discoveryPolicyRevision() (string, error) {
	value := c.DiscoveryPolicyRevision
	if value == "" {
		value = defaultDiscoveryPolicyRevision
	}
	if err := basespec.ValidateRequiredText(
		"workspace discovery policy revision",
		value,
		basespec.MaxVersionBytes,
	); err != nil {
		return "", fmt.Errorf(
			"%w: %w",
			spec.ErrInvalidWorkspace,
			err,
		)
	}
	return value, nil
}

func (c Config) runtimePolicy() artifactadapter.SourceUsePolicy {
	if c.SourceUsePolicy != nil {
		return c.SourceUsePolicy
	}
	return artifactadapter.NewArtifactRuntimePolicy()
}

func (c Config) contextCompositionPolicy() contextadapter.CompositionPolicy {
	return c.ContextComposition.Normalized()
}
