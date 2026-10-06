package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	"github.com/flexigpt/flexigpt-app/internal/inferencewrapper"
	inferencewrapperSpec "github.com/flexigpt/flexigpt-app/internal/inferencewrapper/spec"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/connection"
	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
	settingStore "github.com/flexigpt/flexigpt-app/internal/setting/store"
)

var appSlogLevelVar slog.LevelVar

const preCanceledRetention = 2 * time.Minute

func init() {
	appSlogLevelVar.Set(slog.LevelInfo)
}

type CompletionWrapper struct {
	providersetAPI *inferencewrapper.ProviderSetAPI

	appContext          context.Context
	completionCancelMux sync.Mutex
	completionCancels   map[string]context.CancelFunc
	preCanceled         map[string]time.Time
	completionWG        sync.WaitGroup
	closed              bool
}

type CompletionRequestBody struct {
	History []conversationSpec.ConversationMessage `json:"history"`
	Current conversationSpec.ConversationMessage   `json:"current"`

	// Applied after Provider/Model source and overlay defaults.
	RequestPatch *inferencewrapperSpec.RuntimeRequestPatch `json:"requestPatch,omitempty"`

	// These are current-call selections, not persisted historical ToolChoices.
	ToolSelections []conversationSpec.ToolSelection `json:"toolSelections,omitempty"`

	MCPContext     *conversationSpec.MCPConversationContext `json:"mcpContext,omitempty"`
	SkillSessionID string                                   `json:"skillSessionID,omitempty"`
}

func InitCompletionWrapper(
	w *CompletionWrapper,
	models inferencewrapperSpec.ModelRuntime,
	settings *settingStore.SettingStore,
	tools inferencewrapper.ToolSource,
	skills inferencewrapper.SkillSource,
	mcpRuntime *mcpConnection.MCPRuntimeManager,
	workspaceSource inferencewrapper.WorkspaceSource,
) error {
	if w == nil || settings == nil {
		return errors.New("completion wrapper dependencies are incomplete")
	}

	var bridge *inferencewrapper.MCPInferenceBridge
	if mcpRuntime != nil {
		bridge = inferencewrapper.NewMCPInferenceBridge(mcpRuntime)
	}

	debugConfig := inferencewrapper.DefaultDebugConfig()
	providers, err := inferencewrapper.NewProviderSetAPI(
		models,
		tools,
		skills,
		bridge,
		workspaceSource,
		inferencewrapper.WithLogger(slog.Default()),
		inferencewrapper.WithDebugConfig(&debugConfig),
	)
	if err != nil {
		return fmt.Errorf("initialize completion inference: %w", err)
	}

	settings.SetDebugSettingsApplier(func(
		_ context.Context,
		config settingSpec.DebugSettings,
	) error {
		return applyDebugSettings(providers, config)
	})
	if err := settings.ApplyCurrentDebugSettings(context.Background(), true); err != nil {
		return fmt.Errorf("apply persisted completion debug settings: %w", err)
	}

	w.providersetAPI = providers
	w.completionCancels = make(map[string]context.CancelFunc)
	w.preCanceled = make(map[string]time.Time)
	return nil
}

func SetWrappedProviderAppContext(w *CompletionWrapper, ctx context.Context) {
	w.completionCancelMux.Lock()
	defer w.completionCancelMux.Unlock()

	w.appContext = ctx
}

