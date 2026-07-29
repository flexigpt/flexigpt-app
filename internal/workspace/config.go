package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

const defaultDiscoveryPolicyRevision = "workspace.discovery.v1"

type Config struct {
	Supports                []engine.ArtifactSupport
	DiscoveryProfiles       engine.DiscoveryProfiles
	DiscoveryPolicyRevision string
	SkillRoots              []basespec.Locator
	ContextComposition      contextadapter.CompositionPolicy
	SourceUsePolicy         engine.SourceUsePolicy
}

type builtinArtifactSupport struct {
	support engine.ArtifactSupport
}

// builtinArtifactSupportMatrix is the workspace artifact support matrix.
//
// DefaultConfig and decoder construction both derive from this matrix.
var builtinArtifactSupportMatrix = []builtinArtifactSupport{
	{
		support: contextadapter.ArtifactSupport(),
	},
	{
		support: engine.ArtifactSupport{
			Kind:      skillartifact.Kind,
			SchemaID:  skillartifact.SchemaID,
			DecoderID: skillartifact.DecoderID,
			Validator: skillartifact.ValidateDefinition,
		},
	},
}

func DefaultConfig() Config {
	return Config{
		Supports:                BuiltinArtifactSupports(),
		DiscoveryPolicyRevision: defaultDiscoveryPolicyRevision,
		SkillRoots:              skilladapter.DefaultSkillRoots(),
		ContextComposition:      contextadapter.DefaultCompositionPolicy(),
		SourceUsePolicy:         engine.NewArtifactRuntimePolicy(),
	}
}

func (c Config) normalized() Config {
	output := c
	if len(output.Supports) == 0 {
		output.Supports = BuiltinArtifactSupports()
	}
	if output.DiscoveryPolicyRevision == "" {
		output.DiscoveryPolicyRevision = defaultDiscoveryPolicyRevision
	}
	return output
}

func (c Config) normalizedDiscoveryProfiles(
	skillConventions *skilladapter.ConventionRegistry,
) engine.DiscoveryProfiles {
	var profiles engine.DiscoveryProfiles
	if len(c.DiscoveryProfiles.Primary.ExplicitLocators) == 0 &&
		len(c.DiscoveryProfiles.Primary.DirectoryRoots) == 0 &&
		len(c.DiscoveryProfiles.Attached.ExplicitLocators) == 0 &&
		len(c.DiscoveryProfiles.Attached.DirectoryRoots) == 0 {
		profiles = engine.DefaultDiscoveryProfiles()
	} else {
		profiles = c.DiscoveryProfiles
	}
	contextProfile := contextadapter.DiscoveryProfile()
	profiles.Primary = engine.MergeDiscoveryProfile(
		profiles.Primary,
		contextProfile,
	)
	skillProfile := skillConventions.DiscoveryProfile()
	profiles.Primary = engine.MergeDiscoveryProfile(
		profiles.Primary,
		skillProfile,
	)
	profiles.Attached = engine.MergeDiscoveryProfile(
		profiles.Attached,
		skillProfile,
	)
	return profiles
}

func (c Config) normalizedSupports() ([]engine.ArtifactSupport, error) {
	if len(c.Supports) == 0 {
		return nil, fmt.Errorf(
			"%w: workspace artifact support is required",
			engine.ErrInvalidWorkspace,
		)
	}

	output := make([]engine.ArtifactSupport, 0, len(c.Supports))
	seenKinds := make(map[basespec.ArtifactKind]struct{}, len(c.Supports))

	for _, support := range c.Supports {
		if err := support.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seenKinds[support.Kind]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate workspace artifact kind %q",
				engine.ErrInvalidWorkspace,
				support.Kind,
			)
		}
		seenKinds[support.Kind] = struct{}{}
		output = append(output, support)
	}
	return output, nil
}

func BuiltinArtifactSupports() []engine.ArtifactSupport {
	output := make(
		[]engine.ArtifactSupport,
		0,
		len(builtinArtifactSupportMatrix),
	)
	for _, value := range builtinArtifactSupportMatrix {
		output = append(output, value.support)
	}
	return output
}

func BuiltinDecoders() []discovery.Decoder {
	config := DefaultConfig()
	registry, err := config.skillConventions()
	if err != nil {
		panic(err)
	}
	decoder, err := skillartifact.NewDecoder(registry)
	if err != nil {
		panic(err)
	}
	return []discovery.Decoder{
		contextadapter.NewContextDecoder(),
		decoder,
	}
}

func BuiltinDiscoveryProfiles() engine.DiscoveryProfiles {
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
			engine.ErrInvalidWorkspace,
			err,
		)
	}
	return value, nil
}

func (c Config) runtimePolicy() engine.SourceUsePolicy {
	if c.SourceUsePolicy != nil {
		return c.SourceUsePolicy
	}
	return engine.NewArtifactRuntimePolicy()
}

func (c Config) contextCompositionPolicy() contextadapter.CompositionPolicy {
	return c.ContextComposition.Normalized()
}
