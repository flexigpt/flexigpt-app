package modelruntime

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/flexigpt/inference-go/capabilityoverride"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	inferenceRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	modelAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model"
)

// PreparedRuntimeConfiguration is the durable non-secret portion of one
// effective Model invocation. Credentials are represented only by a
// Store-validated binding and are re-created privately by Model setup.
type PreparedRuntimeConfiguration struct {
	Provider    ProviderRuntimeSnapshot `json:"provider"`
	Model       jsonutil.CanonicalValue `json:"model"`
	Overrides   jsonutil.CanonicalValue `json:"overrides,omitempty"`
	Credential  jsonutil.CanonicalValue `json:"credential,omitempty"`
	Fingerprint string                  `json:"fingerprint"`
}

type ProviderRuntimeSnapshot struct {
	Name                     inferenceSpec.ProviderName    `json:"name"`
	SDKType                  inferenceSpec.ProviderSDKType `json:"sdkType"`
	Origin                   string                        `json:"origin"`
	ChatCompletionPathPrefix string                        `json:"chatCompletionPathPrefix"`
	APIKeyHeaderKey          string                        `json:"apiKeyHeaderKey"`
	DefaultHeaders           map[string]string             `json:"defaultHeaders,omitempty"`
}

func (v PreparedRuntimeConfiguration) Validate() error {
	if strings.TrimSpace(v.Fingerprint) == "" ||
		!v.Model.Valid() ||
		(len(v.Overrides) != 0 && !v.Overrides.Valid()) ||
		(len(v.Credential) != 0 && !v.Credential.Valid()) {
		return errors.New("prepared Model runtime configuration is invalid")
	}
	configuration := inferenceRuntime.RuntimeConfiguration{
		ProviderParam: inferenceSpec.ProviderParam{
			Name:                     v.Provider.Name,
			SDKType:                  v.Provider.SDKType,
			Origin:                   v.Provider.Origin,
			ChatCompletionPathPrefix: v.Provider.ChatCompletionPathPrefix,
			APIKeyHeaderKey:          v.Provider.APIKeyHeaderKey,
			DefaultHeaders:           maps.Clone(v.Provider.DefaultHeaders),
		},
		Fingerprint: v.Fingerprint,
	}
	return configuration.Validate()
}

func (a *RuntimeAdapter) PreparedRuntimeConfiguration(
	resolved modelAPI.ResolvedModel,
	value inferenceRuntime.RuntimeConfiguration,
) (jsonutil.CanonicalValue, error) {
	modelRaw, err := jsonutil.EncodeToJSONRaw(value.ModelParam)
	if err != nil {
		return nil, err
	}
	model, err := jsonutil.CanonicalValueFrom(modelRaw)
	if err != nil {
		return nil, err
	}

	overridesRaw, err := jsonutil.EncodeToJSONRaw(
		value.CapabilityOverrides,
	)
	if err != nil {
		return nil, err
	}
	overrides, err := jsonutil.CanonicalValueFrom(overridesRaw)
	if err != nil {
		return nil, err
	}

	var credential jsonutil.CanonicalValue
	if resolved.ProviderCredential != nil {
		raw, encodeErr := jsonutil.EncodeToJSONRaw(
			resolved.ProviderCredential.Clone(),
		)
		if encodeErr != nil {
			return nil, encodeErr
		}
		credential, err = jsonutil.CanonicalValueFrom(raw)
		if err != nil {
			return nil, err
		}
	}

	snapshot := PreparedRuntimeConfiguration{
		Provider: ProviderRuntimeSnapshot{
			Name:                     value.ProviderParam.Name,
			SDKType:                  value.ProviderParam.SDKType,
			Origin:                   value.ProviderParam.Origin,
			ChatCompletionPathPrefix: value.ProviderParam.ChatCompletionPathPrefix,
			APIKeyHeaderKey:          value.ProviderParam.APIKeyHeaderKey,
			DefaultHeaders:           maps.Clone(value.ProviderParam.DefaultHeaders),
		},
		Model:       model,
		Overrides:   overrides,
		Credential:  credential,
		Fingerprint: value.Fingerprint,
	}
	if err := snapshot.Validate(); err != nil {
		return nil, err
	}
	raw, err := jsonutil.EncodeToJSONRaw(snapshot)
	if err != nil {
		return nil, err
	}
	return jsonutil.CanonicalValueFrom(raw)
}

