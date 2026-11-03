package inference

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"strings"

	inferenceGo "github.com/flexigpt/inference-go"
	"github.com/flexigpt/inference-go/debugclient"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

const (
	defaultFlushIntervalMillis  = 32
	defaultFlushChunkSize       = 256
	emptyCompletionResponseCode = "empty_completion_response"
)

// ProviderSetAPI owns provider lifecycle, completion-scoped provider
// registration, SDK capability resolution, provider execution, and transport
// streaming. It has no domain-resolution dependencies.
type ProviderSetAPI struct {
	inner    *inferenceGo.ProviderSetAPI
	logger   *slog.Logger
	debugger *debugclient.HTTPCompletionDebugger
}

type providerSetOptions struct {
	logger      *slog.Logger
	debugConfig *debugclient.DebugConfig
}

type ProviderSetOption func(*providerSetOptions)

func WithLogger(logger *slog.Logger) ProviderSetOption {
	return func(options *providerSetOptions) {
		options.logger = logger
	}
}

func WithDebugConfig(config *debugclient.DebugConfig) ProviderSetOption {
	return func(options *providerSetOptions) {
		if config == nil {
			options.debugConfig = nil
			return
		}
		value := *config
		options.debugConfig = &value
	}
}

func NewProviderSetAPI(
	options ...ProviderSetOption,
) (*ProviderSetAPI, error) {
	config := providerSetOptions{
		logger: slog.Default(),
	}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	if config.logger == nil {
		return nil, errors.New("inference requires a logger")
	}

	debugConfig := disabledDebugConfig()
	if config.debugConfig != nil {
		debugConfig = *config.debugConfig
	}
	debugger := debugclient.NewHTTPCompletionDebugger(&debugConfig)

	inner, err := inferenceGo.NewProviderSetAPI(
		inferenceGo.WithLogger(config.logger),
		inferenceGo.WithDebugClientBuilder(
			func(inferenceSpec.ProviderParam) inferenceSpec.CompletionDebugger {
				return debugger
			},
		),
	)
	if err != nil {
		return nil, err
	}

	return &ProviderSetAPI{
		inner:    inner,
		logger:   config.logger,
		debugger: debugger,
	}, nil
}

func (ps *ProviderSetAPI) SetDebugConfig(config *debugclient.DebugConfig) {
	next := disabledDebugConfig()
	if config != nil {
		next = *config
	}
	ps.debugger.SetConfig(next)
}

func (ps *ProviderSetAPI) GetDebugConfig() *debugclient.DebugConfig {
	config := ps.debugger.GetConfig()
	return &config
}

func (ps *ProviderSetAPI) AddProvider(
	ctx context.Context,
	request *AddProviderRequest,
) (*AddProviderResponse, error) {
	if request == nil || request.Body == nil ||
		request.Provider == "" ||
		strings.TrimSpace(request.Body.Origin) == "" {
		return nil, errors.New("invalid provider publication request")
	}

	_, err := ps.inner.AddProvider(
		ctx,
		request.Provider,
		&inferenceGo.AddProviderConfig{
			SDKType:                  request.Body.SDKType,
			Origin:                   request.Body.Origin,
			ChatCompletionPathPrefix: request.Body.ChatCompletionPathPrefix,
			APIKeyHeaderKey:          request.Body.APIKeyHeaderKey,
			DefaultHeaders:           maps.Clone(request.Body.DefaultHeaders),
		},
	)
	if err != nil {
		return nil, err
	}
	return &AddProviderResponse{}, nil
}

func (ps *ProviderSetAPI) DeleteProvider(
	ctx context.Context,
	request *DeleteProviderRequest,
) (*DeleteProviderResponse, error) {
	if request == nil || request.Provider == "" {
		return nil, errors.New("invalid provider removal request")
	}
	if err := ps.inner.DeleteProvider(ctx, request.Provider); err != nil {
		return nil, err
	}
	return &DeleteProviderResponse{}, nil
}

func (ps *ProviderSetAPI) SetProviderAPIKey(
	ctx context.Context,
	request *SetProviderAPIKeyRequest,
) (*SetProviderAPIKeyResponse, error) {
	if request == nil || request.Body == nil || request.Provider == "" {
		return nil, errors.New("invalid provider credential request")
	}
	if err := ps.inner.SetProviderAPIKey(
		ctx,
		request.Provider,
		request.Body.APIKey,
	); err != nil {
		return nil, err
	}
	return &SetProviderAPIKeyResponse{}, nil
}

func (ps *ProviderSetAPI) FetchCompletion(
	ctx context.Context,
	request *CompletionRequest,
) (*CompletionResponse, error) {
	if request == nil {
		return nil, errors.New("prepared completion request is required")
	}
	if err := request.Configuration.Validate(); err != nil {
		return nil, err
	}
	if len(request.Inputs) == 0 {
		return nil, errors.New("prepared completion has no inputs")
	}

	completionKey := uuidutil.NewUUIDv7()
	provider, release, err := ps.registerRuntimeProvider(
		ctx,
		request.Configuration,
		completionKey,
	)
	if err != nil {
		return nil, err
	}
	defer release()

	capabilities, err := ps.newRuntimeCapabilityResolver(
		ctx,
		provider,
		request.Configuration,
		completionKey,
	)
	if err != nil {
		return nil, err
	}

	options := &inferenceSpec.FetchCompletionOptions{
		CompletionKey:      completionKey,
		CapabilityResolver: capabilities,
	}
	if request.OnStreamText != nil || request.OnStreamThinking != nil {
		options.StreamHandler = makeStreamHandler(
			request.OnStreamText,
			request.OnStreamThinking,
		)
		options.StreamConfig = &inferenceSpec.StreamConfig{
			FlushIntervalMillis: defaultFlushIntervalMillis,
			FlushChunkSize:      defaultFlushChunkSize,
		}
	}

	response, executeErr := ps.inner.FetchCompletion(
		ctx,
		provider,
		&inferenceSpec.FetchCompletionRequest{
			ModelParam:  request.Configuration.ModelParam,
			Inputs:      request.Inputs,
			ToolChoices: request.ToolChoices,
		},
		options,
	)

	if response == nil && executeErr == nil {
		response = &inferenceSpec.FetchCompletionResponse{
			Error: &inferenceSpec.Error{
				Code:    emptyCompletionResponseCode,
				Message: "provider finished without a final response payload",
			},
		}
	}
	if response != nil &&
		executeErr == nil &&
		response.Error == nil &&
		len(response.Outputs) == 0 {
		response.Error = &inferenceSpec.Error{
			Code:    emptyCompletionResponseCode,
			Message: "provider finished without final output",
		}
	}

	return &CompletionResponse{
		Body: &CompletionResponseBody{
			InferenceResponse: response,
		},
	}, executeErr
}
