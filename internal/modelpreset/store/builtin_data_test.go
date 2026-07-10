package store

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
)

const (
	corruptedBuiltInTestValue = "corrupted"
	windows                   = "windows"
)

func TestNewBuiltInPresets(t *testing.T) {
	tests := []struct {
		name           string
		setupDir       func(*testing.T) string
		snapshotMaxAge time.Duration
		wantErr        bool
	}{
		{
			name: "happy_path",
			setupDir: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
		},
		{
			name: "zero_snapshot_age_defaults",
			setupDir: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			snapshotMaxAge: 0,
		},
		{
			name: "negative_snapshot_age_defaults",
			setupDir: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			snapshotMaxAge: -1,
		},
		{
			name: "empty_base_dir",
			setupDir: func(t *testing.T) string {
				t.Helper()
				return ""
			},
			wantErr: true,
		},
		{
			name: "base_dir_is_file",
			setupDir: func(t *testing.T) string {
				t.Helper()
				tmp := t.TempDir()
				f := filepath.Join(tmp, "file")
				if err := os.WriteFile(f, []byte("dummy"), 0o600); err != nil {
					t.Fatalf("write temp file: %v", err)
				}
				return f
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			dir := tc.setupDir(t)

			bi, err := NewBuiltInPresets(ctx, dir, tc.snapshotMaxAge)
			t.Cleanup(func() {
				closeBuiltInPresetsForTest(t, bi)
			})

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			providers, models, err := bi.ListBuiltInPresets(ctx)
			if err != nil {
				t.Fatalf("ListBuiltInPresets: %v", err)
			}
			if len(providers) == 0 {
				t.Fatal("expected non-empty provider data")
			}
			if len(models) == 0 {
				t.Fatal("expected non-empty model data")
			}

			if _, err := os.Stat(filepath.Join(dir, spec.ModelPresetsBuiltInOverlayDBFileName)); err != nil {
				t.Fatalf("overlay file missing: %v", err)
			}

			for providerName, provider := range providers {
				if !provider.IsBuiltIn {
					t.Errorf("provider %s not flagged built-in", providerName)
				}
				if provider.SchemaVersion != spec.SchemaVersion {
					t.Errorf("provider %s schemaVersion got %q want %q",
						providerName, provider.SchemaVersion, spec.SchemaVersion)
				}
				if provider.DefaultModelPresetID == "" {
					t.Errorf("provider %s has empty defaultModelPresetID", providerName)
				}
				if _, ok := provider.ModelPresets[provider.DefaultModelPresetID]; !ok {
					t.Errorf("provider %s default model %s missing from provider.ModelPresets",
						providerName, provider.DefaultModelPresetID)
				}
				if _, ok := models[providerName][provider.DefaultModelPresetID]; !ok {
					t.Errorf("provider %s default model %s missing from model view",
						providerName, provider.DefaultModelPresetID)
				}
			}

			for providerName, modelMap := range models {
				for modelID, model := range modelMap {
					if !model.IsBuiltIn {
						t.Errorf("model %s/%s not flagged built-in", providerName, modelID)
					}
					if model.SchemaVersion != spec.SchemaVersion {
						t.Errorf("model %s/%s schemaVersion got %q want %q",
							providerName, modelID, model.SchemaVersion, spec.SchemaVersion)
					}
					if model.ID != modelID {
						t.Errorf("model map key mismatch for %s/%s: model.ID=%s",
							providerName, modelID, model.ID)
					}
					if model.Slug != spec.ModelSlug(modelID) {
						t.Errorf("model %s/%s slug got %q want %q",
							providerName, modelID, model.Slug, modelID)
					}
				}
			}
		})
	}
}

