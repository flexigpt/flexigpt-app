package inferencewrapper

import (
	"context"
	"errors"

	"github.com/flexigpt/inference-go/capabilityoverride"
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/inferencewrapper/spec"
)

// Resolve capabilities through inference-go using the registered SDK baseline.
// The app supplies ordered authored overrides, never an invented base profile.
func (ps *ProviderSetAPI) newRuntimeCapabilityResolver(
	ctx context.Context,
	provider inferenceSpec.ProviderName,
	runtimeModel spec.RuntimeModel,
	completionKey string,
) (inferenceSpec.ModelCapabilityResolver, error) {
	if len(runtimeModel.CapabilityOverrides) == 0 {
		//nolint:nilnil // Explicit.
		return nil, nil
	}

	p := runtimeModel.ProviderParam
	baseResolver, err := ps.inner.NewPresetCapabilityResolver(
		ctx,
		provider,
		modelpreset.ProviderPreset{
			Name:                     provider,
			DisplayName:              string(provider),
			SDKType:                  p.SDKType,
			Origin:                   p.Origin,
			ChatCompletionPathPrefix: p.ChatCompletionPathPrefix,
			APIKeyHeaderKey:          p.APIKeyHeaderKey,
			DefaultHeaders:           cloneStringMap(p.DefaultHeaders),
		},
		modelpreset.ModelPreset{
			ID:          modelpreset.ModelPresetID("runtime"),
			Name:        runtimeModel.ModelParam.Name,
			DisplayName: string(runtimeModel.ModelParam.Name),
			ModelParam: inferenceSpec.ModelParam{
				Name: runtimeModel.ModelParam.Name,
			},
		},
		completionKey,
	)
	if err != nil {
		return nil, err
	}

	base, err := baseResolver.ResolveModelCapabilities(
		ctx,
		inferenceSpec.ResolveModelCapabilitiesRequest{
			ProviderSDKType: p.SDKType,
			ModelName:       runtimeModel.ModelParam.Name,
			CompletionKey:   completionKey,
		},
	)
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, errors.New("inference-go returned nil SDK capabilities")
	}

	// Override semantics and application are owned by inference-go.
	effective := capabilityoverride.DeriveModelCapabilities(
		*base,
		runtimeModel.CapabilityOverrides...,
	)
	return capabilityoverride.NewCompletionKeyResolver(completionKey, &effective), nil
}
