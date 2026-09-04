package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	skillArtifact "github.com/flexigpt/flexigpt-app/internal/skill/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/workspace/artifactadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type Config struct {
	WorkspaceRootID    basespec.RootID
	Supports           []spec.ArtifactSupport
	ContextComposition contextadapter.CompositionPolicy
	SourceUsePolicy    artifactadapter.SourceUsePolicy
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
			Kind:      artifactbuiltin.AgentSkillArtifactKind,
			SchemaID:  artifactbuiltin.AgentSkillSchemaID,
			DecoderID: artifactbuiltin.AgentSkillDecoderID,
			Validator: skillArtifact.ValidateDefinition,
		},
	},
}

func (c Config) normalized() Config {
	output := c
	if len(output.Supports) == 0 {
		output.Supports = DefaultArtifactSupports()
	}
	if output.WorkspaceRootID == "" {
		output.WorkspaceRootID = artifactbuiltin.WorkspaceRootID
	}
	return output
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

func (c Config) runtimePolicy() artifactadapter.SourceUsePolicy {
	if c.SourceUsePolicy != nil {
		return c.SourceUsePolicy
	}
	return artifactadapter.NewArtifactRuntimePolicy()
}

func (c Config) contextCompositionPolicy() contextadapter.CompositionPolicy {
	return c.ContextComposition.Normalized()
}

func DefaultConfig() Config {
	return Config{
		WorkspaceRootID:    artifactbuiltin.WorkspaceRootID,
		Supports:           DefaultArtifactSupports(),
		ContextComposition: contextadapter.DefaultCompositionPolicy(),
		SourceUsePolicy:    artifactadapter.NewArtifactRuntimePolicy(),
	}
}
