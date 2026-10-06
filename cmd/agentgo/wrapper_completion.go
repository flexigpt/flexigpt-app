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
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/connection"
	modelAggregate "github.com/flexigpt/flexigpt-app/internal/model/aggregate"
	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
	settingStore "github.com/flexigpt/flexigpt-app/internal/setting/store"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
)

var appSlogLevelVar slog.LevelVar

const preCanceledRetention = 2 * time.Minute

func init() {
	appSlogLevelVar.Set(slog.LevelInfo)
}

type CompletionWrapper struct {
	modelAggregate *modelAggregate.Service
	settingStore   *settingStore.SettingStore
	toolService    *toolAPI.Service
	artifactSkills *skillAggregate.Service
	providersetAPI *inferencewrapper.ProviderSetAPI

	appContext          context.Context
	completionCancelMux sync.Mutex
	completionCancels   map[string]context.CancelFunc
	preCanceled         map[string]time.Time
}

type CompletionRequestBody struct {
	// Past turns of the conversation, already persisted.
	History []conversationSpec.ConversationMessage `json:"history"`

	// New user turn to complete. Must have Role=user. Typically will have:
	//   - Attachments (ref attachments),
	//   - either:
	//       * pre-built InputUnion(s) in Inputs, or
	//       * just Messages + Attachments and let the aggregator build
	//         the InputUnion for this turn.
	Current conversationSpec.ConversationMessage `json:"current"`

	// After all Provider and Model source and overlay layers. A nil value means
	// that this completion has no request-level runtime override.
	RequestPatch *modelAggregate.RuntimeRequestPatch `json:"requestPatch,omitempty"`

	// ToolSelections is the set of mapped Go or SDK Tool targets that should
	// be enabled for this call.
	//
	// The inference aggregate hydrates Go Tools into local function choices
	// and SDK Tools into provider-native choices. It does not infer tools from
	// History[i].ToolChoices or Current.ToolChoices, and SDK Tools must never
	// be passed to ToolRuntimeWrapper.
	// (Those are persisted for UI/analytics only.)
	ToolSelections []conversationSpec.ToolSelection `json:"toolSelections,omitempty"`

	MCPContext     *conversationSpec.MCPConversationContext `json:"mcpContext,omitempty"`
	SkillSessionID string                                   `json:"skillSessionID,omitempty"`
}

func InitCompletionWrapper(
	agg *CompletionWrapper,
	models *modelAggregate.Service,
	ss *settingStore.SettingStore,
	ts *toolAPI.Service,
	artifactSkills *skillAggregate.Service,
	mr *mcpConnection.MCPRuntimeManager,
	workspaceSource inferencewrapper.WorkspaceSource,
) error {
	if agg == nil || ts == nil || models == nil || ss == nil || artifactSkills == nil || workspaceSource == nil {
		panic("initializing aggregate store wrapper on nil receivers")
	}

	agg.toolService = ts
	agg.modelAggregate = models
	agg.settingStore = ss
	agg.artifactSkills = artifactSkills

	defaultDebugConfig := inferencewrapper.DefaultDebugConfig()

	var bridge *inferencewrapper.MCPInferenceBridge
	if mr != nil {
		bridge = inferencewrapper.NewMCPInferenceBridge(mr)
	}

	p, err := inferencewrapper.NewProviderSetAPI(
		agg.toolService,
		agg.artifactSkills,
		bridge,
		workspaceSource,
		inferencewrapper.WithLogger(slog.Default()),
		inferencewrapper.WithDebugConfig(&defaultDebugConfig),
		inferencewrapper.WithSkillsRunScriptEnabled(artifactSkills.RunScriptsEnabled()),
	)
	if err != nil {
		return errors.Join(err, errors.New("invalid default provider"))
	}
	agg.providersetAPI = p
	agg.completionCancels = map[string]context.CancelFunc{}
	agg.preCanceled = map[string]time.Time{}

	agg.settingStore.SetDebugSettingsApplier(func(_ context.Context, cfg settingSpec.DebugSettings) error {
		return applyDebugSettings(agg.providersetAPI, cfg)
	})
	if err := agg.settingStore.ApplyCurrentDebugSettings(context.Background(), true); err != nil {
		slog.Error("couldn't apply persisted debug settings", "error", err)
		return err
	}
	return nil
}

func SetWrappedProviderAppContext(w *CompletionWrapper, ctx context.Context) {
	w.appContext = ctx
}

