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

	"github.com/flexigpt/flexigpt-app/internal/inferencewrapper"
	inferencewrapperSpec "github.com/flexigpt/flexigpt-app/internal/inferencewrapper/spec"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
	settingStore "github.com/flexigpt/flexigpt-app/internal/setting/store"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

var appSlogLevelVar slog.LevelVar

func init() {
	appSlogLevelVar.Set(slog.LevelInfo)
}

type AggregrateWrapper struct {
	modelPresetStore *modelpresetStore.ModelPresetStore
	settingStore     *settingStore.SettingStore
	toolStore        *toolStore.ToolStore
	skillStore       *skillStore.SkillStore
	providersetAPI   *inferencewrapper.ProviderSetAPI

	appContext          context.Context
	completionCancelMux sync.Mutex
	completionCancels   map[string]context.CancelFunc
	preCanceled         map[string]time.Time
}

func InitAggregrateWrapper(
	agg *AggregrateWrapper,
	mps *modelpresetStore.ModelPresetStore,
	ss *settingStore.SettingStore,
	ts *toolStore.ToolStore,
	skillSt *skillStore.SkillStore,
) error {
	if agg == nil || ts == nil || mps == nil || ss == nil {
		panic("initializing aggregate store wrapper on nil receivers")
	}

	agg.toolStore = ts
	agg.modelPresetStore = mps
	agg.settingStore = ss
	agg.skillStore = skillSt
	defaultDebugConfig := inferencewrapper.DefaultDebugConfig()
	p, err := inferencewrapper.NewProviderSetAPI(
		agg.toolStore,
		agg.modelPresetStore,
		agg.skillStore,
		inferencewrapper.WithLogger(slog.Default()),
		inferencewrapper.WithDebugConfig(&defaultDebugConfig),
	)
	if err != nil {
		return errors.Join(err, errors.New("invalid default provider"))
	}
	agg.providersetAPI = p
	agg.completionCancels = map[string]context.CancelFunc{}
	agg.preCanceled = map[string]time.Time{}

	err = initProviderSetUsingSettingsAndPresets(
		context.Background(),
		agg.modelPresetStore,
		agg.settingStore,
		agg.providersetAPI,
	)
	if err != nil {
		return err
	}

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

func (w *AggregrateWrapper) PostProviderPreset(
	req *modelpresetSpec.PostProviderPresetRequest,
) (*modelpresetSpec.PostProviderPresetResponse, error) {
	return middleware.WithRecoveryResp(func() (*modelpresetSpec.PostProviderPresetResponse, error) {
		// First try to delete from provider apis, it is ok if it is not present.
		_, _ = w.providersetAPI.DeleteProvider(
			context.Background(),
			&inferencewrapperSpec.DeleteProviderRequest{
				Provider: inferenceSpec.ProviderName(string(req.ProviderName)),
			},
		)
		// Then try to add in provider apis, need to skip adding to store if it cannot be added.
		if _, err := w.providersetAPI.AddProvider(
			context.Background(),
			&inferencewrapperSpec.AddProviderRequest{
				Provider: inferenceSpec.ProviderName(string(req.ProviderName)),
				Body: &inferencewrapperSpec.AddProviderRequestBody{
					SDKType:                  req.Body.SDKType,
					Origin:                   req.Body.Origin,
					ChatCompletionPathPrefix: req.Body.ChatCompletionPathPrefix,
					APIKeyHeaderKey:          req.Body.APIKeyHeaderKey,
					DefaultHeaders:           req.Body.DefaultHeaders,
				},
			}); err != nil {
			return nil, err
		}
		resp, err := w.modelPresetStore.PostProviderPreset(context.Background(), req)
		if err != nil {
			return nil, err
		}
		return resp, nil
	})
}

func (w *AggregrateWrapper) DeleteProviderPreset(
	req *modelpresetSpec.DeleteProviderPresetRequest,
) (*modelpresetSpec.DeleteProviderPresetResponse, error) {
	return middleware.WithRecoveryResp(func() (*modelpresetSpec.DeleteProviderPresetResponse, error) {
		_, err := w.DeleteAuthKey(
			&settingSpec.DeleteAuthKeyRequest{
				Type:    settingSpec.AuthKeyTypeProvider,
				KeyName: settingSpec.AuthKeyName(req.ProviderName),
			},
		)
		if err != nil {
			return nil, err
		}
		_, _ = w.providersetAPI.DeleteProvider(
			context.Background(),
			&inferencewrapperSpec.DeleteProviderRequest{
				Provider: inferenceSpec.ProviderName(string(req.ProviderName)),
			},
		)
		resp, err := w.modelPresetStore.DeleteProviderPreset(context.Background(), req)
		if err != nil {
			return nil, err
		}
		return resp, nil
	})
}

func (w *AggregrateWrapper) SetAuthKey(
	req *settingSpec.SetAuthKeyRequest,
) (*settingSpec.SetAuthKeyResponse, error) {
	return middleware.WithRecoveryResp(func() (*settingSpec.SetAuthKeyResponse, error) {
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
	return middleware.WithRecoveryResp(func() (*settingSpec.DeleteAuthKeyResponse, error) {
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
	provider string,
	modelPresetID string,
	completionData *inferencewrapperSpec.CompletionRequestBody,
	textCallbackID string,
	thinkingCallbackID string,
	requestID string,
) (*inferencewrapperSpec.CompletionResponse, error) {
	return middleware.WithRecoveryResp(func() (*inferencewrapperSpec.CompletionResponse, error) {
		if requestID == "" {
			return nil, errors.New("requestID is empty")
		}
		if w.appContext == nil {
			return nil, errors.New("appContext is not set (call SetWrappedProviderAppContext during startup)")
		}
		if w.completionCancels == nil {
			w.completionCancels = map[string]context.CancelFunc{}
		}
		if w.preCanceled == nil {
			w.preCanceled = map[string]time.Time{}
		}

		ctx, cancel := context.WithCancel(w.appContext)
		defer cancel()

		w.completionCancelMux.Lock()
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

		req := &inferencewrapperSpec.CompletionRequest{
			Provider:      inferenceSpec.ProviderName(provider),
			ModelPresetID: modelpresetSpec.ModelPresetID(modelPresetID),
			Body:          completionData,
		}

		if textCallbackID != "" {
			req.OnStreamText = func(textData string) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				//nolint:contextcheck // Need to pass app context here and not new context.
				runtime.EventsEmit(w.appContext, textCallbackID, textData)
				return nil
			}
		}
		if thinkingCallbackID != "" {
			req.OnStreamThinking = func(thinkingData string) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				//nolint:contextcheck // Need to pass app context here and not new context.
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
			// If we have a partial response, attach error info there and return it.
			if resp != nil && resp.Body != nil && resp.Body.InferenceResponse != nil {
				if resp.Body.InferenceResponse.Error == nil {
					resp.Body.InferenceResponse.Error = &inferenceSpec.Error{
						Message: err.Error(),
					}
				}
				// Log, but do not propagate Go error so Wails resolves the Promise.
				slog.Error("fetchCompletion failed", "provider", provider, "err", err)
				return resp, nil
			}
			// No response at all => infrastructure error.
			return nil, err
		}

		return resp, nil
	})
}

func (w *AggregrateWrapper) CancelCompletion(id string) error {
	var err error
	defer func() {
		if r := recover(); r != nil {
			// Log the panic plus stack trace.
			slog.Error("panic recovered",
				slog.Any("panic", r),
				slog.String("stacktrace", string(debug.Stack())),
			)

			// Overwrite err so the caller sees we failed.
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	if id == "" {
		return err
	}
	w.completionCancelMux.Lock()
	defer w.completionCancelMux.Unlock()
	if c, ok := w.completionCancels[id]; ok {
		c()
		delete(w.completionCancels, id)
		return err
	}

	// Cancel arrived before FetchCompletion registered the cancel func.
	w.preCanceled[id] = time.Now().UTC()

	// Best-effort pruning to avoid unbounded growth.
	cutoff := time.Now().UTC().Add(-2 * time.Minute)
	for k, t := range w.preCanceled {
		if t.Before(cutoff) {
			delete(w.preCanceled, k)
		}
	}
	return err
}

func initProviderSetUsingSettingsAndPresets(
	ctx context.Context,
	mpw *modelpresetStore.ModelPresetStore,
	s *settingStore.SettingStore,
	p *inferencewrapper.ProviderSetAPI,
) error {
	allProviders, err := getAllProviderPresets(ctx, mpw)
	if err != nil {
		return err
	}
	keySecrets, err := getAllProviderSecrets(ctx, s)
	if err != nil {
		return err
	}

	if err := initProviders(ctx, p, allProviders, keySecrets); err != nil {
		return err
	}

	slog.Info("initProviderSetUsingSettingsAndPresets completed",
		"authKeys", len(keySecrets))

	return nil
}

func getAllProviderPresets(
	ctx context.Context,
	mpw *modelpresetStore.ModelPresetStore,
) ([]modelpresetSpec.ProviderPreset, error) {
	const maxSafetyHops = 16

	var (
		all   []modelpresetSpec.ProviderPreset
		token string
		hops  int
	)

	for {
		resp, err := mpw.ListProviderPresets(ctx, &modelpresetSpec.ListProviderPresetsRequest{
			IncludeDisabled: true,
			PageSize:        modelpresetSpec.MaxPageSize,
			PageToken:       token,
		})
		if err != nil {
			return nil, err
		}
		if resp.Body == nil {
			break
		}
		all = append(all, resp.Body.Providers...)

		if resp.Body.NextPageToken == nil || *resp.Body.NextPageToken == "" {
			break
		}
		if hops >= maxSafetyHops {
			return nil, fmt.Errorf("pagination exceeded %d hops - aborting", maxSafetyHops)
		}
		token = *resp.Body.NextPageToken
		hops++
	}
	return all, nil
}

// getAllProviderSecrets fetches every secret once and caches them in-mem.
func getAllProviderSecrets(
	ctx context.Context,
	s *settingStore.SettingStore,
) (map[string]string, error) {
	resp, err := s.GetSettings(ctx, &settingSpec.GetSettingsRequest{})
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, errors.New("GetSettings: empty response body")
	}

	secrets := make(map[string]string, len(resp.Body.AuthKeys))
	for _, meta := range resp.Body.AuthKeys {
		if meta.Type != settingSpec.AuthKeyTypeProvider {
			continue
		}
		secResp, err := s.GetAuthKey(ctx, &settingSpec.GetAuthKeyRequest{
			Type:    meta.Type,
			KeyName: meta.KeyName,
		})
		if err != nil {
			return nil, err
		}
		if secResp.Body != nil && secResp.Body.Secret != "" {
			secrets[string(meta.KeyName)] = secResp.Body.Secret
		}
	}
	return secrets, nil
}

// BuildAddProviderRequests merges presets + secrets.
// Only providers that have a (valid) preset are considered.  If a matching
// secret exists its value is copied into the request.
func initProviders(
	ctx context.Context,
	providerAPI *inferencewrapper.ProviderSetAPI,
	providers []modelpresetSpec.ProviderPreset,
	secrets map[string]string,
) error {
	providersAdded := 0
	providersWithAPIKey := 0
	for _, pp := range providers {
		if pp.Name == "" || pp.Origin == "" {
			slog.Warn("skipping provider with invalid preset", "name", pp.Name)
			continue
		}

		body := &inferencewrapperSpec.AddProviderRequestBody{
			SDKType:                  pp.SDKType,
			Origin:                   pp.Origin,
			ChatCompletionPathPrefix: pp.ChatCompletionPathPrefix,
			APIKeyHeaderKey:          pp.APIKeyHeaderKey,
			DefaultHeaders:           pp.DefaultHeaders,
		}
		r := &inferencewrapperSpec.AddProviderRequest{
			Provider: inferenceSpec.ProviderName(string(pp.Name)),
			Body:     body,
		}
		if _, err := providerAPI.AddProvider(ctx, r); err != nil {
			return fmt.Errorf("add provider failed. name: %s, err: %w ", pp.Name, err)
		}
		providersAdded++
		if secret, ok := secrets[string(pp.Name)]; ok {
			_, err := providerAPI.SetProviderAPIKey(ctx, &inferencewrapperSpec.SetProviderAPIKeyRequest{
				Provider: inferenceSpec.ProviderName(string(pp.Name)),
				Body: &inferencewrapperSpec.SetProviderAPIKeyRequestBody{
					APIKey: secret,
				},
			})
			if err != nil {
				return fmt.Errorf("set provider api key failed. name: %s, err: %w ", pp.Name, err)
			}
			providersWithAPIKey++
		}
	}

	if providersAdded == 0 {
		slog.Warn("no providers found - nothing to initialize")
	}
	if providersWithAPIKey == 0 {
		slog.Warn("no providers with APIKey")
	}

	return nil
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