func TestBuiltInPresetsInitializedFromInferenceCatalog(t *testing.T) {
	ctx := t.Context()
	bi, _ := mustNewBuiltInPresets(t, time.Hour)
	defer closeBuiltInPresetsForTest(t, bi)

	catalog := modelpreset.DefaultCatalog()
	providers, models, err := bi.ListBuiltInPresets(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInPresets: %v", err)
	}

	if len(providers) != len(catalog.Providers) {
		t.Fatalf("provider count got %d want %d", len(providers), len(catalog.Providers))
	}

	defaultProvider, err := bi.GetBuiltInDefaultProviderName(ctx)
	if err != nil {
		t.Fatalf("GetBuiltInDefaultProviderName: %v", err)
	}
	if defaultProvider != modelpreset.ProviderOpenAIResponses {
		t.Fatalf("default provider got %q want %q",
			defaultProvider, modelpreset.ProviderOpenAIResponses)
	}

	for providerName, inferenceProvider := range catalog.Providers {
		appProvider, ok := providers[providerName]
		if !ok {
			t.Fatalf("provider %q missing from app built-ins", providerName)
		}

		if appProvider.Name != inferenceProvider.Name {
			t.Errorf("provider %s name got %q want %q",
				providerName, appProvider.Name, inferenceProvider.Name)
		}
		if string(appProvider.DisplayName) != inferenceProvider.DisplayName {
			t.Errorf("provider %s displayName got %q want %q",
				providerName, appProvider.DisplayName, inferenceProvider.DisplayName)
		}
		if appProvider.SDKType != inferenceProvider.SDKType {
			t.Errorf("provider %s sdkType got %q want %q",
				providerName, appProvider.SDKType, inferenceProvider.SDKType)
		}
		if appProvider.Origin != inferenceProvider.Origin {
			t.Errorf("provider %s origin got %q want %q",
				providerName, appProvider.Origin, inferenceProvider.Origin)
		}
		if appProvider.ChatCompletionPathPrefix != inferenceProvider.ChatCompletionPathPrefix {
			t.Errorf("provider %s path got %q want %q",
				providerName,
				appProvider.ChatCompletionPathPrefix,
				inferenceProvider.ChatCompletionPathPrefix,
			)
		}
		if appProvider.APIKeyHeaderKey != inferenceProvider.APIKeyHeaderKey {
			t.Errorf("provider %s apiKeyHeaderKey got %q want %q",
				providerName, appProvider.APIKeyHeaderKey, inferenceProvider.APIKeyHeaderKey)
		}

		for k, v := range inferenceProvider.DefaultHeaders {
			if appProvider.DefaultHeaders[k] != v {
				t.Errorf("provider %s default header %q got %q want %q",
					providerName, k, appProvider.DefaultHeaders[k], v)
			}
		}

		if !reflect.DeepEqual(appProvider.CapabilitiesOverride, inferenceProvider.CapabilitiesOverride) {
			t.Errorf("provider %s capabilities override differs from inference catalog", providerName)
		}

		appModels, ok := models[providerName]
		if !ok {
			t.Fatalf("provider %q missing from model view", providerName)
		}
		if len(appModels) != len(inferenceProvider.ModelPresets) {
			t.Fatalf("provider %s model count got %d want %d",
				providerName, len(appModels), len(inferenceProvider.ModelPresets))
		}

		for inferenceModelID, inferenceModel := range inferenceProvider.ModelPresets {
			appModelID := spec.ModelPresetID(inferenceModelID)
			appModel, ok := appModels[appModelID]
			if !ok {
				t.Fatalf("model %s/%s missing from app built-ins", providerName, inferenceModelID)
			}

			if appModel.ID != appModelID {
				t.Errorf("model %s/%s id got %q want %q",
					providerName, inferenceModelID, appModel.ID, appModelID)
			}
			if inferenceSpec.ModelName(appModel.Name) != inferenceModel.ModelParam.Name {
				t.Errorf("model %s/%s name got %q want %q",
					providerName, inferenceModelID, appModel.Name, inferenceModel.ModelParam.Name)
			}
			if string(appModel.DisplayName) != inferenceModel.DisplayName {
				t.Errorf("model %s/%s displayName got %q want %q",
					providerName, inferenceModelID, appModel.DisplayName, inferenceModel.DisplayName)
			}
			if appModel.Stream == nil || *appModel.Stream != inferenceModel.ModelParam.Stream {
				t.Errorf("model %s/%s stream not copied from inference catalog", providerName, inferenceModelID)
			}
			if appModel.MaxPromptLength == nil ||
				*appModel.MaxPromptLength != inferenceModel.ModelParam.MaxPromptLength {
				t.Errorf("model %s/%s maxPromptLength not copied from inference catalog",
					providerName, inferenceModelID)
			}
			if appModel.MaxOutputLength == nil ||
				*appModel.MaxOutputLength != inferenceModel.ModelParam.MaxOutputLength {
				t.Errorf("model %s/%s maxOutputLength not copied from inference catalog",
					providerName, inferenceModelID)
			}
			if appModel.Timeout == nil || *appModel.Timeout != inferenceModel.ModelParam.Timeout {
				t.Errorf("model %s/%s timeout not copied from inference catalog", providerName, inferenceModelID)
			}
			if !reflect.DeepEqual(appModel.CapabilitiesOverride, inferenceModel.CapabilitiesOverride) {
				t.Errorf("model %s/%s capabilities override differs from inference catalog",
					providerName, inferenceModelID)
			}
		}
	}
}