// FinalizePreparedRuntimeConfiguration captures the final effective non-secret
// Model configuration after Session, Workspace, Agent, Skill, instruction,
// and explicit invocation composition has completed. It retains the original
// credential binding but never serializes a credential value.
func FinalizePreparedRuntimeConfiguration(
	raw jsonutil.CanonicalValue,
	value inferenceRuntime.RuntimeConfiguration,
) (inferenceRuntime.RuntimeConfiguration, jsonutil.CanonicalValue, error) {
	snapshot, err := jsonutil.DecodeCanonicalValue[PreparedRuntimeConfiguration](raw)
	if err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, nil, err
	}
	if err := snapshot.Validate(); err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, nil, err
	}
	modelRaw, err := jsonutil.EncodeToJSONRaw(value.ModelParam)
	if err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, nil, err
	}
	model, err := jsonutil.CanonicalValueFrom(modelRaw)
	if err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, nil, err
	}
	overridesRaw, err := jsonutil.EncodeToJSONRaw(value.CapabilityOverrides)
	if err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, nil, err
	}
	overrides, err := jsonutil.CanonicalValueFrom(overridesRaw)
	if err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, nil, err
	}
	snapshot.Model = model
	snapshot.Overrides = overrides
	fingerprint, err := cryptoutil.CanonicalDigest(struct {
		Provider   ProviderRuntimeSnapshot `json:"provider"`
		Model      jsonutil.CanonicalValue `json:"model"`
		Overrides  jsonutil.CanonicalValue `json:"overrides,omitempty"`
		Credential jsonutil.CanonicalValue `json:"credential,omitempty"`
	}{
		Provider:   snapshot.Provider,
		Model:      snapshot.Model,
		Overrides:  snapshot.Overrides,
		Credential: snapshot.Credential,
	})
	if err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, nil, err
	}
	snapshot.Fingerprint = string(fingerprint)
	value.Fingerprint = string(fingerprint)
	if err := snapshot.Validate(); err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, nil, err
	}
	finalRaw, err := jsonutil.EncodeToJSONRaw(snapshot)
	if err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, nil, err
	}
	final, err := jsonutil.CanonicalValueFrom(finalRaw)
	if err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, nil, err
	}
	return value, final, nil
}

func (a *RuntimeAdapter) RehydrateRuntimeConfiguration(
	ctx context.Context,
	raw jsonutil.CanonicalValue,
) (inferenceRuntime.RuntimeConfiguration, error) {
	snapshot, err := jsonutil.DecodeCanonicalValue[PreparedRuntimeConfiguration](raw)
	if err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, err
	}
	if err := snapshot.Validate(); err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, err
	}

	model, err := jsonutil.DecodeCanonicalValue[inferenceSpec.ModelParam](
		snapshot.Model,
	)
	if err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, err
	}

	overrideValues := []capabilityoverride.ModelCapabilitiesOverride(nil)
	if len(snapshot.Overrides) != 0 {
		overrideValues, err = jsonutil.DecodeCanonicalValue[[]capabilityoverride.ModelCapabilitiesOverride](
			snapshot.Overrides,
		)
		if err != nil {
			return inferenceRuntime.RuntimeConfiguration{}, err
		}
	}
	overrides := make(
		[]*capabilityoverride.ModelCapabilitiesOverride,
		0,
		len(overrideValues),
	)
	for index := range overrideValues {
		value := overrideValues[index]
		overrides = append(overrides, &value)
	}

	apiKey := ""
	if len(snapshot.Credential) != 0 {
		binding, decodeErr := jsonutil.DecodeCanonicalValue[secretModel.Binding](snapshot.Credential)
		if decodeErr != nil {
			return inferenceRuntime.RuntimeConfiguration{}, decodeErr
		}
		credential, resolveErr := a.credentials.ResolveModelCredential(ctx, binding)
		if resolveErr != nil {
			return inferenceRuntime.RuntimeConfiguration{}, resolveErr
		}
		apiKey = credential.APIKey
	}

	output := inferenceRuntime.RuntimeConfiguration{
		ProviderParam: inferenceSpec.ProviderParam{
			Name:                     snapshot.Provider.Name,
			SDKType:                  snapshot.Provider.SDKType,
			APIKey:                   apiKey,
			Origin:                   snapshot.Provider.Origin,
			ChatCompletionPathPrefix: snapshot.Provider.ChatCompletionPathPrefix,
			APIKeyHeaderKey:          snapshot.Provider.APIKeyHeaderKey,
			DefaultHeaders:           maps.Clone(snapshot.Provider.DefaultHeaders),
		},
		ModelParam:          model,
		CapabilityOverrides: overrides,
		Fingerprint:         snapshot.Fingerprint,
	}
	if err := output.Validate(); err != nil {
		return inferenceRuntime.RuntimeConfiguration{}, err
	}
	return output, nil
}
