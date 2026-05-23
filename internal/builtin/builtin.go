package builtin

import (
	"embed"

	"github.com/flexigpt/inference-go/modelpreset"
)

//go:embed tools
var BuiltInToolBundlesFS embed.FS

//go:embed prompts
var BuiltInPromptBundlesFS embed.FS

//go:embed skills
var BuiltInSkillBundlesFS embed.FS

//go:embed assistantpresets
var BuiltInAssistantPresetBundlesFS embed.FS

const (
	BuiltInToolBundlesRootDir = "tools"
	BuiltInToolBundlesJSON    = "tools.bundles.json"

	BuiltInPromptBundlesRootDir = "prompts"
	BuiltInPromptBundlesJSON    = "prompts.bundles.json"

	BuiltInSkillBundlesRootDir = "skills"
	BuiltInSkillBundlesJSON    = "skills.json"

	BuiltInAssistantPresetBundlesRootDir = "assistantpresets"
	BuiltInAssistantPresetBundlesJSON    = "assistantpresets.bundles.json"
)

// IMPORTANT: keep these stable (match tools.bundles.json).
const (
	BuiltinBundleIDLLMToolsFS    = "018fe0f4-b8cd-7e55-82d5-9df0bd70e400"
	BuiltinBundleIDLLMToolsImage = "018fe0f4-b8cd-7e55-82d5-9df0bd70e401"
	BuiltinBundleIDLLMToolsExec  = "019c0415-40f7-70cb-9200-d804c9388a57"
	BuiltinBundleIDLLMToolsText  = "019c0da2-02d2-7e71-81d7-35f2e5a0bebf"
)

const (
	ProviderNameAnthropic             = string(modelpreset.ProviderAnthropic)
	ProviderNameGoogleGemini          = string(modelpreset.ProviderGoogleGemini)
	ProviderNameHuggingFace           = string(modelpreset.ProviderHuggingFace)
	ProviderNameLlamaCPP              = string(modelpreset.ProviderLlamaCPP)
	ProviderNameMistral               = string(modelpreset.ProviderMistral)
	ProviderNameOpenAIChatCompletions = string(modelpreset.ProviderOpenAIChat)
	ProviderNameOpenAIResponses       = string(modelpreset.ProviderOpenAIResponses)
	ProviderNameOpenRouter            = string(modelpreset.ProviderOpenRouter)
	ProviderNameXAI                   = string(modelpreset.ProviderXAI)
)

var BuiltInProviderNames = []string{
	ProviderNameAnthropic,
	ProviderNameGoogleGemini,
	ProviderNameHuggingFace,
	ProviderNameLlamaCPP,
	ProviderNameMistral,
	ProviderNameOpenAIChatCompletions,
	ProviderNameOpenAIResponses,
	ProviderNameOpenRouter,
	ProviderNameXAI,
}