// FetchCompletion owns Wails request admission, cancellation, and event delivery.
// Model resolution and inference preparation belong to ProviderSetAPI.
func (w *CompletionWrapper) FetchCompletion(
	model artifactModel.ArtifactRef,
	completionData *CompletionRequestBody,
	textCallbackID string,
	thinkingCallbackID string,
	requestID string,
) (*inferencewrapperSpec.CompletionResponse, error) {
	return withRecoveryResp(func() (*inferencewrapperSpec.CompletionResponse, error) {
		if w == nil {
			return nil, errors.New("completion wrapper is unavailable")
		}
		if requestID == "" {
			return nil, errors.New("requestID is empty")
		}
		if completionData == nil {
			return nil, errors.New("completionData is nil")
		}

		w.completionCancelMux.Lock()
		if w.closed || w.providersetAPI == nil || w.appContext == nil {
			w.completionCancelMux.Unlock()
			return nil, errors.New("completion wrapper is not ready")
		}
		w.prunePreCanceledLocked(time.Now().UTC())

		if _, canceled := w.preCanceled[requestID]; canceled {
			delete(w.preCanceled, requestID)
			w.completionCancelMux.Unlock()
			return nil, context.Canceled
		}
		if _, exists := w.completionCancels[requestID]; exists {
			w.completionCancelMux.Unlock()
			return nil, errors.New("duplicate requestID: a completion with this id is already in flight")
		}

		appContext := w.appContext
		ctx, cancel := context.WithCancel(appContext)
		w.completionCancels[requestID] = cancel
		// Admission and shutdown share this mutex, so Add cannot race Wait
		// after shutdown has stopped accepting requests.
		w.completionWG.Add(1)
		w.completionCancelMux.Unlock()

		defer func() {
			cancel()
			w.completionCancelMux.Lock()
			delete(w.completionCancels, requestID)
			w.completionCancelMux.Unlock()
			w.completionWG.Done()
		}()

		request := &inferencewrapperSpec.CompletionRequest{
			Model: inferencewrapperSpec.RuntimeModelRequest{
				Model:        model,
				RequestPatch: completionData.RequestPatch,
			},
			History:        completionData.History,
			Current:        completionData.Current,
			ToolSelections: completionData.ToolSelections,
			MCPContext:     completionData.MCPContext,
			SkillSessionID: completionData.SkillSessionID,
		}

		if textCallbackID != "" {
			request.OnStreamText = func(text string) error {
				// This is a cancellation boundary, not another dependency
				// validation. Do not emit more events after cancellation.
				if err := ctx.Err(); err != nil {
					return err
				}
				runtime.EventsEmit(appContext, textCallbackID, text)
				return nil
			}
		}
		if thinkingCallbackID != "" {
			request.OnStreamThinking = func(thinking string) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				runtime.EventsEmit(appContext, thinkingCallbackID, thinking)
				return nil
			}
		}

		response, err := w.providersetAPI.FetchCompletion(ctx, request)
		if err == nil {
			return response, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			if response != nil {
				return response, nil
			}
			return nil, err
		}

		// Preserve hydration metadata even when inference produced no result.
		if response != nil && response.Body != nil {
			if response.Body.InferenceResponse == nil {
				response.Body.InferenceResponse = &inferenceSpec.FetchCompletionResponse{
					Error: &inferenceSpec.Error{
						Code:    "completion_failed",
						Message: err.Error(),
					},
				}
			} else if response.Body.InferenceResponse.Error == nil {
				response.Body.InferenceResponse.Error = &inferenceSpec.Error{
					Code:    "completion_failed",
					Message: err.Error(),
				}
			}
			slog.Error("fetchCompletion failed", "model", model, "err", err)
			return response, nil
		}
		return nil, err
	})
}

func (w *CompletionWrapper) CancelCompletion(id string) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error(
				"panic recovered",
				slog.Any("panic", recovered),
				slog.String("stacktrace", string(debug.Stack())),
			)
			err = fmt.Errorf("panic recovered: %v", recovered)
		}
	}()

	if id == "" {
		return nil
	}
	if w == nil {
		return errors.New("completion wrapper is unavailable")
	}

	w.completionCancelMux.Lock()
	defer w.completionCancelMux.Unlock()

	if w.closed {
		return nil
	}
	if w.providersetAPI == nil {
		return errors.New("completion wrapper is not ready")
	}

	w.prunePreCanceledLocked(time.Now().UTC())
	if cancel, exists := w.completionCancels[id]; exists {
		cancel()
		// Keep the entry until FetchCompletion exits to prevent ID reuse
		// while the canceled request is still unwinding.
		return nil
	}

	w.preCanceled[id] = time.Now().UTC()
	return nil
}

func (w *CompletionWrapper) prunePreCanceledLocked(now time.Time) {
	cutoff := now.Add(-preCanceledRetention)
	for requestID, canceledAt := range w.preCanceled {
		if canceledAt.Before(cutoff) {
			delete(w.preCanceled, requestID)
		}
	}
}

// close stops admission and drains in-flight work before its Model, Workspace,
// Tool, Skill, MCP, or settings dependencies are closed.
func (w *CompletionWrapper) close() {
	w.completionCancelMux.Lock()
	w.closed = true
	for _, cancel := range w.completionCancels {
		cancel()
	}
	clear(w.preCanceled)
	w.completionCancelMux.Unlock()

	w.completionWG.Wait()
}

func applyDebugSettings(
	providerSet *inferencewrapper.ProviderSetAPI,
	config settingSpec.DebugSettings,
) error {
	appSlogLevelVar.Set(toSlogLevel(config.LogLevel))

	debugConfig := providerSet.GetDebugConfig()
	debugConfig.LogToSlog = config.LogLLMReqResp
	debugConfig.DisableContentStripping = config.DisableContentStripping
	providerSet.SetDebugConfig(debugConfig)

	slog.Info(
		"applied debug settings",
		"logLLMReqResp", config.LogLLMReqResp,
		"disableContentStripping", config.DisableContentStripping,
		"logLevel", config.LogLevel,
	)
	return nil
}

func toSlogLevel(level settingSpec.DebugLogLevel) slog.Level {
	switch level {
	case settingSpec.DebugLogLevelDebug:
		return slog.LevelDebug
	case settingSpec.DebugLogLevelWarn:
		return slog.LevelWarn
	case settingSpec.DebugLogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
