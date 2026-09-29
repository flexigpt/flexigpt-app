package catalog

import (
	"github.com/flexigpt/inference-go/modelpreset"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

// defaultModelByProvider is application-owned selection policy copied out of
// the legacy ModelPreset store. It is intentionally expressed using
// inference-go catalog identities instead of importing legacy application
// packages.
//
// The generated Provider declaration uses the corresponding generated Model
// Artifact logical name, never the legacy preset slug.
var defaultModelByProvider = map[inferenceSpec.ProviderName]modelpreset.ModelPresetID{
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
	modelpreset.ProviderXAI:             modelpreset.PresetGrok47,
	modelpreset.ProviderXiaomi:          modelpreset.PresetMiMoV26Pro,
	modelpreset.ProviderZAI:             modelpreset.PresetGLM53,
	modelpreset.ProviderZAICodingPlan:   modelpreset.PresetGLM53,
}

// disabledModelsByProvider is generated into Model declaration labels and
// applied only when an Artifact is first created. Later user enablement
// decisions are local Artifact state and are never reset by package refresh.
var disabledModelsByProvider = map[inferenceSpec.ProviderName]map[modelpreset.ModelPresetID]struct{}{
	modelpreset.ProviderAnthropic: {
		modelpreset.PresetClaudeOpus45:   {},
		modelpreset.PresetClaudeOpus41:   {},
		modelpreset.PresetClaudeSonnet45: {},
		modelpreset.PresetClaudeSonnet4:  {},
	},
	modelpreset.ProviderGoogleGemini: {
		modelpreset.PresetGemini37Flash:     {},
		modelpreset.PresetGemini36Flash:     {},
		modelpreset.PresetGemini35Flash:     {},
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

var staticProviderHeaderOverlays = map[inferenceSpec.ProviderName]map[string]string{
	modelpreset.ProviderOpenRouter: {
		"HTTP-Referer": "https://github.com/flexigpt/flexigpt-app",
		"X-Title":      "FlexiGPT",
	},
}
