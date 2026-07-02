package store

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/flexigpt/inference-go/capabilityoverride"
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
)

var defaultBuiltInProvider = modelpreset.ProviderOpenAIResponses

var builtInBaseTimestamp = time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC)

var builtInProviderTimestamps = map[inferenceSpec.ProviderName]time.Time{
	modelpreset.ProviderAnthropic:       time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderLocalAI:         time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderLMStudio:        time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderGoogleGemini:    time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderHuggingFace:     time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderLlamaCPP:        time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderMistral:         time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderOllama:          time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderOpenAIChat:      time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderOpenAIResponses: time.Date(2025, 9, 25, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderOpenRouter:      time.Date(2025, 9, 25, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderSGLang:          time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderVLLM:            time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
	modelpreset.ProviderXAI:             time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
}

var builtInDefaultModelPresetIDs = map[inferenceSpec.ProviderName]spec.ModelPresetID{
	modelpreset.ProviderAnthropic:       spec.ModelPresetID(modelpreset.PresetClaudeSonnet5),
	modelpreset.ProviderLocalAI:         spec.ModelPresetID(modelpreset.PresetGemma426BA4B),
	modelpreset.ProviderLMStudio:        spec.ModelPresetID(modelpreset.PresetQwen3627B),
	modelpreset.ProviderGoogleGemini:    spec.ModelPresetID(modelpreset.PresetGemini35Flash),
	modelpreset.ProviderHuggingFace:     spec.ModelPresetID(modelpreset.PresetGLM52FireworksAI),
	modelpreset.ProviderLlamaCPP:        spec.ModelPresetID(modelpreset.PresetQwen3635BA3B),
	modelpreset.ProviderMistral:         spec.ModelPresetID(modelpreset.PresetMistralMedium35),
	modelpreset.ProviderOllama:          spec.ModelPresetID(modelpreset.PresetQwen3635B),
	modelpreset.ProviderOpenAIChat:      spec.ModelPresetID(modelpreset.PresetGPT41),
	modelpreset.ProviderOpenAIResponses: spec.ModelPresetID(modelpreset.PresetGPT54Mini),
	modelpreset.ProviderOpenRouter:      spec.ModelPresetID(modelpreset.PresetDeepSeekV4Flash),
	modelpreset.ProviderSGLang:          spec.ModelPresetID(modelpreset.PresetDeepSeekR18B),
	modelpreset.ProviderVLLM:            spec.ModelPresetID(modelpreset.PresetQwen3VL30BA3B),
	modelpreset.ProviderXAI:             spec.ModelPresetID(modelpreset.PresetGrok43),
}

var builtInDisabledModelPresetIDs = map[inferenceSpec.ProviderName]map[spec.ModelPresetID]struct{}{
	modelpreset.ProviderAnthropic: {
		spec.ModelPresetID(modelpreset.PresetClaudeOpus45):   {},
		spec.ModelPresetID(modelpreset.PresetClaudeOpus41):   {},
		spec.ModelPresetID(modelpreset.PresetClaudeSonnet45): {},
		spec.ModelPresetID(modelpreset.PresetClaudeSonnet4):  {},
	},
	modelpreset.ProviderGoogleGemini: {
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
		spec.ModelPresetID(modelpreset.PresetGPT52):         {},
		spec.ModelPresetID(modelpreset.PresetGPT52Codex):    {},
		spec.ModelPresetID(modelpreset.PresetGPT51):         {},
		spec.ModelPresetID(modelpreset.PresetGPT51Codex):    {},
		spec.ModelPresetID(modelpreset.PresetGPT51CodexMax): {},
		spec.ModelPresetID(modelpreset.PresetGPT5Mini):      {},
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

	providers := make(map[inferenceSpec.ProviderName]spec.ProviderPreset, len(catalog.Providers))
	models := make(map[inferenceSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset, len(catalog.Providers))

	for providerName, inferenceProvider := range catalog.Providers {
		ts := builtInTimestampForProvider(providerName)

		appModels := make(map[spec.ModelPresetID]spec.ModelPreset, len(inferenceProvider.ModelPresets))
		for _, inferenceModel := range inferenceProvider.ModelPresets {
			appModel := appModelPresetFromInference(providerName, inferenceModel, ts)
			appModels[appModel.ID] = appModel
		}

		defaultModelID := builtInDefaultModelPresetIDs[providerName]
		if defaultModelID == "" {
			defaultModelID = firstModelPresetID(appModels)
		}
		if defaultModelID == "" {
			return fmt.Errorf("provider %q has no model presets", providerName)
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
		ModelPresetPatch: spec.ModelPresetPatch{
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
		},
		SchemaVersion: spec.SchemaVersion,
		ID:            modelID,
		Name:          spec.ModelName(modelParam.Name),
		DisplayName:   spec.ModelDisplayName(in.DisplayName),
		Slug:          spec.ModelSlug(modelID),
		IsEnabled:     builtInModelPresetEnabled(provider, modelID),
		CreatedAt:     ts,
		ModifiedAt:    ts,
		IsBuiltIn:     true,
	}
}

func builtInTimestampForProvider(provider inferenceSpec.ProviderName) time.Time {
	if ts, ok := builtInProviderTimestamps[provider]; ok {
		return ts
	}
	return builtInBaseTimestamp
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

func firstModelPresetID(models map[spec.ModelPresetID]spec.ModelPreset) spec.ModelPresetID {
	if len(models) == 0 {
		return ""
	}

	ids := make([]spec.ModelPresetID, 0, len(models))
	for id := range models {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids[0]
}
