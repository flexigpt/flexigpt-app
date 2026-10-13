package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	inferenceRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference"
	"github.com/flexigpt/flexigpt-app/internal/setting"
)

var appSlogLevelVar slog.LevelVar

func init() {
	appSlogLevelVar.Set(slog.LevelInfo)
}

// CompletionWrapper is no longer a Wails completion execution endpoint.
//
// It owns provider lifecycle publication and live debug transport configuration.
// The Session Runtime owns every accepted model execution.
type CompletionWrapper struct {
	mu sync.RWMutex

	providersetAPI *inferenceRuntime.ProviderSetAPI
	closed         bool
}

func InitCompletionWrapper(
	wrapper *CompletionWrapper,
	settings *setting.SettingStore,
) error {
	if wrapper == nil || settings == nil {
		return errors.New("completion wrapper dependencies are incomplete")
	}

	debugConfig := inferenceRuntime.DefaultDebugConfig()
	providers, err := inferenceRuntime.NewProviderSetAPI(
		inferenceRuntime.WithLogger(slog.Default()),
		inferenceRuntime.WithDebugConfig(&debugConfig),
	)
	if err != nil {
		return fmt.Errorf("initialize provider runtime: %w", err)
	}

	settings.SetDebugSettingsApplier(func(
		_ context.Context,
		config setting.DebugSettings,
	) error {
		return applyDebugSettings(providers, config)
	})
	if err := settings.ApplyCurrentDebugSettings(
		context.Background(),
		true,
	); err != nil {
		return fmt.Errorf(
			"apply persisted provider debug settings: %w",
			err,
		)
	}

	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()

	if wrapper.providersetAPI != nil {
		return errors.New("completion wrapper is already initialized")
	}
	wrapper.providersetAPI = providers
	return nil
}

func (w *CompletionWrapper) providerSet() (
	*inferenceRuntime.ProviderSetAPI,
	error,
) {
	if w == nil {
		return nil, errors.New("completion wrapper is unavailable")
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.closed || w.providersetAPI == nil {
		return nil, errors.New("completion wrapper is closed")
	}
	return w.providersetAPI, nil
}

func (w *CompletionWrapper) close() {
	if w == nil {
		return
	}

	w.mu.Lock()
	w.closed = true
	w.providersetAPI = nil
	w.mu.Unlock()
}

func applyDebugSettings(
	providerSet *inferenceRuntime.ProviderSetAPI,
	config setting.DebugSettings,
) error {
	appSlogLevelVar.Set(toSlogLevel(config.LogLevel))

	debugConfig := providerSet.GetDebugConfig()
	debugConfig.LogToSlog = config.LogLLMReqResp
	debugConfig.DisableContentStripping = config.DisableContentStripping
	providerSet.SetDebugConfig(debugConfig)

	slog.Info(
		"applied provider debug settings",
		"logLLMReqResp", config.LogLLMReqResp,
		"disableContentStripping", config.DisableContentStripping,
		"logLevel", config.LogLevel,
	)
	return nil
}

func toSlogLevel(level setting.DebugLogLevel) slog.Level {
	switch level {
	case setting.DebugLogLevelDebug:
		return slog.LevelDebug
	case setting.DebugLogLevelWarn:
		return slog.LevelWarn
	case setting.DebugLogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
