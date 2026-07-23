package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

type Config struct {
	Supports           []engine.ArtifactSupport
	DiscoveryProfiles  engine.DiscoveryProfiles
	SkillRoots         []artifactstore.Locator
	ContextComposition contextadapter.CompositionPolicy
	SourceUsePolicy    engine.SourceUsePolicy
}

type builtinArtifactSupport struct {
	support engine.ArtifactSupport
}

// builtinArtifactSupportMatrix is the workspace artifact support matrix.
//
// DefaultConfig and decoder construction both derive from this matrix.
var builtinArtifactSupportMatrix = []builtinArtifactSupport{
	{
		support: engine.ArtifactSupport{
			Kind:      engine.DefinitionKind,
			SchemaID:  engine.DefinitionSchemaID,
			DecoderID: engine.DefinitionDecoderID,
			Validator: engine.ValidateWorkspaceDefinition,
		},
	},
	{
		support: contextadapter.ArtifactSupport(),
	},
	{
		support: skilladapter.ArtifactSupport(),
	},
}

func DefaultConfig() Config {
	return Config{
		Supports:           BuiltinArtifactSupports(),
		SkillRoots:         skilladapter.DefaultSkillRoots(),
		ContextComposition: contextadapter.DefaultCompositionPolicy(),
		SourceUsePolicy:    engine.NewRecordRuntimePolicy(),
	}
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
	profiles.Primary.ExplicitLocators = append(
		profiles.Primary.ExplicitLocators,
		contextProfile.ExplicitLocators...,
	)
	profiles.Primary.ReadmeLocator = contextProfile.ReadmeLocator
	skillProfile := skilladapter.DiscoveryProfileWithConventions(
		skillConventions,
	)
	profiles.Primary.DirectoryRoots = append(
		profiles.Primary.DirectoryRoots,
		skillProfile.DirectoryRoots...,
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
	seenKinds := make(map[artifactstore.ArtifactKind]struct{}, len(c.Supports))
	definitionSupported := false

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

		if support.Kind == engine.DefinitionKind {
			if support.SchemaID != engine.DefinitionSchemaID ||
				support.DecoderID != engine.DefinitionDecoderID {
				return nil, fmt.Errorf(
					"%w: workspace definition support is fixed",
					engine.ErrInvalidWorkspace,
				)
			}
			definitionSupported = true
		}
		output = append(output, support)
	}

	if !definitionSupported {
		return nil, fmt.Errorf(
			"%w: workspace definition support is required",
			engine.ErrInvalidWorkspace,
		)
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
	decoder, err := skilladapter.NewSkillDecoderWithConventions(registry)
	if err != nil {
		panic(err)
	}
	return []discovery.Decoder{
		engine.NewDefinitionDecoder(),
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

func (c Config) runtimePolicy() engine.SourceUsePolicy {
	if c.SourceUsePolicy != nil {
		return c.SourceUsePolicy
	}
	return engine.NewRecordRuntimePolicy()
}

func (c Config) contextCompositionPolicy() contextadapter.CompositionPolicy {
	return c.ContextComposition.Normalized()
}
