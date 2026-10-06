package inferencewrapper

import (
	"context"
	"errors"
	"maps"
	"strings"
	"time"

	"github.com/flexigpt/inference-go"
	"github.com/flexigpt/inference-go/capabilityoverride"
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	inferencewrapperSpec "github.com/flexigpt/flexigpt-app/internal/inferencewrapper/spec"
)

const runtimeProviderCleanupTimeout = 5 * time.Second

// PublishProvider implements the application provider-lifecycle publication
// capability. The supplied configuration has already been resolved by the
// runtime adapter; no store or credential resolution happens here.
func (ps *ProviderSetAPI) PublishProvider(
	ctx context.Context,
	provider inferenceSpec.ProviderParam,
) error {
	return ps.publishRuntimeProvider(ctx, provider.Name, provider)
}

// ClearProvider implements the application provider-lifecycle removal
// capability. Completion-scoped providers use separate generated names.
func (ps *ProviderSetAPI) ClearProvider(
	ctx context.Context,
	provider inferenceSpec.ProviderName,
) error {
	return ps.inner.DeleteProvider(ctx, provider)
}

// publishRuntimeProvider is shared by long-lived provider publication and
// completion-scoped provider registration.
func (ps *ProviderSetAPI) publishRuntimeProvider(
	ctx context.Context,
	name inferenceSpec.ProviderName,
	provider inferenceSpec.ProviderParam,
) error {
	config := &inference.AddProviderConfig{
		SDKType:                  provider.SDKType,
		Origin:                   provider.Origin,
		ChatCompletionPathPrefix: provider.ChatCompletionPathPrefix,
		APIKeyHeaderKey:          provider.APIKeyHeaderKey,
		DefaultHeaders:           maps.Clone(provider.DefaultHeaders),
	}
	if _, err := ps.inner.AddProvider(ctx, name, config); err != nil {
		return err
	}

	// An empty key is intentional, including after a credential is cleared.
	if err := ps.inner.SetProviderAPIKey(ctx, name, provider.APIKey); err != nil {
		return errors.Join(
			err,
			ps.removeRuntimeProvider(ctx, name),
		)
	}
	return nil
}

func (ps *ProviderSetAPI) removeRuntimeProvider(
	ctx context.Context,
	name inferenceSpec.ProviderName,
) error {
	cleanupCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		runtimeProviderCleanupTimeout,
	)
	defer cancel()

	return ps.inner.DeleteProvider(cleanupCtx, name)
}

// registerRuntimeProvider consumes the configuration already checked by
// FetchCompletion. Each completion receives an isolated provider name so
// concurrent requests cannot replace each other's credentials or settings.
func (ps *ProviderSetAPI) registerRuntimeProvider(
	ctx context.Context,
	runtimeModel inferencewrapperSpec.RuntimeConfiguration,
	completionKey string,
) (inferenceSpec.ProviderName, func(), error) {
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(
			completionKey+"\x00"+
				string(runtimeModel.ProviderParam.Name)+"\x00"+
				string(runtimeModel.Fingerprint),
		))),
		cryptoutil.DigestSHA256Prefix,
	)
	providerName := inferenceSpec.ProviderName("runtime-" + digest[:24])

	if err := ps.publishRuntimeProvider(
		ctx,
		providerName,
		runtimeModel.ProviderParam,
	); err != nil {
		return "", nil, err
	}

	release := func() {
		if err := ps.removeRuntimeProvider(ctx, providerName); err != nil {
			ps.logger.Warn(
				"could not remove completion-scoped Provider",
				"provider", providerName,
				"error", err,
			)
		}
	}
	return providerName, release, nil
}

// Resolve capabilities through inference-go using the registered SDK baseline.
// The app supplies ordered authored overrides, never an invented base profile.
func (ps *ProviderSetAPI) newRuntimeCapabilityResolver(
	ctx context.Context,
	provider inferenceSpec.ProviderName,
	runtimeModel inferencewrapperSpec.RuntimeConfiguration,
	completionKey string,
) (inferenceSpec.ModelCapabilityResolver, error) {
	if len(runtimeModel.CapabilityOverrides) == 0 {
		//nolint:nilnil // No override resolver is needed.
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
			DefaultHeaders:           maps.Clone(p.DefaultHeaders),
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

	effective := capabilityoverride.DeriveModelCapabilities(
		*base,
		runtimeModel.CapabilityOverrides...,
	)
	return capabilityoverride.NewCompletionKeyResolver(completionKey, &effective), nil
}
