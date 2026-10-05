package aggregate

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/consumerapi"
)

const (
	MappedTargetProviderV1 = "flexigpt.spec.aggregate.v1"
	targetIdentifierV1     = "v1"
)

type TargetV1 struct {
	ModelArtifact            artifactModel.ArtifactRef `json:"modelArtifact"`
	ModelDefinitionDigest    cryptoutil.Digest         `json:"modelDefinitionDigest"`
	ProviderArtifact         artifactModel.ArtifactRef `json:"providerArtifact"`
	ProviderDefinitionDigest cryptoutil.Digest         `json:"providerDefinitionDigest"`
	ConfigurationFingerprint cryptoutil.Digest         `json:"configurationFingerprint"`
	Name                     spec.LogicalName          `json:"name"`
}

func (t TargetV1) Validate() error {
	if err := t.ModelArtifact.Validate(); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(
		t.ModelDefinitionDigest,
	); err != nil {
		return err
	}
	if err := t.ProviderArtifact.Validate(); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(
		t.ProviderDefinitionDigest,
	); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(
		t.ConfigurationFingerprint,
	); err != nil {
		return err
	}
	return t.Name.Validate()
}

func NewMappedTarget(
	value modelConsumerAPI.ResolvedModel,
) (composition.MappedTarget, error) {
	targetValue := TargetV1{
		ModelArtifact:            value.Model.Artifact.Ref(),
		ModelDefinitionDigest:    value.Model.Definition.Digest,
		ProviderArtifact:         value.Provider.Artifact.Ref(),
		ProviderDefinitionDigest: value.Provider.Definition.Digest,
		ConfigurationFingerprint: value.Fingerprint,
		Name:                     value.Model.Artifact.LogicalName,
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
		Type:       declaration.TypeModel,
		Name:       targetValue.Name,
		Builtin:    value.Model.Artifact.RootID == topology.BuiltinRootID(),
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
			"%w: unsupported Model mapped target provider %q",
			spec.ErrUnsupported,
			target.Provider,
		)
	}
	if target.Type != declaration.TypeModel {
		return TargetV1{}, fmt.Errorf(
			"%w: mapped target type is %q, expected Model",
			spec.ErrInvalid,
			target.Type,
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
			"%w: mapped Model target name differs from payload",
			spec.ErrInvalid,
		)
	}
	return value, nil
}
