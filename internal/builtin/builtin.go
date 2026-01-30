package builtin

import (
	"embed"
)

//go:embed modelpresets
var BuiltInModelPresetsFS embed.FS

//go:embed tools
var BuiltInToolBundlesFS embed.FS

//go:embed prompts
var BuiltInPromptBundlesFS embed.FS

const (
	BuiltInModelPresetsRootDir = "modelpresets"
	BuiltInModelPresetsJSON    = "modelpresets.json"

	BuiltInToolBundlesRootDir = "tools"
	BuiltInToolBundlesJSON    = "tools.bundles.json"

	BuiltInPromptBundlesRootDir = "prompts"
	BuiltInPromptBundlesJSON    = "prompts.bundles.json"
)

// IMPORTANT: keep these stable (match tools.bundles.json).
const (
	BuiltinBundleIDLLMToolsFS    = "018fe0f4-b8cd-7e55-82d5-9df0bd70e400"
	BuiltinBundleIDLLMToolsImage = "018fe0f4-b8cd-7e55-82d5-9df0bd70e401"
	BuiltinBundleIDLLMToolsShell = "019c0415-40f7-70cb-9200-d804c9388a57"
	BuiltinBundleIDLLMToolsText  = "019c0da2-02d2-7e71-81d7-35f2e5a0bebf"
)

const (
	ProviderNameAnthropic             = "anthropic"
	ProviderNameDeepseek              = "deepseek"
	ProviderNameGoogleGemini          = "googlegemini"
	ProviderNameHuggingFace           = "huggingface"
	ProviderNameLlamaCPP              = "llamacpp"
	ProviderNameOpenAIChatCompletions = "openai"
	ProviderNameOpenAIResponses       = "openairesponses"
	ProviderNameOpenRouter            = "openrouter"
	ProviderNameXAI                   = "xai"
)

var BuiltInProviderNames = []string{
	ProviderNameAnthropic,
	ProviderNameDeepseek,
	ProviderNameGoogleGemini,
	ProviderNameHuggingFace,
	ProviderNameLlamaCPP,
	ProviderNameOpenAIChatCompletions,
	ProviderNameOpenAIResponses,
	ProviderNameOpenRouter,
	ProviderNameXAI,
}