// FetchCompletion handles the completion request and streams data back to the frontend.
func (w *CompletionWrapper) FetchCompletion(
	model artifactModel.ArtifactRef,
	completionData *CompletionRequestBody,
	textCallbackID string,
	thinkingCallbackID string,
	requestID string,
) (*inferencewrapperSpec.CompletionResponse, error) {
	return withRecoveryResp(func() (*inferencewrapperSpec.CompletionResponse, error) {
		if requestID == "" {
			return nil, errors.New("requestID is empty")
		}
		if w.appContext == nil {
			return nil, errors.New("appContext is not set (call SetWrappedProviderAppContext during startup)")
		}
		if completionData == nil {
			return nil, errors.New("completionData is nil")
		}

		ctx, cancel := context.WithCancel(w.appContext)
		defer cancel()

		w.completionCancelMux.Lock()
		w.ensureCompletionStateLocked()
		w.prunePreCanceledLocked(time.Now().UTC())

		// If a cancel arrived before the fetch registered, honor it.
		if _, ok := w.preCanceled[requestID]; ok {
			delete(w.preCanceled, requestID)
			w.completionCancelMux.Unlock()
			return nil, context.Canceled
		}
		// Protect against requestID reuse while in-flight.
		if _, exists := w.completionCancels[requestID]; exists {
			w.completionCancelMux.Unlock()
			return nil, errors.New("duplicate requestID: a completion with this id is already in flight")
		}

		w.completionCancels[requestID] = cancel
		w.completionCancelMux.Unlock()

		defer func() {
			w.completionCancelMux.Lock()
			delete(w.completionCancels, requestID)
			w.completionCancelMux.Unlock()
		}()

		runtimeModel, err := w.modelAggregate.ResolveRuntimeConfiguration(
			ctx,
			modelAggregate.RuntimeModelRequest{
				Model:        model,
				RequestPatch: completionData.RequestPatch,
			},
		)
		if err != nil {
			return nil, err
		}

		req := &inferencewrapperSpec.CompletionRequest{
			Runtime: &inferencewrapperSpec.RuntimeModel{
				ProviderParam:            runtimeModel.ProviderParam,
				ModelParam:               runtimeModel.ModelParam,
				CapabilityOverrides:      runtimeModel.CapabilityOverrides,
				ConfigurationFingerprint: runtimeModel.Fingerprint,
			},
			History:        completionData.History,
			Current:        completionData.Current,
			ToolSelections: completionData.ToolSelections,
			MCPContext:     completionData.MCPContext,
			SkillSessionID: completionData.SkillSessionID,
		}

		if textCallbackID != "" {
			req.OnStreamText = func(textData string) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				if textData == "" {
					// Empty deltas are valid no-ops. They do not indicate that
					// the stream has finished or failed.
					return nil
				}

				runtime.EventsEmit(w.appContext, textCallbackID, textData)
				return nil
			}
		}
		if thinkingCallbackID != "" {
			req.OnStreamThinking = func(thinkingData string) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				if thinkingData == "" {
					return nil
				}

				runtime.EventsEmit(w.appContext, thinkingCallbackID, thinkingData)
				return nil
			}
		}
		resp, err := w.providersetAPI.FetchCompletion(
			ctx,
			req,
		)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				// Expected lifecycle event; return partial resp if present without noisy error logging.
				if resp != nil {
					return resp, nil
				}
				return nil, err
			}
			// Preserve hydration metadata even when the provider failed before
			// producing an inference response. In particular, Workspace Context
			// hydration must not disappear from the persisted user message.
			if resp != nil && resp.Body != nil {
				if resp.Body.InferenceResponse == nil {
					resp.Body.InferenceResponse = &inferenceSpec.FetchCompletionResponse{
						Error: &inferenceSpec.Error{
							Code:    "completion_failed",
							Message: err.Error(),
						},
					}
				} else if resp.Body.InferenceResponse.Error == nil {
					resp.Body.InferenceResponse.Error = &inferenceSpec.Error{
						Code:    "completion_failed",
						Message: err.Error(),
					}
				}
				slog.Error("fetchCompletion failed", "model", model, "err", err)
				return resp, nil
			}
			// No response at all => infrastructure error.
			return nil, err
		}

		return resp, nil
	})
}

func (w *CompletionWrapper) CancelCompletion(id string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic recovered",
				slog.Any("panic", r),
				slog.String("stacktrace", string(debug.Stack())),
			)
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	if id == "" {
		return nil
	}

	w.completionCancelMux.Lock()
	defer w.completionCancelMux.Unlock()

	w.ensureCompletionStateLocked()
	w.prunePreCanceledLocked(time.Now().UTC())

	if c, ok := w.completionCancels[id]; ok {
		c()
		// Keep the entry until FetchCompletion exits. This prevents request-ID
		// reuse from registering another request while the original one is still
		// unwinding after cancellation.
		return nil
	}

	// Cancel arrived before FetchCompletion registered the cancel func.
	w.preCanceled[id] = time.Now().UTC()
	return nil
}

func (w *CompletionWrapper) ensureCompletionStateLocked() {
	if w.completionCancels == nil {
		w.completionCancels = map[string]context.CancelFunc{}
	}
	if w.preCanceled == nil {
		w.preCanceled = map[string]time.Time{}
	}
}

func (w *CompletionWrapper) prunePreCanceledLocked(now time.Time) {
	cutoff := now.Add(-preCanceledRetention)
	for requestID, canceledAt := range w.preCanceled {
		if canceledAt.Before(cutoff) {
			delete(w.preCanceled, requestID)
		}
	}
}

func applyDebugSettings(providerSet *inferencewrapper.ProviderSetAPI, cfg settingSpec.DebugSettings) error {
	appSlogLevelVar.Set(toSlogLevel(cfg.LogLevel))
	if providerSet != nil {
		clone := providerSet.GetDebugConfig()
		if clone != nil {
			clone.LogToSlog = cfg.LogLLMReqResp
			clone.DisableContentStripping = cfg.DisableContentStripping
			providerSet.SetDebugConfig(clone)
		}
	}

	slog.Info(
		"applied debug settings",
		"logLLMReqResp", cfg.LogLLMReqResp,
		"disableContentStripping", cfg.DisableContentStripping,
		"logLevel", cfg.LogLevel,
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
