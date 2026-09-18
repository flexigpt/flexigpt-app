package artifactfallback

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

const (
	// MappedTargetProviderV1 identifies ModelPresetStore-backed mapped targets.
	MappedTargetProviderV1 = "flexigpt.modelpreset.artifactfallback.v1"

	targetIdentifierVersion = "v1"
)

// TargetV1 is the stable identity of one built-in model preset.
type TargetV1 struct {
	ProviderName  inferenceSpec.ProviderName `json:"providerName"`
	ModelPresetID modelpreset.ModelPresetID  `json:"modelPresetID"`
	ModelName     inferenceSpec.ModelName    `json:"modelName"`
}

func (t TargetV1) Validate() error {
	if err := basespec.ValidateRequiredText(
		"Model fallback provider name",
		string(t.ProviderName),
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := bundleitemutils.ValidateTag(string(t.ModelPresetID)); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Model fallback model name",
		string(t.ModelName),
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return nil
}

// NewMappedTarget creates a Model Preset mapped target. The Artifact logical
// name may be an explicit alias and therefore does not need to equal ModelName.
func NewMappedTarget(
	name basespec.LogicalName,
	value TargetV1,
) (resolve.MappedTarget, error) {
	if err := name.Validate(); err != nil {
		return resolve.MappedTarget{}, err
	}
	if err := value.Validate(); err != nil {
		return resolve.MappedTarget{}, err
	}

	identifier, err := resolve.EncodeMappedIdentifier(
		targetIdentifierVersion,
		value,
	)
	if err != nil {
		return resolve.MappedTarget{}, err
	}

	return resolve.MappedTarget{
		Provider:   MappedTargetProviderV1,
		Identifier: identifier,
		Type:       declaration.TypeModel,
		Name:       name,
		Builtin:    true,
	}, nil
}

// DecodeTarget decodes and validates one Model fallback target without reading
// ModelPresetStore. Store-backed callers should use Service.ResolveTarget.
func DecodeTarget(
	target resolve.MappedTarget,
) (TargetV1, error) {
	if err := target.Validate(); err != nil {
		return TargetV1{}, err
	}
	if target.Provider != MappedTargetProviderV1 {
		return TargetV1{}, fmt.Errorf(
			"%w: unsupported Model mapped target provider %q",
			basespec.ErrUnsupported,
			target.Provider,
		)
	}
	if target.Type != declaration.TypeModel {
		return TargetV1{}, fmt.Errorf(
			"%w: mapped target type is %q, expected Model",
			basespec.ErrInvalid,
			target.Type,
		)
	}
	if !target.Builtin {
		return TargetV1{}, fmt.Errorf(
			"%w: Model mapped target is not built-in",
			basespec.ErrInvalid,
		)
	}

	value, err := resolve.DecodeMappedIdentifier[TargetV1](
		target.Identifier,
		targetIdentifierVersion,
	)
	if err != nil {
		return TargetV1{}, err
	}
	if err := value.Validate(); err != nil {
		return TargetV1{}, err
	}
	return value, nil
}
