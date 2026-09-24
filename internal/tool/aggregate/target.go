package aggregate

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

const (
	MappedTargetProviderV1 = "flexigpt.tool.aggregate.v1"
	targetIdentifierV1     = "v1"
)

type TargetV1 struct {
	ToolArtifact     artifact.ArtifactRef `json:"toolArtifact"`
	DefinitionDigest cryptoutil.Digest    `json:"definitionDigest"`
	Name             basespec.LogicalName `json:"name"`
	Version          string               `json:"version"`
	Implementation   string               `json:"implementation"`
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
	if err := basespec.ValidateRequiredText(
		"mapped Tool version",
		t.Version,
		basespec.MaxVersionBytes,
	); err != nil {
		return err
	}
	switch t.Implementation {
	case "go", "sdk":
		return nil
	default:
		return fmt.Errorf(
			"%w: mapped Tool implementation %q is unsupported",
			basespec.ErrInvalid,
			t.Implementation,
		)
	}
}

func NewMappedTarget(
	value toolDomain.ResolvedTool,
) (resolve.MappedTarget, error) {
	if !value.Enabled() {
		return resolve.MappedTarget{}, fmt.Errorf(
			"%w: Tool %q is disabled",
			basespec.ErrReferenceUnresolved,
			value.Tool.Artifact.LogicalName,
		)
	}

	document := value.Tool.Document
	targetValue := TargetV1{
		ToolArtifact:     value.Tool.Artifact.Ref(),
		DefinitionDigest: value.Tool.Definition.Digest,
		Name:             value.Tool.Artifact.LogicalName,
		Version:          string(document.Version),
		Implementation:   string(document.Implementation.Kind),
	}
	if err := targetValue.Validate(); err != nil {
		return resolve.MappedTarget{}, err
	}

	identifier, err := resolve.EncodeMappedIdentifier(
		targetIdentifierV1,
		targetValue,
	)
	if err != nil {
		return resolve.MappedTarget{}, err
	}

	return resolve.MappedTarget{
		Provider:   MappedTargetProviderV1,
		Identifier: identifier,
		Type:       declaration.TypeTool,
		Name:       targetValue.Name,
		Builtin:    true,
	}, nil
}

func DecodeTarget(
	target resolve.MappedTarget,
) (TargetV1, error) {
	if err := target.Validate(); err != nil {
		return TargetV1{}, err
	}
	if target.Provider != MappedTargetProviderV1 {
		return TargetV1{}, fmt.Errorf(
			"%w: unsupported Tool mapped target provider %q",
			basespec.ErrUnsupported,
			target.Provider,
		)
	}
	if target.Type != declaration.TypeTool || !target.Builtin {
		return TargetV1{}, fmt.Errorf(
			"%w: mapped target is not a built-in Tool",
			basespec.ErrInvalid,
		)
	}

	value, err := resolve.DecodeMappedIdentifier[TargetV1](
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
			basespec.ErrInvalid,
		)
	}
	return value, nil
}
