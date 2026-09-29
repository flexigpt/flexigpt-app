package inferencewrapper

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/flexigpt/inference-go"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	inferencewrapperSpec "github.com/flexigpt/flexigpt-app/internal/inferencewrapper/spec"
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
	runtimeModel inferencewrapperSpec.RuntimeModel,
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

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	maps.Copy(output, input)
	return output
}