func TestBuiltInPresetAppOverlays(t *testing.T) {
	ctx := t.Context()
	bi, _ := mustNewBuiltInPresets(t, time.Hour)
	defer closeBuiltInPresetsForTest(t, bi)

	providers, models, err := bi.ListBuiltInPresets(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInPresets: %v", err)
	}

	openAIResponses := providers[modelpreset.ProviderOpenAIResponses]
	if openAIResponses.DefaultModelPresetID != spec.ModelPresetID(modelpreset.PresetGPT56Terra) {
		t.Fatalf("openairesponses default model got %q want %q",
			openAIResponses.DefaultModelPresetID,
			modelpreset.PresetGPT54Mini,
		)
	}

	anthropic := providers[modelpreset.ProviderAnthropic]
	if anthropic.DefaultModelPresetID != spec.ModelPresetID(modelpreset.PresetClaudeSonnet5) {
		t.Fatalf("anthropic default model got %q want %q",
			anthropic.DefaultModelPresetID,
			modelpreset.PresetClaudeSonnet5,
		)
	}

	openRouter := providers[modelpreset.ProviderOpenRouter]
	if openRouter.DefaultHeaders["HTTP-Referer"] == "" {
		t.Fatal("openrouter missing app-specific HTTP-Referer header overlay")
	}
	if openRouter.DefaultHeaders["X-Title"] == "" {
		t.Fatal("openrouter missing app-specific X-Title header overlay")
	}

	disabledChecks := []struct {
		provider inferenceSpec.ProviderName
		modelID  spec.ModelPresetID
	}{
		{
			provider: modelpreset.ProviderAnthropic,
			modelID:  spec.ModelPresetID(modelpreset.PresetClaudeSonnet45),
		},
		{
			provider: modelpreset.ProviderGoogleGemini,
			modelID:  spec.ModelPresetID(modelpreset.PresetGemini3Flash),
		},
		{
			provider: modelpreset.ProviderOpenAIChat,
			modelID:  spec.ModelPresetID(modelpreset.PresetGPT4o),
		},
		{
			provider: modelpreset.ProviderOpenAIResponses,
			modelID:  spec.ModelPresetID(modelpreset.PresetGPT52),
		},
	}

	for _, tc := range disabledChecks {
		model, ok := models[tc.provider][tc.modelID]
		if !ok {
			t.Fatalf("expected model %s/%s to exist", tc.provider, tc.modelID)
		}
		if model.IsEnabled {
			t.Fatalf("model %s/%s should be disabled by app overlay", tc.provider, tc.modelID)
		}
	}
}

func TestSetProviderEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T, *BuiltInPresets) (inferenceSpec.ProviderName, bool)
		wantErr bool
	}{
		{
			name: "toggle_existing_provider",
			setup: func(t *testing.T, bi *BuiltInPresets) (inferenceSpec.ProviderName, bool) {
				t.Helper()
				providers, _, err := bi.ListBuiltInPresets(t.Context())
				if err != nil {
					t.Fatalf("ListBuiltInPresets: %v", err)
				}
				providerName, provider := anyBuiltInProvider(t, providers)
				return providerName, !provider.IsEnabled
			},
		},
		{
			name: nonexistentProviderName,
			setup: func(t *testing.T, _ *BuiltInPresets) (inferenceSpec.ProviderName, bool) {
				t.Helper()
				return inferenceSpec.ProviderName("ghost-provider"), true
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			bi, _ := mustNewBuiltInPresets(t, 0)
			defer closeBuiltInPresetsForTest(t, bi)

			providerName, enabled := tc.setup(t, bi)
			got, err := bi.SetProviderEnabled(ctx, providerName, enabled)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.IsEnabled != enabled {
				t.Fatalf("returned provider enabled got %v want %v", got.IsEnabled, enabled)
			}

			providers, _, err := bi.ListBuiltInPresets(ctx)
			if err != nil {
				t.Fatalf("ListBuiltInPresets: %v", err)
			}
			if providers[providerName].IsEnabled != enabled {
				t.Errorf("snapshot enabled got %v want %v", providers[providerName].IsEnabled, enabled)
			}

			flag, ok, err := bi.providerOverlayFlags.GetFlag(ctx, builtInProviderKey(providerName))
			if err != nil {
				t.Fatalf("providerOverlayFlags.GetFlag: %v", err)
			}
			if !ok || flag.Value != enabled {
				t.Errorf("overlay mismatch: present=%v value=%v want present=true value=%v",
					ok, flag.Value, enabled)
			}
		})
	}
}

func TestSetModelPresetEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T, *BuiltInPresets) (inferenceSpec.ProviderName, spec.ModelPresetID, bool)
		wantErr bool
	}{
		{
			name: "toggle_existing_model",
			setup: func(t *testing.T, bi *BuiltInPresets) (inferenceSpec.ProviderName, spec.ModelPresetID, bool) {
				t.Helper()
				_, models, err := bi.ListBuiltInPresets(t.Context())
				if err != nil {
					t.Fatalf("ListBuiltInPresets: %v", err)
				}
				providerName, modelID, model := anyBuiltInModel(t, models)
				return providerName, modelID, !model.IsEnabled
			},
		},
		{
			name: nonexistentProviderName,
			setup: func(t *testing.T, _ *BuiltInPresets) (inferenceSpec.ProviderName, spec.ModelPresetID, bool) {
				t.Helper()
				return inferenceSpec.ProviderName("ghost-provider"), spec.ModelPresetID("m"), true
			},
			wantErr: true,
		},
		{
			name: "nonexistent_model",
			setup: func(t *testing.T, bi *BuiltInPresets) (inferenceSpec.ProviderName, spec.ModelPresetID, bool) {
				t.Helper()
				providers, _, err := bi.ListBuiltInPresets(t.Context())
				if err != nil {
					t.Fatalf("ListBuiltInPresets: %v", err)
				}
				providerName, _ := anyBuiltInProvider(t, providers)
				return providerName, spec.ModelPresetID("ghost-model"), true
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			bi, _ := mustNewBuiltInPresets(t, 0)
			defer closeBuiltInPresetsForTest(t, bi)

			providerName, modelID, enabled := tc.setup(t, bi)
			got, err := bi.SetModelPresetEnabled(ctx, providerName, modelID, enabled)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.IsEnabled != enabled {
				t.Fatalf("returned model enabled got %v want %v", got.IsEnabled, enabled)
			}

			providers, models, err := bi.ListBuiltInPresets(ctx)
			if err != nil {
				t.Fatalf("ListBuiltInPresets: %v", err)
			}
			if models[providerName][modelID].IsEnabled != enabled {
				t.Errorf("models snapshot enabled got %v want %v",
					models[providerName][modelID].IsEnabled, enabled)
			}
			if providers[providerName].ModelPresets[modelID].IsEnabled != enabled {
				t.Errorf("provider snapshot enabled got %v want %v",
					providers[providerName].ModelPresets[modelID].IsEnabled, enabled)
			}

			flag, ok, err := bi.modelOverlayFlags.GetFlag(ctx, getModelKey(providerName, modelID))
			if err != nil {
				t.Fatalf("modelOverlayFlags.GetFlag: %v", err)
			}
			if !ok || flag.Value != enabled {
				t.Errorf("overlay mismatch: present=%v value=%v want present=true value=%v",
					ok, flag.Value, enabled)
			}
		})
	}
}

