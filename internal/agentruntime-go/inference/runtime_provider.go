package inference

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"maps"
	"time"

	inferenceGo "github.com/flexigpt/inference-go"
	"github.com/flexigpt/inference-go/capabilityoverride"
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	inferenceRuntimeSpec "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference/spec"
)

const runtimeProviderCleanupTimeout = 5 * time.Second

func (ps *ProviderSetAPI) PublishProvider(
	ctx context.Context,
	provider inferenceSpec.ProviderParam,
) error {
	return ps.publishRuntimeProvider(ctx, provider.Name, provider)
}

func (ps *ProviderSetAPI) ClearProvider(
	ctx context.Context,
	provider inferenceSpec.ProviderName,
) error {
	return ps.inner.DeleteProvider(ctx, provider)
}

func (ps *ProviderSetAPI) publishRuntimeProvider(
	ctx context.Context,
	name inferenceSpec.ProviderName,
	provider inferenceSpec.ProviderParam,
) error {
	_, err := ps.inner.AddProvider(
		ctx,
		name,
		&inferenceGo.AddProviderConfig{
			SDKType:                  provider.SDKType,
			Origin:                   provider.Origin,
			ChatCompletionPathPrefix: provider.ChatCompletionPathPrefix,
			APIKeyHeaderKey:          provider.APIKeyHeaderKey,
			DefaultHeaders:           maps.Clone(provider.DefaultHeaders),
		},
	)
	if err != nil {
		return err
	}

	if err := ps.inner.SetProviderAPIKey(ctx, name, provider.APIKey); err != nil {
		return errors.Join(err, ps.removeRuntimeProvider(ctx, name))
	}
	return nil
}

func (ps *ProviderSetAPI) removeRuntimeProvider(
	ctx context.Context,
	name inferenceSpec.ProviderName,
) error {
	cleanupContext, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		runtimeProviderCleanupTimeout,
	)
	defer cancel()

	return ps.inner.DeleteProvider(cleanupContext, name)
}

func (ps *ProviderSetAPI) registerRuntimeProvider(
	ctx context.Context,
	config inferenceRuntimeSpec.RuntimeConfiguration,
	completionKey string,
) (inferenceSpec.ProviderName, func(), error) {
	sum := sha256.Sum256([]byte(
		completionKey + "\x00" +
			string(config.ProviderParam.Name) + "\x00" +
			config.Fingerprint,
	))
	digest := hex.EncodeToString(sum[:])
	name := inferenceSpec.ProviderName("runtime-" + digest[:24])

	if err := ps.publishRuntimeProvider(ctx, name, config.ProviderParam); err != nil {
		return "", nil, err
	}

	return name, func() {
		if err := ps.removeRuntimeProvider(ctx, name); err != nil {
			ps.logger.Warn(
				"could not remove completion-scoped provider",
				"provider", name,
				"error", err,
			)
		}
	}, nil
}

func (ps *ProviderSetAPI) newRuntimeCapabilityResolver(
	ctx context.Context,
	provider inferenceSpec.ProviderName,
	config inferenceRuntimeSpec.RuntimeConfiguration,
	completionKey string,
) (inferenceSpec.ModelCapabilityResolver, error) {
	if len(config.CapabilityOverrides) == 0 {
		return nil, nil //nolint:nilnil // No authored override requires a resolver.
	}

	value := config.ProviderParam
	baselineResolver, err := ps.inner.NewPresetCapabilityResolver(
		ctx,
		provider,
		modelpreset.ProviderPreset{
			Name:                     provider,
			DisplayName:              string(provider),
			SDKType:                  value.SDKType,
			Origin:                   value.Origin,
			ChatCompletionPathPrefix: value.ChatCompletionPathPrefix,
			APIKeyHeaderKey:          value.APIKeyHeaderKey,
			DefaultHeaders:           maps.Clone(value.DefaultHeaders),
		},
		modelpreset.ModelPreset{
			ID:          modelpreset.ModelPresetID("runtime"),
			Name:        config.ModelParam.Name,
			DisplayName: string(config.ModelParam.Name),
			ModelParam: inferenceSpec.ModelParam{
				Name: config.ModelParam.Name,
			},
		},
		completionKey,
	)
	if err != nil {
		return nil, err
	}

	baseline, err := baselineResolver.ResolveModelCapabilities(
		ctx,
		inferenceSpec.ResolveModelCapabilitiesRequest{
			ProviderSDKType: value.SDKType,
			ModelName:       config.ModelParam.Name,
			CompletionKey:   completionKey,
		},
	)
	if err != nil {
		return nil, err
	}
	if baseline == nil {
		return nil, errors.New("inference-go returned no SDK capability baseline")
	}

	effective := capabilityoverride.DeriveModelCapabilities(
		*baseline,
		config.CapabilityOverrides...,
	)
	return capabilityoverride.NewCompletionKeyResolver(
		completionKey,
		&effective,
	), nil
}
