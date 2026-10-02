package spec

import (
	"fmt"

	"github.com/flexigpt/inference-go/capabilityoverride"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// RuntimeModel is the fully resolved runtime-only model configuration supplied
// by Model Aggregate. It contains no Artifact Store dependency and no
// ModelPresetStore dependency.
//
// ProviderParam.APIKey is intentionally non-serialized. It exists only in
// process memory for the duration of a completion.
type RuntimeModel struct {
	ProviderParam            inferenceSpec.ProviderParam                     `json:"-"`
	ModelParam               inferenceSpec.ModelParam                        `json:"-"`
	CapabilityOverrides      []*capabilityoverride.ModelCapabilitiesOverride `json:"-"`
	ConfigurationFingerprint cryptoutil.Digest                               `json:"-"`
}

func (v RuntimeModel) Validate() error {
	if err := model.ValidateRequiredText(
		"runtime Provider name",
		string(v.ProviderParam.Name),
		model.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := model.ValidateRequiredText(
		"runtime Provider SDK type",
		string(v.ProviderParam.SDKType),
		model.MaxKindBytes,
	); err != nil {
		return err
	}
	if err := model.ValidateRequiredText(
		"runtime Provider origin",
		v.ProviderParam.Origin,
		model.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := model.ValidateRequiredText(
		"runtime Provider path",
		v.ProviderParam.ChatCompletionPathPrefix,
		model.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := model.ValidateRequiredText(
		"runtime Provider API key header",
		v.ProviderParam.APIKeyHeaderKey,
		model.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := model.ValidateRequiredText(
		"runtime Model name",
		string(v.ModelParam.Name),
		model.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(
		v.ConfigurationFingerprint,
	); err != nil {
		return fmt.Errorf(
			"runtime Model configuration fingerprint: %w",
			err,
		)
	}
	return nil
}