func TestSetDefaultModelPreset(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T, *BuiltInPresets) (inferenceSpec.ProviderName, spec.ModelPresetID)
		wantErr bool
	}{
		{
			name: "change_existing_provider",
			setup: func(t *testing.T, bi *BuiltInPresets) (inferenceSpec.ProviderName, spec.ModelPresetID) {
				t.Helper()
				providers, models, err := bi.ListBuiltInPresets(t.Context())
				if err != nil {
					t.Fatalf("ListBuiltInPresets: %v", err)
				}
				return anyProviderWithNonDefaultModel(t, providers, models)
			},
		},
		{
			name: nonexistentProviderName,
			setup: func(t *testing.T, _ *BuiltInPresets) (inferenceSpec.ProviderName, spec.ModelPresetID) {
				t.Helper()
				return inferenceSpec.ProviderName("ghost-provider"), spec.ModelPresetID("m1")
			},
			wantErr: true,
		},
		{
			name: "nonexistent_model",
			setup: func(t *testing.T, bi *BuiltInPresets) (inferenceSpec.ProviderName, spec.ModelPresetID) {
				t.Helper()
				providers, _, err := bi.ListBuiltInPresets(t.Context())
				if err != nil {
					t.Fatalf("ListBuiltInPresets: %v", err)
				}
				providerName, _ := anyBuiltInProvider(t, providers)
				return providerName, spec.ModelPresetID("ghost-model")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			bi, _ := mustNewBuiltInPresets(t, 0)
			defer closeBuiltInPresetsForTest(t, bi)

			providerName, modelID := tc.setup(t, bi)
			got, err := bi.SetDefaultModelPreset(ctx, providerName, modelID)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.DefaultModelPresetID != modelID {
				t.Fatalf("returned provider default got %s want %s", got.DefaultModelPresetID, modelID)
			}

			providers, _, err := bi.ListBuiltInPresets(ctx)
			if err != nil {
				t.Fatalf("ListBuiltInPresets: %v", err)
			}
			if providers[providerName].DefaultModelPresetID != modelID {
				t.Errorf("snapshot default got %s want %s",
					providers[providerName].DefaultModelPresetID, modelID)
			}

			flag, ok, err := bi.providerDefaultModelIDOverlayFlags.GetFlag(
				ctx,
				builtInProviderDefaultModelIDKey(providerName),
			)
			if err != nil {
				t.Fatalf("providerDefaultModelIDOverlayFlags.GetFlag: %v", err)
			}
			if !ok || flag.Value != modelID {
				t.Errorf("overlay mismatch: present=%v value=%s want present=true value=%s",
					ok, flag.Value, modelID)
			}
		})
	}
}

func TestListBuiltInPresetsReturnsIndependentCopies(t *testing.T) {
	ctx := t.Context()
	bi, _ := mustNewBuiltInPresets(t, time.Hour)
	defer closeBuiltInPresetsForTest(t, bi)

	providers1, models1, err := bi.ListBuiltInPresets(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInPresets: %v", err)
	}

	for providerName, provider := range providers1 {
		provider.DisplayName = corruptedBuiltInTestValue
		if provider.DefaultHeaders == nil {
			provider.DefaultHeaders = map[string]string{}
		}
		provider.DefaultHeaders["x-corrupted"] = corruptedBuiltInTestValue

		for modelID, model := range provider.ModelPresets {
			model.DisplayName = corruptedBuiltInTestValue
			provider.ModelPresets[modelID] = model
			break
		}

		providers1[providerName] = provider
		break
	}

	for providerName, modelMap := range models1 {
		for modelID, model := range modelMap {
			model.DisplayName = corruptedBuiltInTestValue
			if model.Temperature != nil {
				*model.Temperature = 99
			}
			modelMap[modelID] = model
			models1[providerName] = modelMap
			break
		}
		break
	}

	providers2, models2, err := bi.ListBuiltInPresets(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInPresets second call: %v", err)
	}

	for providerName, provider := range providers2 {
		if provider.DisplayName == corruptedBuiltInTestValue {
			t.Errorf("provider %s shares displayName memory", providerName)
		}
		if provider.DefaultHeaders["x-corrupted"] == corruptedBuiltInTestValue {
			t.Errorf("provider %s shares defaultHeaders memory", providerName)
		}
		for modelID, model := range provider.ModelPresets {
			if model.DisplayName == corruptedBuiltInTestValue {
				t.Errorf("provider model %s/%s shares memory", providerName, modelID)
			}
		}
	}

	for providerName, modelMap := range models2 {
		for modelID, model := range modelMap {
			if model.DisplayName == corruptedBuiltInTestValue {
				t.Errorf("model %s/%s shares displayName memory", providerName, modelID)
			}
			if model.Temperature != nil && *model.Temperature == 99 {
				t.Errorf("model %s/%s shares temperature pointer", providerName, modelID)
			}
		}
	}
}

