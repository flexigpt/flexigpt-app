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
	Supports          []engine.ArtifactSupport
	DiscoveryProfiles engine.DiscoveryProfiles
}

type builtinArtifactSupport struct {
	support    engine.ArtifactSupport
	newDecoder func() discovery.Decoder
}

// builtinArtifactSupportMatrix is the workspace artifact support matrix.
//
// DefaultConfig and BuiltinDecoders both derive from this matrix.
var builtinArtifactSupportMatrix = []builtinArtifactSupport{
	{
		support: engine.ArtifactSupport{
			Kind:      engine.DefinitionKind,
			SchemaID:  engine.DefinitionSchemaID,
			DecoderID: engine.DefinitionDecoderID,
		},
		newDecoder: func() discovery.Decoder {
			return engine.NewDefinitionDecoder()
		},
	},
	{
		support: contextadapter.GetContextArtifactSupport(),
		newDecoder: func() discovery.Decoder {
			return contextadapter.NewContextDecoder()
		},
	},
	{
		support: skilladapter.GetSkillArtifactSupport(),
		newDecoder: func() discovery.Decoder {
			return skilladapter.NewSkillDecoder()
		},
	},
}

func DefaultConfig() Config {
	return Config{
		Supports:          BuiltinArtifactSupports(),
		DiscoveryProfiles: BuiltinDiscoveryProfiles(),
	}
}

func (c Config) normalizedDiscoveryProfiles() engine.DiscoveryProfiles {
	if len(c.DiscoveryProfiles.Primary.ExplicitLocators) == 0 &&
		len(c.DiscoveryProfiles.Primary.DirectoryRoots) == 0 &&
		len(c.DiscoveryProfiles.Attached.ExplicitLocators) == 0 &&
		len(c.DiscoveryProfiles.Attached.DirectoryRoots) == 0 {
		return BuiltinDiscoveryProfiles()
	}
	return c.DiscoveryProfiles
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
	output := make(
		[]discovery.Decoder,
		0,
		len(builtinArtifactSupportMatrix),
	)
	for _, value := range builtinArtifactSupportMatrix {
		output = append(output, value.newDecoder())
	}
	return output
}

func BuiltinDiscoveryProfiles() engine.DiscoveryProfiles {
	profiles := engine.DefaultDiscoveryProfiles()

	contextProfile := contextadapter.PrimaryDiscoveryProfile()
	profiles.Primary.ExplicitLocators = append(
		profiles.Primary.ExplicitLocators,
		contextProfile.ExplicitLocators...,
	)
	profiles.Primary.ReadmeLocator = contextProfile.ReadmeLocator

	skillProfile := skilladapter.PrimaryDiscoveryProfile()
	profiles.Primary.DirectoryRoots = append(
		profiles.Primary.DirectoryRoots,
		skillProfile.DirectoryRoots...,
	)
	return profiles
}
