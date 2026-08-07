package builtin

import "github.com/flexigpt/inference-go/modelpreset"

const (
	ProviderNameAnthropic       = string(modelpreset.ProviderAnthropic)
	ProviderNameLocalAI         = string(modelpreset.ProviderLocalAI)
	ProviderNameLMStudio        = string(modelpreset.ProviderLMStudio)
	ProviderNameGoogleGemini    = string(modelpreset.ProviderGoogleGemini)
	ProviderNameHuggingFace     = string(modelpreset.ProviderHuggingFace)
	ProviderNameLlamaCPP        = string(modelpreset.ProviderLlamaCPP)
	ProviderNameMistral         = string(modelpreset.ProviderMistral)
	ProviderNameOllama          = string(modelpreset.ProviderOllama)
	ProviderNameOpenAIChat      = string(modelpreset.ProviderOpenAIChat)
	ProviderNameOpenAIResponses = string(modelpreset.ProviderOpenAIResponses)
	ProviderNameOpenRouter      = string(modelpreset.ProviderOpenRouter)
	ProviderNameSGLang          = string(modelpreset.ProviderSGLang)
	ProviderNameVLLM            = string(modelpreset.ProviderVLLM)
	ProviderNameXAI             = string(modelpreset.ProviderXAI)
)

var BuiltInProviderNames = []string{
	ProviderNameAnthropic,
	ProviderNameLocalAI,
	ProviderNameLMStudio,
	ProviderNameGoogleGemini,
	ProviderNameHuggingFace,
	ProviderNameLlamaCPP,
	ProviderNameMistral,
	ProviderNameOllama,
	ProviderNameOpenAIChat,
	ProviderNameOpenAIResponses,
	ProviderNameOpenRouter,
	ProviderNameSGLang,
	ProviderNameVLLM,
	ProviderNameXAI,
}