func TestProviderModelSnapshotConsistency(t *testing.T) {
	ctx := t.Context()
	bi, _ := mustNewBuiltInPresets(t, time.Hour)
	defer closeBuiltInPresetsForTest(t, bi)

	providers, models, err := bi.ListBuiltInPresets(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInPresets: %v", err)
	}

	for providerName, provider := range providers {
		modelMap, ok := models[providerName]
		if !ok {
			t.Fatalf("provider %s missing from model view", providerName)
		}
		if len(provider.ModelPresets) != len(modelMap) {
			t.Fatalf("provider %s model count mismatch: provider=%d models=%d",
				providerName, len(provider.ModelPresets), len(modelMap))
		}
		for modelID, modelFromModelsView := range modelMap {
			modelFromProvider, ok := provider.ModelPresets[modelID]
			if !ok {
				t.Fatalf("provider %s missing model %s from embedded model map", providerName, modelID)
			}
			if !reflect.DeepEqual(modelFromProvider, modelFromModelsView) {
				t.Fatalf("provider %s model %s differs between provider and model views",
					providerName, modelID)
			}
		}
	}

	for providerName := range models {
		if _, ok := providers[providerName]; !ok {
			t.Fatalf("model view provider %s missing from provider view", providerName)
		}
	}
}

func TestRebuildSnapshotAppliesPersistedOverlays(t *testing.T) {
	ctx := t.Context()
	bi, _ := mustNewBuiltInPresets(t, time.Hour)
	defer closeBuiltInPresetsForTest(t, bi)

	providers, models, err := bi.ListBuiltInPresets(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInPresets: %v", err)
	}

	providerName, provider := anyBuiltInProvider(t, providers)
	modelProviderName, modelID, model := anyBuiltInModel(t, models)
	defaultProviderName, newDefaultModelID := anyProviderWithNonDefaultModel(t, providers, models)

	if _, err := bi.providerOverlayFlags.SetFlag(
		ctx,
		builtInProviderKey(providerName),
		!provider.IsEnabled,
	); err != nil {
		t.Fatalf("providerOverlayFlags.SetFlag: %v", err)
	}

	if _, err := bi.modelOverlayFlags.SetFlag(
		ctx,
		getModelKey(modelProviderName, modelID),
		!model.IsEnabled,
	); err != nil {
		t.Fatalf("modelOverlayFlags.SetFlag: %v", err)
	}

	if _, err := bi.providerDefaultModelIDOverlayFlags.SetFlag(
		ctx,
		builtInProviderDefaultModelIDKey(defaultProviderName),
		newDefaultModelID,
	); err != nil {
		t.Fatalf("providerDefaultModelIDOverlayFlags.SetFlag: %v", err)
	}

	bi.mu.Lock()
	err = bi.rebuildSnapshot(ctx)
	bi.mu.Unlock()
	if err != nil {
		t.Fatalf("rebuildSnapshot: %v", err)
	}

	providers2, models2, err := bi.ListBuiltInPresets(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInPresets: %v", err)
	}

	if providers2[providerName].IsEnabled == provider.IsEnabled {
		t.Fatal("provider overlay was not applied")
	}
	if models2[modelProviderName][modelID].IsEnabled == model.IsEnabled {
		t.Fatal("model overlay was not applied")
	}
	if providers2[modelProviderName].ModelPresets[modelID].IsEnabled == model.IsEnabled {
		t.Fatal("model overlay was not applied to provider embedded model map")
	}
	if providers2[defaultProviderName].DefaultModelPresetID != newDefaultModelID {
		t.Fatalf("default model overlay got %s want %s",
			providers2[defaultProviderName].DefaultModelPresetID, newDefaultModelID)
	}
}

