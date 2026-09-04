package builtin

import (
	"time"

	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

const (
	ProviderNameAnthropic       = string(modelpreset.ProviderAnthropic)
	ProviderNameDeepSeek        = string(modelpreset.ProviderDeepSeek)
	ProviderNameLocalAI         = string(modelpreset.ProviderLocalAI)
	ProviderNameLMStudio        = string(modelpreset.ProviderLMStudio)
	ProviderNameGoogleGemini    = string(modelpreset.ProviderGoogleGemini)
	ProviderNameHuggingFace     = string(modelpreset.ProviderHuggingFace)
	ProviderNameLlamaCPP        = string(modelpreset.ProviderLlamaCPP)
	ProviderNameMeta            = string(modelpreset.ProviderMeta)
	ProviderNameMiniMax         = string(modelpreset.ProviderMiniMax)
	ProviderNameMistral         = string(modelpreset.ProviderMistral)
	ProviderNameMoonshot        = string(modelpreset.ProviderMoonshot)
	ProviderNameOllama          = string(modelpreset.ProviderOllama)
	ProviderNameOpenAIChat      = string(modelpreset.ProviderOpenAIChat)
	ProviderNameOpenAIResponses = string(modelpreset.ProviderOpenAIResponses)
	ProviderNameOpenRouter      = string(modelpreset.ProviderOpenRouter)
	ProviderNameQwen            = string(modelpreset.ProviderQwen)
	ProviderNameSGLang          = string(modelpreset.ProviderSGLang)
	ProviderNameVLLM            = string(modelpreset.ProviderVLLM)
	ProviderNameXAI             = string(modelpreset.ProviderXAI)
	ProviderNameXiaomi          = string(modelpreset.ProviderXiaomi)
	ProviderNameZAI             = string(modelpreset.ProviderZAI)
	ProviderNameZAICodingPlan   = string(modelpreset.ProviderZAICodingPlan)
)

var BuiltInProviderNames = []string{
	ProviderNameAnthropic,
	ProviderNameDeepSeek,
	ProviderNameLocalAI,
	ProviderNameLMStudio,
	ProviderNameGoogleGemini,
	ProviderNameHuggingFace,
	ProviderNameLlamaCPP,
	ProviderNameMeta,
	ProviderNameMiniMax,
	ProviderNameMistral,
	ProviderNameMoonshot,
	ProviderNameOllama,
	ProviderNameOpenAIChat,
	ProviderNameOpenAIResponses,
	ProviderNameOpenRouter,
	ProviderNameQwen,
	ProviderNameSGLang,
	ProviderNameVLLM,
	ProviderNameXAI,
	ProviderNameXiaomi,
	ProviderNameZAI,
	ProviderNameZAICodingPlan,
}

var DefaultBuiltInProvider = modelpreset.ProviderOpenAIResponses

var BuiltInProviderTimestamps = map[inferenceSpec.ProviderName]time.Time{
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

var BuiltInDefaultModelPresetIDs = map[inferenceSpec.ProviderName]modelpreset.ModelPresetID{
	modelpreset.ProviderAnthropic:       modelpreset.PresetClaudeSonnet5,
	modelpreset.ProviderDeepSeek:        modelpreset.PresetDeepSeekV4Flash,
	modelpreset.ProviderLocalAI:         modelpreset.PresetGemma426BA4B,
	modelpreset.ProviderLMStudio:        modelpreset.PresetQwen3627B,
	modelpreset.ProviderGoogleGemini:    modelpreset.PresetGemini38Flash,
	modelpreset.ProviderHuggingFace:     modelpreset.PresetGLM52FireworksAI,
	modelpreset.ProviderLlamaCPP:        modelpreset.PresetQwen3635BA3B,
	modelpreset.ProviderMeta:            modelpreset.PresetMuseSpark13,
	modelpreset.ProviderMiniMax:         modelpreset.PresetMiniMaxM3,
	modelpreset.ProviderMistral:         modelpreset.PresetMistralMedium35,
	modelpreset.ProviderMoonshot:        modelpreset.PresetMoonshotKimiK3,
	modelpreset.ProviderOllama:          modelpreset.PresetQwen3635B,
	modelpreset.ProviderOpenAIChat:      modelpreset.PresetGPT41,
	modelpreset.ProviderOpenAIResponses: modelpreset.PresetGPT56Terra,
	modelpreset.ProviderOpenRouter:      modelpreset.PresetDeepSeekV4Flash,
	modelpreset.ProviderQwen:            modelpreset.PresetQwen38Max,
	modelpreset.ProviderSGLang:          modelpreset.PresetDeepSeekR18B,
	modelpreset.ProviderVLLM:            modelpreset.PresetQwen3VL30BA3B,
	modelpreset.ProviderXAI:             modelpreset.PresetGrok46,
	modelpreset.ProviderXiaomi:          modelpreset.PresetMiMoV25Pro,
	modelpreset.ProviderZAI:             modelpreset.PresetGLM53,
	modelpreset.ProviderZAICodingPlan:   modelpreset.PresetGLM53,
}

var BuiltInDisabledModelPresetIDs = map[inferenceSpec.ProviderName]map[modelpreset.ModelPresetID]struct{}{
	modelpreset.ProviderAnthropic: {
		modelpreset.PresetClaudeOpus45:   {},
		modelpreset.PresetClaudeOpus41:   {},
		modelpreset.PresetClaudeSonnet45: {},
		modelpreset.PresetClaudeSonnet4:  {},
	},
	modelpreset.ProviderGoogleGemini: {
		modelpreset.PresetGemini37Flash: {},
		modelpreset.PresetGemini36Flash: {},
		modelpreset.PresetGemini35Flash: {},

		modelpreset.PresetGemini31Pro:       {},
		modelpreset.PresetGemini31FlashLite: {},

		modelpreset.PresetGemini3Flash:      {},
		modelpreset.PresetGemini25Flash:     {},
		modelpreset.PresetGemini25FlashLite: {},
	},
	modelpreset.ProviderOpenAIChat: {
		modelpreset.PresetGPT41Mini: {},
		modelpreset.PresetGPT4o:     {},
		modelpreset.PresetGPT4oMini: {},
	},
	modelpreset.ProviderOpenAIResponses: {
		modelpreset.PresetGPT54Mini:     {},
		modelpreset.PresetGPT54:         {},
		modelpreset.PresetGPT54Nano:     {},
		modelpreset.PresetGPT53Codex:    {},
		modelpreset.PresetGPT52:         {},
		modelpreset.PresetGPT52Codex:    {},
		modelpreset.PresetGPT51:         {},
		modelpreset.PresetGPT51Codex:    {},
		modelpreset.PresetGPT51CodexMax: {},
		modelpreset.PresetGPT5Mini:      {},
	},
	modelpreset.ProviderXAI: {
		modelpreset.PresetGrok45:             {},
		modelpreset.PresetGrok43:             {},
		modelpreset.PresetBuild01:            {},
		modelpreset.PresetGrok42Reasoning:    {},
		modelpreset.PresetGrok42NonReasoning: {},
	},
}

var BuiltInProviderDefaultHeaderOverlays = map[inferenceSpec.ProviderName]map[string]string{
	modelpreset.ProviderOpenRouter: {
		"HTTP-Referer": "https://github.com/flexigpt/flexigpt-app",
		"X-Title":      "FlexiGPT",
	},
}
