package inferencewrapper

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/flexigpt/inference-go"
	"github.com/flexigpt/inference-go/capabilityoverride"
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/inferencewrapper/spec"
)

// registerRuntimeProvider creates one ephemeral inference-go provider for one
// resolved Model runtime configuration.
//
// The provider name is completion-scoped, preventing one request from
// replacing another request's Provider configuration or API key. The
// ProviderSetAPI is intentionally used as an inference-go execution engine,
// not as a persistence layer.
func (ps *ProviderSetAPI) registerRuntimeProvider(
	ctx context.Context,
	runtimeModel spec.RuntimeModel,
	completionKey string,
) (
	inferenceSpec.ProviderName,
	func(),
	error,
) {
	if ps == nil || ps.inner == nil {
		return "", nil, errors.New("provider set is not initialized")
	}
	if err := runtimeModel.Validate(); err != nil {
		return "", nil, err
	}

	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(
			completionKey+"\x00"+
				string(runtimeModel.ProviderParam.Name)+"\x00"+
				string(runtimeModel.ConfigurationFingerprint),
		))),
		cryptoutil.DigestSHA256Prefix,
	)
	if len(digest) > 24 {
		digest = digest[:24]
	}

	providerName := inferenceSpec.ProviderName(
		"runtime-" + digest,
	)
	config := &inference.AddProviderConfig{
		SDKType: runtimeModel.ProviderParam.SDKType,
		Origin:  runtimeModel.ProviderParam.Origin,
		ChatCompletionPathPrefix: runtimeModel.ProviderParam.
			ChatCompletionPathPrefix,
		APIKeyHeaderKey: runtimeModel.ProviderParam.APIKeyHeaderKey,
		DefaultHeaders: cloneStringMap(
			runtimeModel.ProviderParam.DefaultHeaders,
		),
	}

	if _, err := ps.inner.AddProvider(
		ctx,
		providerName,
		config,
	); err != nil {
		return "", nil, err
	}
	if runtimeModel.ProviderParam.APIKey != "" {
		if err := ps.inner.SetProviderAPIKey(
			ctx,
			providerName,
			runtimeModel.ProviderParam.APIKey,
		); err != nil {
			_ = ps.inner.DeleteProvider(
				context.WithoutCancel(ctx),
				providerName,
			)
			return "", nil, err
		}
	}

	//nolint:contextcheck // Ok.
	release := func() {
		_ = ps.inner.DeleteProvider(
			context.Background(),
			providerName,
		)
	}
	return providerName, release, nil
}

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

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	maps.Copy(output, input)
	return output
}