func TestBuiltInPresetsOverlayPersistsAcrossReopen(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()

	bi1, err := NewBuiltInPresets(ctx, dir, time.Hour)
	if err != nil {
		t.Fatalf("NewBuiltInPresets first open: %v", err)
	}

	providerName := modelpreset.ProviderOpenAIResponses
	modelID := spec.ModelPresetID(modelpreset.PresetGPT54Mini)

	providers, models, err := bi1.ListBuiltInPresets(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInPresets: %v", err)
	}
	if _, ok := providers[providerName]; !ok {
		t.Fatalf("provider %s missing", providerName)
	}
	if _, ok := models[providerName][modelID]; !ok {
		t.Fatalf("model %s/%s missing", providerName, modelID)
	}

	var newDefaultModelID spec.ModelPresetID
	for id := range models[providerName] {
		if id != providers[providerName].DefaultModelPresetID {
			newDefaultModelID = id
			break
		}
	}
	if newDefaultModelID == "" {
		t.Skip("openairesponses has only one model; cannot test default model persistence")
	}

	if _, err := bi1.SetProviderEnabled(ctx, providerName, false); err != nil {
		t.Fatalf("SetProviderEnabled: %v", err)
	}
	if _, err := bi1.SetModelPresetEnabled(ctx, providerName, modelID, false); err != nil {
		t.Fatalf("SetModelPresetEnabled: %v", err)
	}
	if _, err := bi1.SetDefaultModelPreset(ctx, providerName, newDefaultModelID); err != nil {
		t.Fatalf("SetDefaultModelPreset: %v", err)
	}

	closeBuiltInPresetsForTest(t, bi1)

	bi2, err := NewBuiltInPresets(ctx, dir, time.Hour)
	if err != nil {
		t.Fatalf("NewBuiltInPresets second open: %v", err)
	}
	defer closeBuiltInPresetsForTest(t, bi2)

	providers2, models2, err := bi2.ListBuiltInPresets(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInPresets second open: %v", err)
	}

	if providers2[providerName].IsEnabled {
		t.Fatal("provider enabled overlay did not persist")
	}
	if models2[providerName][modelID].IsEnabled {
		t.Fatal("model enabled overlay did not persist")
	}
	if providers2[providerName].ModelPresets[modelID].IsEnabled {
		t.Fatal("model enabled overlay did not persist into provider embedded model map")
	}
	if providers2[providerName].DefaultModelPresetID != newDefaultModelID {
		t.Fatalf("default model overlay got %s want %s",
			providers2[providerName].DefaultModelPresetID, newDefaultModelID)
	}
}

func TestConcurrentBuiltInPresetAccess(t *testing.T) {
	ctx := t.Context()
	bi, _ := mustNewBuiltInPresets(t, 10*time.Millisecond)
	defer closeBuiltInPresetsForTest(t, bi)

	providers, models, err := bi.ListBuiltInPresets(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInPresets: %v", err)
	}

	providerName, provider := anyBuiltInProvider(t, providers)
	modelProviderName, modelID, model := anyBuiltInModel(t, models)

	t.Run("concurrent_reads", func(t *testing.T) {
		var wg sync.WaitGroup

		for range 10 {
			wg.Go(func() {
				for range 100 {
					if _, _, err := bi.ListBuiltInPresets(ctx); err != nil {
						t.Errorf("ListBuiltInPresets: %v", err)
					}
				}
			})
		}

		wg.Wait()
	})

	t.Run("concurrent_writes", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := range 5 {
			wg.Add(2)

			go func(i int) {
				defer wg.Done()
				_, _ = bi.SetProviderEnabled(ctx, providerName, i%2 == 0)
			}(i)

			go func(i int) {
				defer wg.Done()
				_, _ = bi.SetModelPresetEnabled(ctx, modelProviderName, modelID, i%2 == 0)
			}(i)
		}

		wg.Wait()

		_, _, err := bi.ListBuiltInPresets(ctx)
		if err != nil {
			t.Fatalf("ListBuiltInPresets after concurrent writes: %v", err)
		}

		// Restore deterministic final values for easier debugging if subsequent
		// assertions are added later.
		_, _ = bi.SetProviderEnabled(ctx, providerName, provider.IsEnabled)
		_, _ = bi.SetModelPresetEnabled(ctx, modelProviderName, modelID, model.IsEnabled)
	})
}

