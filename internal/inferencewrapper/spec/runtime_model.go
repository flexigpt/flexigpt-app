package spec

import (
	"fmt"

	"github.com/flexigpt/inference-go/capabilityoverride"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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
	if err := spec.ValidateRequiredText(
		"runtime Provider name",
		string(v.ProviderParam.Name),
		spec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"runtime Provider SDK type",
		string(v.ProviderParam.SDKType),
		spec.MaxKindBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"runtime Provider origin",
		v.ProviderParam.Origin,
		spec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"runtime Provider path",
		v.ProviderParam.ChatCompletionPathPrefix,
		spec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"runtime Provider API key header",
		v.ProviderParam.APIKeyHeaderKey,
		spec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"runtime Model name",
		string(v.ModelParam.Name),
		spec.MaxURIBytes,
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
