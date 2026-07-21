package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
)

type ArtifactSupport struct {
	Kind      artifactstore.ArtifactKind
	SchemaID  artifactstore.SchemaID
	DecoderID artifactstore.DecoderID
}

func (s ArtifactSupport) Validate() error {
	if err := artifactstore.ValidateArtifactKind(s.Kind); err != nil {
		return err
	}
	if err := artifactstore.ValidateSchemaID(s.SchemaID); err != nil {
		return err
	}
	return artifactstore.ValidateDecoderID(s.DecoderID)
}

type Config struct {
	Supports []ArtifactSupport
}

type builtinArtifactSupport struct {
	support    ArtifactSupport
	newDecoder func() discovery.Decoder
}

// builtinArtifactSupportMatrix is the workspace artifact support matrix.
//
// DefaultConfig and BuiltinDecoders both derive from this matrix.
var builtinArtifactSupportMatrix = [...]builtinArtifactSupport{
	{
		support: ArtifactSupport{
			Kind:      DefinitionKind,
			SchemaID:  DefinitionSchemaID,
			DecoderID: DefinitionDecoderID,
		},
		newDecoder: func() discovery.Decoder {
			return NewDefinitionDecoder()
		},
	},
	{
		support: ArtifactSupport{
			Kind:      ContextKind,
			SchemaID:  ContextSchemaID,
			DecoderID: ContextDecoderID,
		},
		newDecoder: func() discovery.Decoder {
			return NewContextDecoder()
		},
	},
	{
		support: ArtifactSupport{
			Kind:      SkillKind,
			SchemaID:  SkillSchemaID,
			DecoderID: SkillDecoderID,
		},
		newDecoder: func() discovery.Decoder {
			return NewSkillDecoder()
		},
	},
}

func DefaultConfig() Config {
	return Config{Supports: BuiltinArtifactSupports()}
}

func BuiltinArtifactSupports() []ArtifactSupport {
	output := make(
		[]ArtifactSupport,
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

func (c Config) normalizedSupports() ([]ArtifactSupport, error) {
	if len(c.Supports) == 0 {
		return nil, fmt.Errorf(
			"%w: workspace artifact support is required",
			ErrInvalidWorkspace,
		)
	}

	output := make([]ArtifactSupport, 0, len(c.Supports))
	seenKinds := make(map[artifactstore.ArtifactKind]struct{}, len(c.Supports))
	definitionSupported := false

	for _, support := range c.Supports {
		if err := support.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seenKinds[support.Kind]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate workspace artifact kind %q",
				ErrInvalidWorkspace,
				support.Kind,
			)
		}
		seenKinds[support.Kind] = struct{}{}

		if support.Kind == DefinitionKind {
			if support.SchemaID != DefinitionSchemaID ||
				support.DecoderID != DefinitionDecoderID {
				return nil, fmt.Errorf(
					"%w: workspace definition support is fixed",
					ErrInvalidWorkspace,
				)
			}
			definitionSupported = true
		}
		output = append(output, support)
	}

	if !definitionSupported {
		return nil, fmt.Errorf(
			"%w: workspace definition support is required",
			ErrInvalidWorkspace,
		)
	}
	return output, nil
}