func TestGetBuiltInProviderAndModelPreset(t *testing.T) {
	ctx := t.Context()
	bi, _ := mustNewBuiltInPresets(t, time.Hour)
	defer closeBuiltInPresetsForTest(t, bi)

	provider, err := bi.GetBuiltInProvider(ctx, modelpreset.ProviderAnthropic)
	if err != nil {
		t.Fatalf("GetBuiltInProvider: %v", err)
	}
	if provider.Name != modelpreset.ProviderAnthropic {
		t.Fatalf("provider name got %q want %q", provider.Name, modelpreset.ProviderAnthropic)
	}

	modelID := spec.ModelPresetID(modelpreset.PresetClaudeSonnet46)
	model, err := bi.GetBuiltInModelPreset(ctx, modelpreset.ProviderAnthropic, modelID)
	if err != nil {
		t.Fatalf("GetBuiltInModelPreset: %v", err)
	}
	if model.ID != modelID {
		t.Fatalf("model id got %q want %q", model.ID, modelID)
	}

	provider.DisplayName = corruptedBuiltInTestValue
	model.DisplayName = corruptedBuiltInTestValue

	provider2, err := bi.GetBuiltInProvider(ctx, modelpreset.ProviderAnthropic)
	if err != nil {
		t.Fatalf("GetBuiltInProvider second call: %v", err)
	}
	if provider2.DisplayName == corruptedBuiltInTestValue {
		t.Fatal("GetBuiltInProvider returned shared provider memory")
	}

	model2, err := bi.GetBuiltInModelPreset(ctx, modelpreset.ProviderAnthropic, modelID)
	if err != nil {
		t.Fatalf("GetBuiltInModelPreset second call: %v", err)
	}
	if model2.DisplayName == corruptedBuiltInTestValue {
		t.Fatal("GetBuiltInModelPreset returned shared model memory")
	}
}

func mustNewBuiltInPresets(t *testing.T, maxSnapshotAge time.Duration) (bi *BuiltInPresets, dir string) {
	t.Helper()

	dir = t.TempDir()
	bi, err := NewBuiltInPresets(t.Context(), dir, maxSnapshotAge)
	if err != nil {
		t.Fatalf("NewBuiltInPresets: %v", err)
	}

	return bi, dir
}

func closeBuiltInPresetsForTest(t *testing.T, bi *BuiltInPresets) {
	t.Helper()

	if bi == nil {
		return
	}

	if err := bi.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if runtime.GOOS == "windows" {
		// SQLite/file-lock cleanup on Windows can lag very slightly after Close.
		time.Sleep(20 * time.Millisecond)
	}
}

func anyBuiltInProvider(
	t *testing.T,
	providers map[inferenceSpec.ProviderName]spec.ProviderPreset,
) (inferenceSpec.ProviderName, spec.ProviderPreset) {
	t.Helper()

	names := make([]inferenceSpec.ProviderName, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		return name, providers[name]
	}

	t.Fatal("no providers available")
	return "", spec.ProviderPreset{}
}

func anyBuiltInModel(
	t *testing.T,
	models map[inferenceSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset,
) (inferenceSpec.ProviderName, spec.ModelPresetID, spec.ModelPreset) {
	t.Helper()

	providerNames := make([]inferenceSpec.ProviderName, 0, len(models))
	for providerName := range models {
		providerNames = append(providerNames, providerName)
	}
	slices.Sort(providerNames)

	for _, providerName := range providerNames {
		modelIDs := make([]spec.ModelPresetID, 0, len(models[providerName]))
		for modelID := range models[providerName] {
			modelIDs = append(modelIDs, modelID)
		}
		slices.Sort(modelIDs)

		for _, modelID := range modelIDs {
			return providerName, modelID, models[providerName][modelID]
		}
	}

	t.Fatal("no models available")
	return "", "", spec.ModelPreset{}
}

func anyProviderWithNonDefaultModel(
	t *testing.T,
	providers map[inferenceSpec.ProviderName]spec.ProviderPreset,
	models map[inferenceSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset,
) (inferenceSpec.ProviderName, spec.ModelPresetID) {
	t.Helper()

	providerNames := make([]inferenceSpec.ProviderName, 0, len(providers))
	for providerName := range providers {
		providerNames = append(providerNames, providerName)
	}
	slices.Sort(providerNames)

	for _, providerName := range providerNames {
		modelMap := models[providerName]
		if len(modelMap) < 2 {
			continue
		}

		modelIDs := make([]spec.ModelPresetID, 0, len(modelMap))
		for modelID := range modelMap {
			modelIDs = append(modelIDs, modelID)
		}
		slices.Sort(modelIDs)

		for _, modelID := range modelIDs {
			if modelID != providers[providerName].DefaultModelPresetID {
				return providerName, modelID
			}
		}
	}

	t.Fatal("no provider with a non-default alternate model available")
	return "", ""
}
