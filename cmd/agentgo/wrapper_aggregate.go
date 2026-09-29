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
	"github.com/flexigpt/llmtools-go/texttool"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/inferencewrapper"
	inferencewrapperSpec "github.com/flexigpt/flexigpt-app/internal/inferencewrapper/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/connection"
	modelAggregate "github.com/flexigpt/flexigpt-app/internal/model/aggregate"
	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
	settingStore "github.com/flexigpt/flexigpt-app/internal/setting/store"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
	toolAggregate "github.com/flexigpt/flexigpt-app/internal/tool/aggregate"
	workspaceConversation "github.com/flexigpt/flexigpt-app/internal/workspace/conversation"
)

var appSlogLevelVar slog.LevelVar

const preCanceledRetention = 2 * time.Minute

func init() {
	appSlogLevelVar.Set(slog.LevelInfo)
}

type AggregrateWrapper struct {
	modelAggregate       *modelAggregate.Service
	settingStore         *settingStore.SettingStore
	toolAggregateService *toolAggregate.Service
	artifactSkills       *skillAggregate.Service
	providersetAPI       *inferencewrapper.ProviderSetAPI

	appContext          context.Context
	completionCancelMux sync.Mutex
	completionCancels   map[string]context.CancelFunc
	preCanceled         map[string]time.Time
}

func InitAggregrateWrapper(
	agg *AggregrateWrapper,
	models *modelAggregate.Service,
	ss *settingStore.SettingStore,
	ts *toolAggregate.Service,
	artifactSkills *skillAggregate.Service,
	mr *mcpConnection.MCPRuntimeManager,
	workspaceAPI workspaceConversation.WorkspaceSource,
) error {
	if agg == nil || ts == nil || models == nil || ss == nil || artifactSkills == nil || workspaceAPI == nil {
		panic("initializing aggregate store wrapper on nil receivers")
	}

	agg.toolAggregateService = ts
	agg.modelAggregate = models
	agg.settingStore = ss
	agg.artifactSkills = artifactSkills

	defaultDebugConfig := inferencewrapper.DefaultDebugConfig()

	var bridge *inferencewrapper.MCPInferenceBridge
	if mr != nil {
		bridge = inferencewrapper.NewMCPInferenceBridge(mr)
	}

	cr, err := workspaceConversation.NewConversationResolver(workspaceAPI)
	if err != nil {
		panic("no workspace api provided")
	}
	workspaceBridge := inferencewrapper.NewWorkspaceInferenceBridge(
		cr,
	)

	p, err := inferencewrapper.NewProviderSetAPI(
		agg.toolAggregateService,
		agg.artifactSkills,
		bridge,
		workspaceBridge,
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

func SetWrappedProviderAppContext(w *AggregrateWrapper, ctx context.Context) {
	w.appContext = ctx
}

func (w *AggregrateWrapper) ApplyUnifiedDiff(
	req *texttool.ApplyUnifiedDiffArgs,
) (*texttool.ApplyUnifiedDiffOut, error) {
	return withRecoveryResp(func() (*texttool.ApplyUnifiedDiffOut, error) {
		if req == nil {
			return nil, errors.New("invalid arguments: nil request received")
		}
		return llmtoolsutil.ApplyUnifiedDiff(context.Background(), *req)
	})
}

func (w *AggregrateWrapper) SetAuthKey(
	req *settingSpec.SetAuthKeyRequest,
) (*settingSpec.SetAuthKeyResponse, error) {
	return withRecoveryResp(func() (*settingSpec.SetAuthKeyResponse, error) {
		if req.Type == settingSpec.AuthKeyTypeProvider {
			_, err := w.providersetAPI.SetProviderAPIKey(
				context.Background(),
				&inferencewrapperSpec.SetProviderAPIKeyRequest{
					Provider: inferenceSpec.ProviderName(req.KeyName),
					Body:     &inferencewrapperSpec.SetProviderAPIKeyRequestBody{APIKey: req.Body.Secret},
				},
			)
			if err != nil {
				return nil, err
			}
		}
		resp, err := w.settingStore.SetAuthKey(context.Background(), req)
		if err != nil {
			return nil, err
		}
		return resp, nil
	})
}

func (w *AggregrateWrapper) DeleteAuthKey(
	req *settingSpec.DeleteAuthKeyRequest,
) (*settingSpec.DeleteAuthKeyResponse, error) {
	return withRecoveryResp(func() (*settingSpec.DeleteAuthKeyResponse, error) {
		resp, err := w.settingStore.DeleteAuthKey(context.Background(), req)
		if err != nil {
			return nil, err
		}
		if req.Type == settingSpec.AuthKeyTypeProvider {
			_, _ = w.providersetAPI.SetProviderAPIKey(
				context.Background(),
				&inferencewrapperSpec.SetProviderAPIKeyRequest{
					Provider: inferenceSpec.ProviderName(req.KeyName),
					Body:     &inferencewrapperSpec.SetProviderAPIKeyRequestBody{APIKey: ""},
				},
			)
		}
		return resp, nil
	})
}

// FetchCompletion handles the completion request and streams data back to the frontend.
func (w *AggregrateWrapper) FetchCompletion(
	model artifact.ArtifactRef,
	completionData *inferencewrapperSpec.CompletionRequestBody,
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

		runtimeModel, err := w.modelAggregate.ResolveRuntimeModel(
			ctx,
			model,
		)
		if err != nil {
			return nil, err
		}

		req := &inferencewrapperSpec.CompletionRequest{
			Runtime: &inferencewrapperSpec.RuntimeModel{
				ProviderParam:            runtimeModel.ProviderParam,
				ModelParam:               runtimeModel.ModelParam,
				Capabilities:             runtimeModel.Capabilities,
				ConfigurationFingerprint: runtimeModel.Fingerprint,
			},
			Body: completionData,
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

func (w *AggregrateWrapper) CancelCompletion(id string) (err error) {
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

func (w *AggregrateWrapper) ensureCompletionStateLocked() {
	if w.completionCancels == nil {
		w.completionCancels = map[string]context.CancelFunc{}
	}
	if w.preCanceled == nil {
		w.preCanceled = map[string]time.Time{}
	}
}

func (w *AggregrateWrapper) prunePreCanceledLocked(now time.Time) {
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
