package aggregate

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/toolv1"
	toolConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/consumerapi"
)

const (
	MappedTargetProviderV1 = "flexigpt.tool.aggregate.v1"
	targetIdentifierV1     = "v1"
)

type TargetV1 struct {
	ToolArtifact     artifactModel.ArtifactRef `json:"toolArtifact"`
	DefinitionDigest cryptoutil.Digest         `json:"definitionDigest"`
	Name             spec.LogicalName          `json:"name"`
	Version          spec.LogicalVersion       `json:"version"`
	Implementation   toolv1.ImplementationKind `json:"implementation"`
}

func (t TargetV1) Validate() error {
	if err := t.ToolArtifact.Validate(); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(t.DefinitionDigest); err != nil {
		return err
	}
	if err := t.Name.Validate(); err != nil {
		return err
	}
	if err := spec.ValidatePortableName(
		"mapped Tool version",
		string(t.Version),
	); err != nil {
		return err
	}
	switch t.Implementation {
	case toolv1.ImplementationKindGo, toolv1.ImplementationKindSDK:
		return nil
	default:
		return fmt.Errorf(
			"%w: mapped Tool implementation %q is unsupported",
			spec.ErrInvalid,
			t.Implementation,
		)
	}
}

func NewMappedTarget(
	value toolConsumerAPI.ResolvedToolView,
) (composition.MappedTarget, error) {
	if !value.Enabled() || !value.Tool.BuiltIn {
		return composition.MappedTarget{}, fmt.Errorf(
			"%w: Tool %q is disabled",
			spec.ErrReferenceUnresolved,
			value.Tool.Artifact.LogicalName,
		)
	}

	targetValue := TargetV1{
		ToolArtifact:     value.Tool.Artifact.Ref(),
		DefinitionDigest: value.Tool.DefinitionDigest,
		Name:             value.Tool.Name,
		Version:          value.Tool.Version,
		Implementation:   value.Tool.Implementation.Kind,
	}
	if err := targetValue.Validate(); err != nil {
		return composition.MappedTarget{}, err
	}

	identifier, err := composition.EncodeMappedIdentifier(
		targetIdentifierV1,
		targetValue,
	)
	if err != nil {
		return composition.MappedTarget{}, err
	}

	return composition.MappedTarget{
		Provider:   MappedTargetProviderV1,
		Identifier: identifier,
		Type:       declaration.TypeTool,
		Name:       targetValue.Name,
		Builtin:    true,
	}, nil
}

func DecodeTarget(
	target composition.MappedTarget,
) (TargetV1, error) {
	if err := target.Validate(); err != nil {
		return TargetV1{}, err
	}
	if target.Provider != MappedTargetProviderV1 {
		return TargetV1{}, fmt.Errorf(
			"%w: unsupported Tool mapped target provider %q",
			spec.ErrUnsupported,
			target.Provider,
		)
	}
	if target.Type != declaration.TypeTool || !target.Builtin {
		return TargetV1{}, fmt.Errorf(
			"%w: mapped target is not a built-in Tool",
			spec.ErrInvalid,
		)
	}

	value, err := composition.DecodeMappedIdentifier[TargetV1](
		target.Identifier,
		targetIdentifierV1,
	)
	if err != nil {
		return TargetV1{}, err
	}
	if err := value.Validate(); err != nil {
		return TargetV1{}, err
	}
	if target.Name != value.Name {
		return TargetV1{}, fmt.Errorf(
			"%w: mapped Tool target name differs from payload",
			spec.ErrInvalid,
		)
	}
	return value, nil
}
