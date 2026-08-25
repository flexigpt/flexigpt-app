package store

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/flexigpt/inference-go/capabilityoverride"
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
)

var defaultBuiltInProvider = modelpreset.ProviderOpenAIResponses

var builtInProviderTimestamps = map[inferenceSpec.ProviderName]time.Time{
	modelpreset.ProviderAnthropic:       time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderDeepSeek:        time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderLocalAI:         time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderLMStudio:        time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderGoogleGemini:    time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderHuggingFace:     time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderLlamaCPP:        time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderMeta:            time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderMiniMax:         time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderMistral:         time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderMoonshot:        time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderOllama:          time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderOpenAIChat:      time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderOpenAIResponses: time.Date(2025, 9, 25, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderOpenRouter:      time.Date(2025, 9, 25, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderQwen:            time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderSGLang:          time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderVLLM:            time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderXAI:             time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderXiaomi:          time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderZAI:             time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderZAICodingPlan:   time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
}

var builtInDefaultModelPresetIDs = map[inferenceSpec.ProviderName]spec.ModelPresetID{
	modelpreset.ProviderAnthropic:       spec.ModelPresetID(modelpreset.PresetClaudeSonnet5),
	modelpreset.ProviderDeepSeek:        spec.ModelPresetID(modelpreset.PresetDeepSeekV4Flash),
	modelpreset.ProviderLocalAI:         spec.ModelPresetID(modelpreset.PresetGemma426BA4B),
	modelpreset.ProviderLMStudio:        spec.ModelPresetID(modelpreset.PresetQwen3627B),
	modelpreset.ProviderGoogleGemini:    spec.ModelPresetID(modelpreset.PresetGemini37Flash),
	modelpreset.ProviderHuggingFace:     spec.ModelPresetID(modelpreset.PresetGLM52FireworksAI),
	modelpreset.ProviderLlamaCPP:        spec.ModelPresetID(modelpreset.PresetQwen3635BA3B),
	modelpreset.ProviderMeta:            spec.ModelPresetID(modelpreset.PresetMuseSpark12),
	modelpreset.ProviderMiniMax:         spec.ModelPresetID(modelpreset.PresetMiniMaxM3),
	modelpreset.ProviderMistral:         spec.ModelPresetID(modelpreset.PresetMistralMedium35),
	modelpreset.ProviderMoonshot:        spec.ModelPresetID(modelpreset.PresetMoonshotKimiK3),
	modelpreset.ProviderOllama:          spec.ModelPresetID(modelpreset.PresetQwen3635B),
	modelpreset.ProviderOpenAIChat:      spec.ModelPresetID(modelpreset.PresetGPT41),
	modelpreset.ProviderOpenAIResponses: spec.ModelPresetID(modelpreset.PresetGPT56Terra),
	modelpreset.ProviderOpenRouter:      spec.ModelPresetID(modelpreset.PresetDeepSeekV4Flash),
	modelpreset.ProviderQwen:            spec.ModelPresetID(modelpreset.PresetQwen38Max),
	modelpreset.ProviderSGLang:          spec.ModelPresetID(modelpreset.PresetDeepSeekR18B),
	modelpreset.ProviderVLLM:            spec.ModelPresetID(modelpreset.PresetQwen3VL30BA3B),
	modelpreset.ProviderXAI:             spec.ModelPresetID(modelpreset.PresetGrok46),
	modelpreset.ProviderXiaomi:          spec.ModelPresetID(modelpreset.PresetMiMoV25Pro),
	modelpreset.ProviderZAI:             spec.ModelPresetID(modelpreset.PresetGLM53),
	modelpreset.ProviderZAICodingPlan:   spec.ModelPresetID(modelpreset.PresetGLM53),
}

var builtInDisabledModelPresetIDs = map[inferenceSpec.ProviderName]map[spec.ModelPresetID]struct{}{
	modelpreset.ProviderAnthropic: {
		spec.ModelPresetID(modelpreset.PresetClaudeOpus45):   {},
		spec.ModelPresetID(modelpreset.PresetClaudeOpus41):   {},
		spec.ModelPresetID(modelpreset.PresetClaudeSonnet45): {},
		spec.ModelPresetID(modelpreset.PresetClaudeSonnet4):  {},
	},
	modelpreset.ProviderGoogleGemini: {
		spec.ModelPresetID(modelpreset.PresetGemini36Flash): {},
		spec.ModelPresetID(modelpreset.PresetGemini35Flash): {},

		spec.ModelPresetID(modelpreset.PresetGemini31Pro):       {},
		spec.ModelPresetID(modelpreset.PresetGemini31FlashLite): {},

		spec.ModelPresetID(modelpreset.PresetGemini3Flash):      {},
		spec.ModelPresetID(modelpreset.PresetGemini25Flash):     {},
		spec.ModelPresetID(modelpreset.PresetGemini25FlashLite): {},
	},
	modelpreset.ProviderOpenAIChat: {
		spec.ModelPresetID(modelpreset.PresetGPT41Mini): {},
		spec.ModelPresetID(modelpreset.PresetGPT4o):     {},
		spec.ModelPresetID(modelpreset.PresetGPT4oMini): {},
	},
	modelpreset.ProviderOpenAIResponses: {
		spec.ModelPresetID(modelpreset.PresetGPT54Mini):     {},
		spec.ModelPresetID(modelpreset.PresetGPT54):         {},
		spec.ModelPresetID(modelpreset.PresetGPT54Nano):     {},
		spec.ModelPresetID(modelpreset.PresetGPT53Codex):    {},
		spec.ModelPresetID(modelpreset.PresetGPT52):         {},
		spec.ModelPresetID(modelpreset.PresetGPT52Codex):    {},
		spec.ModelPresetID(modelpreset.PresetGPT51):         {},
		spec.ModelPresetID(modelpreset.PresetGPT51Codex):    {},
		spec.ModelPresetID(modelpreset.PresetGPT51CodexMax): {},
		spec.ModelPresetID(modelpreset.PresetGPT5Mini):      {},
	},
	modelpreset.ProviderXAI: {
		spec.ModelPresetID(modelpreset.PresetGrok45):             {},
		spec.ModelPresetID(modelpreset.PresetGrok43):             {},
		spec.ModelPresetID(modelpreset.PresetBuild01):            {},
		spec.ModelPresetID(modelpreset.PresetGrok42Reasoning):    {},
		spec.ModelPresetID(modelpreset.PresetGrok42NonReasoning): {},
	},
}

var builtInProviderDefaultHeaderOverlays = map[inferenceSpec.ProviderName]map[string]string{
	modelpreset.ProviderOpenRouter: {
		"HTTP-Referer": "https://github.com/flexigpt/flexigpt-app",
		"X-Title":      "FlexiGPT",
	},
}

func (b *BuiltInPresets) populateDataFromInferenceCatalog(ctx context.Context) error {
	catalog := modelpreset.DefaultCatalog()
	if len(catalog.Providers) == 0 {
		return errors.New("inference model preset catalog contains no providers")
	}
	if err := modelpreset.ValidateCatalog(catalog); err != nil {
		return fmt.Errorf("invalid inference model preset catalog: %w", err)
	}
	if err := validateBuiltInOverlayCoverage(catalog); err != nil {
		return fmt.Errorf("invalid built-in model preset overlay configuration: %w", err)
	}

	providers := make(map[inferenceSpec.ProviderName]spec.ProviderPreset, len(catalog.Providers))
	models := make(map[inferenceSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset, len(catalog.Providers))

	for providerName, inferenceProvider := range catalog.Providers {
		ts, err := builtInTimestampForProvider(providerName)
		if err != nil {
			return err
		}

		appModels := make(map[spec.ModelPresetID]spec.ModelPreset, len(inferenceProvider.ModelPresets))
		for _, inferenceModel := range inferenceProvider.ModelPresets {
			appModel := appModelPresetFromInference(providerName, inferenceModel, ts)
			appModels[appModel.ID] = appModel
		}

		defaultModelID, ok := builtInDefaultModelPresetIDs[providerName]
		if !ok || defaultModelID == "" {
			return fmt.Errorf(
				"provider %q has no explicit built-in defaultModelPresetID",
				providerName,
			)
		}
		if _, ok := appModels[defaultModelID]; !ok {
			return fmt.Errorf(
				"provider %q defaultModelPresetID %q not present: %w",
				providerName,
				defaultModelID,
				spec.ErrModelPresetNotFound,
			)
		}

		appProvider := appProviderPresetFromInference(inferenceProvider, appModels, defaultModelID, ts)
		if err := validateProviderPreset(&appProvider); err != nil {
			return err
		}

		providers[providerName] = appProvider
		models[providerName] = appModels
	}

	if _, ok := providers[defaultBuiltInProvider]; !ok {
		return fmt.Errorf("default provider %q not present in inference catalog", defaultBuiltInProvider)
	}

	b.defaultProvider = defaultBuiltInProvider
	b.providers = providers
	b.models = models

	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rebuildSnapshot(ctx)
}

func appProviderPresetFromInference(
	in modelpreset.ProviderPreset,
	models map[spec.ModelPresetID]spec.ModelPreset,
	defaultModelID spec.ModelPresetID,
	ts time.Time,
) spec.ProviderPreset {
	headers := maps.Clone(in.DefaultHeaders)
	if extra := builtInProviderDefaultHeaderOverlays[in.Name]; len(extra) > 0 {
		if headers == nil {
			headers = map[string]string{}
		}
		maps.Copy(headers, extra)
	}

	return spec.ProviderPreset{
		SchemaVersion:            spec.SchemaVersion,
		Name:                     in.Name,
		DisplayName:              spec.ProviderDisplayName(in.DisplayName),
		SDKType:                  in.SDKType,
		IsEnabled:                true,
		CreatedAt:                ts,
		ModifiedAt:               ts,
		IsBuiltIn:                true,
		Origin:                   in.Origin,
		ChatCompletionPathPrefix: in.ChatCompletionPathPrefix,
		APIKeyHeaderKey:          in.APIKeyHeaderKey,
		DefaultHeaders:           headers,
		CapabilitiesOverride:     capabilityoverride.CloneModelCapabilitiesOverride(in.CapabilitiesOverride),
		DefaultModelPresetID:     defaultModelID,
		ModelPresets:             cloneModelPresetMap(models),
	}
}

func appModelPresetFromInference(
	provider inferenceSpec.ProviderName,
	in modelpreset.ModelPreset,
	ts time.Time,
) spec.ModelPreset {
	modelID := spec.ModelPresetID(in.ID)
	modelParam := in.ModelParam

	var stopSequences *[]string
	if len(modelParam.StopSequences) > 0 {
		s := slices.Clone(modelParam.StopSequences)
		stopSequences = &s
	}

	systemPrompt := modelParam.SystemPrompt

	return spec.ModelPreset{
		Stream:                      new(modelParam.Stream),
		MaxPromptLength:             new(modelParam.MaxPromptLength),
		MaxOutputLength:             new(modelParam.MaxOutputLength),
		Temperature:                 cloneFloat64Ptr(modelParam.Temperature),
		Reasoning:                   cloneReasoningParam(modelParam.Reasoning),
		SystemPrompt:                &systemPrompt,
		Timeout:                     new(modelParam.Timeout),
		CacheControl:                cloneCacheControl(modelParam.CacheControl),
		OutputParam:                 cloneOutputParam(modelParam.OutputParam),
		StopSequences:               stopSequences,
		AdditionalParametersRawJSON: cloneStringPtr(modelParam.AdditionalParametersRawJSON),
		CapabilitiesOverride:        capabilityoverride.CloneModelCapabilitiesOverride(in.CapabilitiesOverride),
		SchemaVersion:               spec.SchemaVersion,
		ID:                          modelID,
		Name:                        spec.ModelName(modelParam.Name),
		DisplayName:                 spec.ModelDisplayName(in.DisplayName),
		Slug:                        spec.ModelSlug(modelID),
		IsEnabled:                   builtInModelPresetEnabled(provider, modelID),
		CreatedAt:                   ts,
		ModifiedAt:                  ts,
		IsBuiltIn:                   true,
	}
}

func validateBuiltInOverlayCoverage(catalog modelpreset.Catalog) error {
	var errs []error

	if _, ok := catalog.Providers[defaultBuiltInProvider]; !ok {
		errs = append(errs, fmt.Errorf(
			"default built-in provider %q is absent from the inference catalog",
			defaultBuiltInProvider,
		))
	}

	for providerName, provider := range catalog.Providers {
		if ts, ok := builtInProviderTimestamps[providerName]; !ok {
			errs = append(errs, fmt.Errorf(
				"provider %q is missing a built-in timestamp",
				providerName,
			))
		} else if ts.IsZero() {
			errs = append(errs, fmt.Errorf(
				"provider %q has a zero built-in timestamp",
				providerName,
			))
		}

		defaultModelID, ok := builtInDefaultModelPresetIDs[providerName]
		if !ok || defaultModelID == "" {
			errs = append(errs, fmt.Errorf(
				"provider %q is missing an explicit built-in defaultModelPresetID",
				providerName,
			))
		} else if _, ok := provider.ModelPresets[modelpreset.ModelPresetID(defaultModelID)]; !ok {
			errs = append(errs, fmt.Errorf(
				"provider %q default model %q is absent from the inference catalog",
				providerName,
				defaultModelID,
			))
		}
	}

	for providerName := range builtInProviderTimestamps {
		if _, ok := catalog.Providers[providerName]; !ok {
			errs = append(errs, fmt.Errorf(
				"built-in timestamp references unknown provider %q",
				providerName,
			))
		}
	}

	for providerName := range builtInDefaultModelPresetIDs {
		if _, ok := catalog.Providers[providerName]; !ok {
			errs = append(errs, fmt.Errorf(
				"built-in default model references unknown provider %q",
				providerName,
			))
		}
	}

	for providerName, disabledModels := range builtInDisabledModelPresetIDs {
		provider, ok := catalog.Providers[providerName]
		if !ok {
			errs = append(errs, fmt.Errorf(
				"disabled model overlay references unknown provider %q",
				providerName,
			))
			continue
		}

		for modelID := range disabledModels {
			if _, ok := provider.ModelPresets[modelpreset.ModelPresetID(modelID)]; !ok {
				errs = append(errs, fmt.Errorf(
					"disabled model overlay references unknown model %q/%q",
					providerName,
					modelID,
				))
			}
		}
	}

	for providerName, headers := range builtInProviderDefaultHeaderOverlays {
		if _, ok := catalog.Providers[providerName]; !ok {
			errs = append(errs, fmt.Errorf(
				"default-header overlay references unknown provider %q",
				providerName,
			))
		}
		for key, value := range headers {
			if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
				errs = append(errs, fmt.Errorf(
					"default-header overlay for provider %q contains an empty key or value",
					providerName,
				))
			}
		}
	}

	return errors.Join(errs...)
}

func builtInTimestampForProvider(provider inferenceSpec.ProviderName) (time.Time, error) {
	ts, ok := builtInProviderTimestamps[provider]
	if !ok {
		return time.Time{}, fmt.Errorf(
			"provider %q has no explicit built-in timestamp",
			provider,
		)
	}
	return ts, nil
}

func builtInModelPresetEnabled(
	provider inferenceSpec.ProviderName,
	modelID spec.ModelPresetID,
) bool {
	disabled, ok := builtInDisabledModelPresetIDs[provider]
	if !ok {
		return true
	}
	_, isDisabled := disabled[modelID]
	return !isDisabled
}
