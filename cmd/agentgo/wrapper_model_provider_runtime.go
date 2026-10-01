package main

import (
	"context"
	"fmt"
	"maps"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/inferencewrapper"
	inferencewrapperSpec "github.com/flexigpt/flexigpt-app/internal/inferencewrapper/spec"
)

type inferenceProviderRuntimePublisher struct {
	providers *inferencewrapper.ProviderSetAPI
}

func newInferenceProviderRuntimePublisher(
	providers *inferencewrapper.ProviderSetAPI,
) (*inferenceProviderRuntimePublisher, error) {
	if providers == nil {
		return nil, fmt.Errorf(
			"%w: inference Provider runtime publisher is unavailable",
			basespec.ErrInvalid,
		)
	}
	return &inferenceProviderRuntimePublisher{
		providers: providers,
	}, nil
}

func (p *inferenceProviderRuntimePublisher) ClearProvider(
	ctx context.Context,
	provider inferenceSpec.ProviderName,
) error {
	if p == nil || p.providers == nil {
		return basespec.ErrClosed
	}

	_, err := p.providers.DeleteProvider(
		ctx,
		&inferencewrapperSpec.DeleteProviderRequest{
			Provider: provider,
		},
	)
	return err
}

func (p *inferenceProviderRuntimePublisher) PublishProvider(
	ctx context.Context,
	provider inferenceSpec.ProviderParam,
) error {
	if p == nil || p.providers == nil {
		return basespec.ErrClosed
	}

	_, err := p.providers.AddProvider(
		ctx,
		&inferencewrapperSpec.AddProviderRequest{
			Provider: provider.Name,
			Body: &inferencewrapperSpec.AddProviderRequestBody{
				SDKType:                  provider.SDKType,
				Origin:                   provider.Origin,
				ChatCompletionPathPrefix: provider.ChatCompletionPathPrefix,
				APIKeyHeaderKey:          provider.APIKeyHeaderKey,
				DefaultHeaders:           maps.Clone(provider.DefaultHeaders),
			},
		},
	)
	if err != nil {
		return err
	}

	_, err = p.providers.SetProviderAPIKey(
		ctx,
		&inferencewrapperSpec.SetProviderAPIKeyRequest{
			Provider: provider.Name,
			Body: &inferencewrapperSpec.SetProviderAPIKeyRequestBody{
				APIKey: provider.APIKey,
			},
		},
	)
	return err
}
