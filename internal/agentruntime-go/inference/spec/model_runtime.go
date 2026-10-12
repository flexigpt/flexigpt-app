package spec

import (
	"context"
	"fmt"

	"github.com/flexigpt/inference-go/capabilityoverride"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
)

// ModelRuntime is the source-backed Model capability consumed by inference.
// The implementation owns source resolution and request-patch preparation.
type ModelRuntime interface {
	ResolveRuntimeConfiguration(
		ctx context.Context,
		request RuntimeModelRequest,
	) (RuntimeConfiguration, error)
}

// RuntimeConfiguration is the single resolved inference configuration contract.
// It contains runtime credentials and must never be serialized or persisted.
type RuntimeConfiguration struct {
	ProviderParam       inferenceSpec.ProviderParam                     `json:"-"`
	ModelParam          inferenceSpec.ModelParam                        `json:"-"`
	CapabilityOverrides []*capabilityoverride.ModelCapabilitiesOverride `json:"-"`
	Fingerprint         cryptoutil.Digest                               `json:"-"`
}

// Validate checks the resolved result at the inference execution boundary.
// Private provider-registration helpers consume this already-checked value.
func (v RuntimeConfiguration) Validate() error {
	fields := []struct {
		name    string
		value   string
		maximum int
	}{
		{
			name:    "runtime Provider name",
			value:   string(v.ProviderParam.Name),
			maximum: storeSpec.MaxURIBytes,
		},
		{
			name:    "runtime Provider SDK type",
			value:   string(v.ProviderParam.SDKType),
			maximum: storeSpec.MaxKindBytes,
		},
		{
			name:    "runtime Provider origin",
			value:   v.ProviderParam.Origin,
			maximum: storeSpec.MaxURIBytes,
		},
		{
			name:    "runtime Provider path",
			value:   v.ProviderParam.ChatCompletionPathPrefix,
			maximum: storeSpec.MaxURIBytes,
		},
		{
			name:    "runtime Provider API key header",
			value:   v.ProviderParam.APIKeyHeaderKey,
			maximum: storeSpec.MaxURIBytes,
		},
		{
			name:    "runtime Model name",
			value:   string(v.ModelParam.Name),
			maximum: storeSpec.MaxURIBytes,
		},
	}
	for _, field := range fields {
		if err := storeSpec.ValidateRequiredText(
			field.name,
			field.value,
			field.maximum,
		); err != nil {
			return err
		}
	}
	if err := cryptoutil.ValidateDigest(v.Fingerprint); err != nil {
		return fmt.Errorf("runtime Model configuration fingerprint: %w", err)
	}
	return nil
}

// RuntimeModelRequest identifies a source-backed Model and its final optional
// request-level defaults layer.
type RuntimeModelRequest struct {
	Model        artifactModel.ArtifactRef `json:"model"`
	RequestPatch *RuntimeRequestPatch      `json:"requestPatch,omitempty"`
}

// RuntimeRequestPatch is a typed portable defaults layer. A nil pointer means
// no request override. Preparation belongs to the ModelRuntime implementation.
type RuntimeRequestPatch struct {
	Defaults *modelDomain.DefaultsPatch `json:"defaults,omitempty"`
	Clear    []RuntimeDefaultField      `json:"clear,omitempty"`
}

type RuntimeDefaultField string

const (
	RuntimeDefaultFieldAdapterParameters RuntimeDefaultField = "adapterParameters"
	RuntimeDefaultFieldCacheControl      RuntimeDefaultField = "cacheControl"
	RuntimeDefaultFieldOutput            RuntimeDefaultField = "output"
	RuntimeDefaultFieldReasoning         RuntimeDefaultField = "reasoning"
	RuntimeDefaultFieldStopSequences     RuntimeDefaultField = "stopSequences"
	RuntimeDefaultFieldTemperature       RuntimeDefaultField = "temperature"
)

func (f RuntimeDefaultField) Validate() error {
	switch f {
	case RuntimeDefaultFieldAdapterParameters,
		RuntimeDefaultFieldCacheControl,
		RuntimeDefaultFieldOutput,
		RuntimeDefaultFieldReasoning,
		RuntimeDefaultFieldStopSequences,
		RuntimeDefaultFieldTemperature:
		return nil
	default:
		return fmt.Errorf(
			"%w: unsupported Model runtime request clear field %q",
			storeSpec.ErrInvalid,
			f,
		)
	}
}
